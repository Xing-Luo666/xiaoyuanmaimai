package handlers

import (
	"database/sql"
	"encoding/json"
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

type createOrderReq struct {
	ProductID string               `json:"productId" binding:"required"`
	Message   string               `json:"message"`
	Price     float64              `json:"price"`
	Quantity  int                  `json:"quantity"`
	Spec      string               `json:"spec"`
	Specs     []createOrderReqSpec `json:"specs"`
}

type createOrderReqSpec struct {
	SpecID   string  `json:"specId"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

func (h *OrderHandler) Create(c *gin.Context) {
	userID := c.GetString("userId")
	username := c.GetString("username")

	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	// 兼容前端 specs 数组格式：从 specs[0] 提取 spec
	if req.Spec == "" && len(req.Specs) > 0 {
		req.Spec = req.Specs[0].SpecID
		if req.Spec == "" {
			req.Spec = req.Specs[0].Name
		}
		if req.Quantity <= 0 {
			req.Quantity = req.Specs[0].Quantity
		}
		if req.Price <= 0 && req.Specs[0].Price > 0 {
			req.Price = req.Specs[0].Price
		}
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	db := h.Store.GetDB()

	// 直接 SELECT 需要的字段，避免 scanProductRow 列数不匹配问题
	var title, sellerID, sellerName, status string
	var price float64
	var productImage sql.NullString
	err := db.QueryRow("SELECT title, seller_id, seller_name, price, status, COALESCE(JSON_UNQUOTE(JSON_EXTRACT(images, '$[0]')), '') FROM products WHERE id = ?", req.ProductID).Scan(&title, &sellerID, &sellerName, &price, &status, &productImage)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "商品不存在"})
		return
	}

	if status != "selling" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "该商品已下架或已售出"})
		return
	}
	if sellerID == userID {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "不能购买自己的商品"})
		return
	}

	finalPrice := price
	if req.Price > 0 {
		finalPrice = req.Price
	}

	now := time.Now()
	order := models.Order{
		ID:           genID("o"),
		ProductID:    req.ProductID,
		ProductTitle: title,
		ProductImage: productImage.String,
		SpecName:     req.Spec,
		Quantity:     req.Quantity,
		BuyerID:      userID,
		BuyerName:    username,
		SellerID:     sellerID,
		SellerName:   sellerName,
		Price:        finalPrice * float64(req.Quantity),
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
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 如果有规格，减少对应规格的库存
	if req.Spec != "" {
		var specsJSON string
		if err := tx.QueryRow("SELECT specs FROM products WHERE id = ? FOR UPDATE", req.ProductID).Scan(&specsJSON); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询商品规格失败"})
			return
		}
		if specsJSON != "" {
			var specs []models.ProductSpec
			json.Unmarshal([]byte(specsJSON), &specs)
			found := false
			for i := range specs {
				if specs[i].Name == req.Spec || specs[i].ID == req.Spec {
					if specs[i].Stock >= 0 && specs[i].Stock < req.Quantity {
						tx.Rollback()
						c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "库存不足"})
						return
					}
					if specs[i].Stock > 0 {
						specs[i].Stock -= req.Quantity
					}
					// stock==-1 表示无限，不减少
					found = true
					break
				}
			}
			if found {
				newSpecsJSON, _ := json.Marshal(specs)
				if _, err := tx.Exec("UPDATE products SET specs = ? WHERE id = ?", string(newSpecsJSON), req.ProductID); err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品规格失败"})
					return
				}

				// 检查是否所有规格库存都为0（排除无限库存 -1）
				allSoldOut := true
				for _, s := range specs {
					if s.Stock == -1 || s.Stock > 0 {
						allSoldOut = false
						break
					}
				}
				if allSoldOut {
					if _, err := tx.Exec("UPDATE products SET status = 'sold_out' WHERE id = ?", req.ProductID); err != nil {
						tx.Rollback()
						c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品状态失败"})
						return
					}
				}
			}
		}
	}

	_, err = tx.Exec(
		"INSERT INTO orders (id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		order.ID, order.ProductID, order.ProductTitle, order.ProductImage, order.SpecName, order.Quantity, order.BuyerID, order.BuyerName, order.SellerID, order.SellerName, order.Price, order.Status, order.Message, order.CreatedAt, order.UpdatedAt,
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
		rows, err = db.Query("SELECT id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, shipped_at, created_at, updated_at FROM orders WHERE buyer_id = ? ORDER BY created_at DESC", userID)
	} else {
		rows, err = db.Query("SELECT id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, shipped_at, created_at, updated_at FROM orders WHERE seller_id = ? ORDER BY created_at DESC", userID)
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
		"shipped":   true,
		"completed": true,
		"cancelled": true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "无效的状态"})
		return
	}

	db := h.Store.GetDB()

	o, err := scanOrderRow(db.QueryRow("SELECT id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, shipped_at, created_at, updated_at FROM orders WHERE id = ?", orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "订单不存在"})
		return
	}

	if req.Status == "accepted" || req.Status == "rejected" || req.Status == "shipped" {
		if o.SellerID != userID {
			c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作"})
			return
		}
	}
	if req.Status == "completed" {
		if o.BuyerID != userID {
			c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作"})
			return
		}
	}
	if req.Status == "cancelled" {
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

	if req.Status == "shipped" {
		_, err = tx.Exec("UPDATE orders SET status = ?, shipped_at = ?, updated_at = ? WHERE id = ?", req.Status, now, now, orderID)
	} else {
		_, err = tx.Exec("UPDATE orders SET status = ?, updated_at = ? WHERE id = ?", req.Status, now, orderID)
	}
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新订单失败"})
		return
	}

	if req.Status == "rejected" || req.Status == "cancelled" {
		// 恢复规格库存 & 若商品是sold_out则改回selling
		if o.SpecName != "" {
			var specsJSON string
			if err := tx.QueryRow("SELECT specs FROM products WHERE id = ? FOR UPDATE", o.ProductID).Scan(&specsJSON); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询商品规格失败"})
				return
			}
			if specsJSON != "" {
				var specs []models.ProductSpec
				json.Unmarshal([]byte(specsJSON), &specs)
				for i := range specs {
					if specs[i].Name == o.SpecName || specs[i].ID == o.SpecName {
						if specs[i].Stock >= 0 {
							specs[i].Stock += o.Quantity
						}
						break
					}
				}
				newSpecsJSON, _ := json.Marshal(specs)
				if _, err := tx.Exec("UPDATE products SET specs = ? WHERE id = ?", string(newSpecsJSON), o.ProductID); err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品规格失败"})
					return
				}
			}
		}
		// 恢复商品为上架状态
		if _, err := tx.Exec("UPDATE products SET status = 'selling' WHERE id = ? AND status IN ('sold_out', 'sold')", o.ProductID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品状态失败"})
			return
		}
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

	o, err := scanOrderRow(db.QueryRow("SELECT id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, shipped_at, created_at, updated_at FROM orders WHERE id = ?", orderID))
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
	var shippedAt sql.NullTime
	err := rows.Scan(&o.ID, &o.ProductID, &o.ProductTitle, &o.ProductImage, &o.SpecName, &o.Quantity,
		&o.BuyerID, &o.BuyerName, &o.SellerID, &o.SellerName, &o.Price, &o.Status, &o.Message, &shippedAt, &o.CreatedAt, &o.UpdatedAt)
	if shippedAt.Valid {
		o.ShippedAt = &shippedAt.Time
	}
	return o, err
}

func scanOrderRow(row *sql.Row) (models.Order, error) {
	var o models.Order
	var shippedAt sql.NullTime
	err := row.Scan(&o.ID, &o.ProductID, &o.ProductTitle, &o.ProductImage, &o.SpecName, &o.Quantity,
		&o.BuyerID, &o.BuyerName, &o.SellerID, &o.SellerName, &o.Price, &o.Status, &o.Message, &shippedAt, &o.CreatedAt, &o.UpdatedAt)
	if shippedAt.Valid {
		o.ShippedAt = &shippedAt.Time
	}
	return o, err
}
