package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"school-trade/models"
	"school-trade/store"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SocialHandler struct {
	Store *store.DBStore
}

func NewSocialHandler(s *store.DBStore) *SocialHandler {
	return &SocialHandler{Store: s}
}

// ========== 购物车 ==========

func (h *SocialHandler) CartList(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()
	// LEFT JOIN products 表，获取商品的实时状态，供前端展示售罄标识
	// 注：products 表无 stock 列，库存由 specs JSON 维护，故只取 status
	rows, err := db.Query(`SELECT c.id, c.user_id, c.product_id, c.product_title, c.product_image, c.spec_name, c.price, c.quantity, c.created_at, COALESCE(p.status, '') FROM cart_items c LEFT JOIN products p ON c.product_id = p.id WHERE c.user_id = ? ORDER BY c.created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()
	var items []models.CartItem
	for rows.Next() {
		var it models.CartItem
		rows.Scan(&it.ID, &it.UserID, &it.ProductID, &it.ProductTitle, &it.ProductImage, &it.SpecName, &it.Price, &it.Quantity, &it.CreatedAt, &it.ProductStatus)
		items = append(items, it)
	}
	if items == nil {
		items = []models.CartItem{}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: items})
}

func (h *SocialHandler) CartAdd(c *gin.Context) {
	userID := c.GetString("userId")
	var req struct {
		ProductID    string  `json:"productId" binding:"required"`
		ProductTitle string  `json:"productTitle" binding:"required"`
		ProductImage string  `json:"productImage"`
		SpecName     string  `json:"specName"`
		Price        float64 `json:"price" binding:"required"`
		Quantity     int     `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	db := h.Store.GetDB()
	now := time.Now()

	// 校验商品状态：已售罄/已下架的商品不允许加入购物车
	// 注：products 表无 stock 列，库存由 specs JSON 维护
	var productStatus string
	var specsNS sql.NullString
	err := db.QueryRow("SELECT status, specs FROM products WHERE id = ?", req.ProductID).Scan(&productStatus, &specsNS)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "商品不存在"})
		return
	}
	if productStatus != "selling" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "商品已下架或已售罄，无法加入购物车"})
		return
	}
	productSpecsJSON := ""
	if specsNS.Valid {
		productSpecsJSON = specsNS.String
	}
	// 如果有规格，校验该规格库存
	if req.SpecName != "" && productSpecsJSON != "" && productSpecsJSON != "null" {
		var specs []models.ProductSpec
		if err := json.Unmarshal([]byte(productSpecsJSON), &specs); err == nil {
			specFound := false
			for _, s := range specs {
				if s.Name == req.SpecName || s.ID == req.SpecName {
					specFound = true
					if s.Stock == 0 {
						c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "该规格已售罄"})
						return
					}
					if s.Stock > 0 && s.Stock < req.Quantity {
						c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "库存不足，仅剩 " + fmt.Sprintf("%d", s.Stock) + " 件"})
						return
					}
					break
				}
			}
			if !specFound {
				c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "商品规格不存在"})
				return
			}
		}
	}
	// 无规格商品不做库存预校验（products 表无 stock 列），由下单时的事务校验拦截

	// 检查相同商品相同规格是否已在购物车
	var existID string
	var existQty int
	err = db.QueryRow("SELECT id, quantity FROM cart_items WHERE user_id = ? AND product_id = ? AND spec_name = ?", userID, req.ProductID, req.SpecName).Scan(&existID, &existQty)
	if err == nil {
		db.Exec("UPDATE cart_items SET quantity = ?, created_at = ? WHERE id = ?", existQty+req.Quantity, now, existID)
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已更新购物车数量"})
		return
	}

	id := genID("cart")
	_, err = db.Exec("INSERT INTO cart_items (id, user_id, product_id, product_title, product_image, spec_name, price, quantity, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, userID, req.ProductID, req.ProductTitle, req.ProductImage, req.SpecName, req.Price, req.Quantity, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "添加失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已添加到购物车", Data: gin.H{"id": id}})
}

