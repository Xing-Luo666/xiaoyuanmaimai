package handlers

import (
	"database/sql"
	"net/http"
	"school-trade/models"
	"school-trade/store"
	"time"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	Store *store.DBStore
}

func NewOrderHandler(s *store.DBStore) *OrderHandler {
	return &OrderHandler{Store: s}
}

func (h *OrderHandler) Create(c *gin.Context) {
	userID := c.GetString("userId")
	username := c.GetString("username")

	var req struct {
		ProductID string `json:"productId" binding:"required"`
		Message   string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	db := h.Store.GetDB()

	p, err := scanProductRow(db.QueryRow("SELECT id, title, seller_id, seller_name, price, status FROM products WHERE id = ?", req.ProductID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "商品不存在"})
		return
	}

	if p.Status != "selling" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "该商品已下架或已售出"})
		return
	}
	if p.SellerID == userID {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "不能购买自己的商品"})
		return
	}

	now := time.Now()
	order := models.Order{
		ID:           genID("o"),
		ProductID:    p.ID,
		ProductTitle: p.Title,
		BuyerID:      userID,
		BuyerName:    username,
		SellerID:     p.SellerID,
		SellerName:   p.SellerName,
		Price:        p.Price,
		Status:       "pending",
		Message:      req.Message,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "下单失败"})
		return
	}

	_, err = tx.Exec("UPDATE products SET status = 'reserved' WHERE id = ?", req.ProductID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品状态失败"})
		return
	}

	_, err = tx.Exec(
		"INSERT INTO orders (id, product_id, product_title, buyer_id, buyer_name, seller_id, seller_name, price, status, message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		order.ID, order.ProductID, order.ProductTitle, order.BuyerID, order.BuyerName, order.SellerID, order.SellerName, order.Price, order.Status, order.Message, order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "创建订单失败"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "下单失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "下单成功", Data: order})
}

func (h *OrderHandler) MyOrders(c *gin.Context) {
	userID := c.GetString("userId")
	role := c.DefaultQuery("role", "buyer")

	db := h.Store.GetDB()

	var err error
	var rows *sql.Rows

	if role == "buyer" {
		rows, err = db.Query("SELECT id, product_id, product_title, buyer_id, buyer_name, seller_id, seller_name, price, status, message, created_at, updated_at FROM orders WHERE buyer_id = ? ORDER BY created_at DESC", userID)
	} else {
		rows, err = db.Query("SELECT id, product_id, product_title, buyer_id, buyer_name, seller_id, seller_name, price, status, message, created_at, updated_at FROM orders WHERE seller_id = ? ORDER BY created_at DESC", userID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	orders := []models.Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			continue
		}
		orders = append(orders, o)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "成功",
		Data:    models.PageData{List: orders, Total: len(orders), Page: 1, PageSize: 100},
	})
}

func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	userID := c.GetString("userId")
	orderID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	validStatuses := map[string]bool{
		"accepted":  true,
		"rejected":  true,
		"completed": true,
		"cancelled": true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "无效的状态"})
		return
	}

	db := h.Store.GetDB()

	o, err := scanOrderRow(db.QueryRow("SELECT id, product_id, product_title, buyer_id, buyer_name, seller_id, seller_name, price, status, message, created_at, updated_at FROM orders WHERE id = ?", orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "订单不存在"})
		return
	}

	if req.Status == "accepted" || req.Status == "rejected" {
		if o.SellerID != userID {
			c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作"})
			return
		}
	}
	if req.Status == "completed" || req.Status == "cancelled" {
		if o.BuyerID != userID && o.SellerID != userID {
			c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作"})
			return
		}
	}

	now := time.Now()

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
		return
	}

	_, err = tx.Exec("UPDATE orders SET status = ?, updated_at = ? WHERE id = ?", req.Status, now, orderID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新订单失败"})
		return
	}

	if req.Status == "rejected" || req.Status == "cancelled" {
		_, err = tx.Exec("UPDATE products SET status = 'selling' WHERE id = ?", o.ProductID)
	} else if req.Status == "completed" {
		_, err = tx.Exec("UPDATE products SET status = 'sold' WHERE id = ?", o.ProductID)
	}
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品状态失败"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
		return
	}

	o.Status = req.Status
	o.UpdatedAt = now
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "操作成功", Data: o})
}

func (h *OrderHandler) Get(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetString("userId")

	db := h.Store.GetDB()

	o, err := scanOrderRow(db.QueryRow("SELECT id, product_id, product_title, buyer_id, buyer_name, seller_id, seller_name, price, status, message, created_at, updated_at FROM orders WHERE id = ?", orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "订单不存在"})
		return
	}

	if o.BuyerID != userID && o.SellerID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权查看"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "成功", Data: o})
}

func scanOrder(rows *sql.Rows) (models.Order, error) {
	var o models.Order
	err := rows.Scan(&o.ID, &o.ProductID, &o.ProductTitle, &o.BuyerID, &o.BuyerName,
		&o.SellerID, &o.SellerName, &o.Price, &o.Status, &o.Message, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

func scanOrderRow(row *sql.Row) (models.Order, error) {
	var o models.Order
	err := row.Scan(&o.ID, &o.ProductID, &o.ProductTitle, &o.BuyerID, &o.BuyerName,
		&o.SellerID, &o.SellerName, &o.Price, &o.Status, &o.Message, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}