func (h *SocialHandler) CartUpdate(c *gin.Context) {
	userID := c.GetString("userId")
	itemID := c.Param("id")
	var req struct {
		Quantity int   `json:"quantity"`
		Selected *bool `json:"selected"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	db := h.Store.GetDB()
	if req.Quantity > 0 {
		// 库存校验：与 CartAdd 保持一致
		var productID, specName, productStatus string
		var specsNS sql.NullString
		err := db.QueryRow("SELECT product_id, spec_name, (SELECT status FROM products WHERE id = cart_items.product_id), (SELECT specs FROM products WHERE id = cart_items.product_id) FROM cart_items WHERE id = ? AND user_id = ?", itemID, userID).Scan(&productID, &specName, &productStatus, &specsNS)
		if err != nil {
			c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "购物车项不存在"})
			return
		}
		if productStatus != "" && productStatus != "selling" {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "商品已下架或已售罄"})
			return
		}
		if specName != "" && specsNS.Valid && specsNS.String != "" && specsNS.String != "null" {
			var specs []models.ProductSpec
			if json.Unmarshal([]byte(specsNS.String), &specs) == nil {
				for _, s := range specs {
					if s.Name == specName || s.ID == specName {
						if s.Stock == 0 {
							c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "该规格已售罄"})
							return
						}
						if s.Stock > 0 && s.Stock < req.Quantity {
							c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "库存不足，仅剩 " + fmt.Sprintf("%d", s.Stock) + " 件"})
							return
						}
						break
					}
				}
			}
		}
		db.Exec("UPDATE cart_items SET quantity = ? WHERE id = ? AND user_id = ?", req.Quantity, itemID, userID)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "更新成功"})
}

func (h *SocialHandler) CartDelete(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()

	ids, exists := c.GetQueryArray("ids")
	if exists && len(ids) > 0 {
		placeholders := make([]string, len(ids))
		allArgs := []interface{}{userID}
		for i, id := range ids {
			placeholders[i] = "?"
			allArgs = append(allArgs, id)
		}
		query := "DELETE FROM cart_items WHERE user_id = ? AND id IN (" + strings.Join(placeholders, ",") + ")"
		if _, err := db.Exec(query, allArgs...); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败"})
			return
		}
	} else {
		id := c.Param("id")
		if _, err := db.Exec("DELETE FROM cart_items WHERE id = ? AND user_id = ?", id, userID); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败"})
			return
		}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "删除成功"})
}

// ========== 收藏 ==========

func (h *SocialHandler) FavoriteList(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()
	rows, err := db.Query("SELECT id, user_id, product_id, product_title, product_image, price, created_at FROM favorites WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()
	var list []models.Favorite
	for rows.Next() {
		var f models.Favorite
		rows.Scan(&f.ID, &f.UserID, &f.ProductID, &f.ProductTitle, &f.ProductImage, &f.Price, &f.CreatedAt)
		list = append(list, f)
	}
	if list == nil {
		list = []models.Favorite{}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: list})
}

func (h *SocialHandler) FavoriteToggle(c *gin.Context) {
	userID := c.GetString("userId")
	var req struct {
		ProductID    string  `json:"productId" binding:"required"`
		ProductTitle string  `json:"productTitle"`
		ProductImage string  `json:"productImage"`
		Price        float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	db := h.Store.GetDB()
	// 使用事务保证 favorites 与 products.fav_count 一致
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var existID string
	err = tx.QueryRow("SELECT id FROM favorites WHERE user_id = ? AND product_id = ?", userID, req.ProductID).Scan(&existID)
	if err == nil {
		if _, err := tx.Exec("DELETE FROM favorites WHERE id = ?", existID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
			return
		}
		tx.Exec("UPDATE products SET fav_count = GREATEST(fav_count - 1, 0) WHERE id = ?", req.ProductID)
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
			return
		}
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已取消收藏", Data: gin.H{"favorited": false}})
		return
	}

	id := genID("fav")
	now := time.Now()
	if _, err := tx.Exec("INSERT INTO favorites (id, user_id, product_id, product_title, product_image, price, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, userID, req.ProductID, req.ProductTitle, req.ProductImage, req.Price, now); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "收藏失败"})
		return
	}
	tx.Exec("UPDATE products SET fav_count = fav_count + 1 WHERE id = ?", req.ProductID)
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已收藏", Data: gin.H{"favorited": true}})
}

func (h *SocialHandler) FavoriteCheck(c *gin.Context) {
	userID := c.GetString("userId")
	ids := c.QueryArray("ids")
	result := make(map[string]bool)
	db := h.Store.GetDB()
	for _, pid := range ids {
		var existID string
		err := db.QueryRow("SELECT id FROM favorites WHERE user_id = ? AND product_id = ?", userID, pid).Scan(&existID)
		result[pid] = err == nil
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: result})
}

// ========== 历史记录 ==========

func (h *SocialHandler) HistoryList(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()
	rows, err := db.Query("SELECT id, user_id, product_id, product_title, product_image, price, viewed_at FROM history WHERE user_id = ? ORDER BY viewed_at DESC LIMIT 50", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()
	var list []models.HistoryItem
	for rows.Next() {
		var h models.HistoryItem
		rows.Scan(&h.ID, &h.UserID, &h.ProductID, &h.ProductTitle, &h.ProductImage, &h.Price, &h.ViewedAt)
		list = append(list, h)
	}
	if list == nil {
		list = []models.HistoryItem{}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: list})
}

func (h *SocialHandler) HistoryAdd(c *gin.Context) {
	userID := c.GetString("userId")
	var req struct {
		ProductID    string  `json:"productId" binding:"required"`
		ProductTitle string  `json:"productTitle"`
		ProductImage string  `json:"productImage"`
		Price        float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	db := h.Store.GetDB()
	now := time.Now()

	var existID string
	err := db.QueryRow("SELECT id FROM history WHERE user_id = ? AND product_id = ?", userID, req.ProductID).Scan(&existID)
	if err == nil {
		db.Exec("UPDATE history SET viewed_at = ?, product_title = ?, product_image = ?, price = ? WHERE id = ?", now, req.ProductTitle, req.ProductImage, req.Price, existID)
	} else {
		id := genID("hist")
		db.Exec("INSERT INTO history (id, user_id, product_id, product_title, product_image, price, viewed_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, userID, req.ProductID, req.ProductTitle, req.ProductImage, req.Price, now)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "ok"})
}

// ========== 收货地址 ==========

func (h *SocialHandler) AddressList(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()
	rows, err := db.Query("SELECT id, user_id, phone, campus, building, dorm_number, is_default, created_at, updated_at FROM user_addresses WHERE user_id = ? ORDER BY is_default DESC, updated_at DESC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var list []models.Address
	for rows.Next() {
		var a models.Address
		rows.Scan(&a.ID, &a.UserID, &a.Phone, &a.Campus, &a.Building, &a.DormNumber, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt)
		list = append(list, a)
	}
	if list == nil {
		list = []models.Address{}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: list})
}

func (h *SocialHandler) AddressSave(c *gin.Context) {
	userID := c.GetString("userId")
	var req struct {
		Phone      string `json:"phone"`
		Campus     string `json:"campus"`
		Building   string `json:"building"`
		DormNumber string `json:"dormNumber"`
		IsDefault  bool   `json:"isDefault"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	db := h.Store.GetDB()
	now := time.Now()
	id := genID("addr")

	// 如果设为默认，先取消其他默认
	if req.IsDefault {
		db.Exec("UPDATE user_addresses SET is_default = 0 WHERE user_id = ?", userID)
	}

	isDefaultVal := 0
	if req.IsDefault {
		isDefaultVal = 1
	}

	_, err := db.Exec("INSERT INTO user_addresses (id, user_id, phone, campus, building, dorm_number, is_default, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, userID, req.Phone, req.Campus, req.Building, req.DormNumber, isDefaultVal, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "保存失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "保存成功", Data: gin.H{"id": id}})
}

func (h *SocialHandler) AddressUpdate(c *gin.Context) {
	userID := c.GetString("userId")
	addrID := c.Param("id")

	var req struct {
		Phone      string `json:"phone"`
		Campus     string `json:"campus"`
		Building   string `json:"building"`
		DormNumber string `json:"dormNumber"`
		IsDefault  bool   `json:"isDefault"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	db := h.Store.GetDB()

	// 验证所有权
	var ownerID string
	if err := db.QueryRow("SELECT user_id FROM user_addresses WHERE id = ?", addrID).Scan(&ownerID); err != nil || ownerID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作"})
		return
	}

	if req.IsDefault {
		db.Exec("UPDATE user_addresses SET is_default = 0 WHERE user_id = ?", userID)
	}

	isDefaultVal := 0
	if req.IsDefault {
		isDefaultVal = 1
	}

	_, err := db.Exec("UPDATE user_addresses SET phone=?, campus=?, building=?, dorm_number=?, is_default=?, updated_at=? WHERE id=? AND user_id=?",
		req.Phone, req.Campus, req.Building, req.DormNumber, isDefaultVal, time.Now(), addrID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "更新成功"})
}

func (h *SocialHandler) AddressDelete(c *gin.Context) {
	userID := c.GetString("userId")
	addrID := c.Param("id")

	db := h.Store.GetDB()
	if _, err := db.Exec("DELETE FROM user_addresses WHERE id = ? AND user_id = ?", addrID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "删除成功"})
}
