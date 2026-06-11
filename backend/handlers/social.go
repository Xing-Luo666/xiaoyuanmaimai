package handlers

import (
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
	rows, err := db.Query("SELECT id, user_id, product_id, product_title, product_image, spec_name, price, quantity, created_at FROM cart_items WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()
	var items []models.CartItem
	for rows.Next() {
		var it models.CartItem
		rows.Scan(&it.ID, &it.UserID, &it.ProductID, &it.ProductTitle, &it.ProductImage, &it.SpecName, &it.Price, &it.Quantity, &it.CreatedAt)
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

	// 检查相同商品相同规格是否已在购物车
	var existID string
	var existQty int
	err := db.QueryRow("SELECT id, quantity FROM cart_items WHERE user_id = ? AND product_id = ? AND spec_name = ?", userID, req.ProductID, req.SpecName).Scan(&existID, &existQty)
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
	var existID string
	err := db.QueryRow("SELECT id FROM favorites WHERE user_id = ? AND product_id = ?", userID, req.ProductID).Scan(&existID)
	if err == nil {
		if _, err := db.Exec("DELETE FROM favorites WHERE id = ?", existID); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
			return
		}
		db.Exec("UPDATE products SET fav_count = GREATEST(fav_count - 1, 0) WHERE id = ?", req.ProductID)
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已取消收藏", Data: gin.H{"favorited": false}})
		return
	}

	id := genID("fav")
	now := time.Now()
	if _, err := db.Exec("INSERT INTO favorites (id, user_id, product_id, product_title, product_image, price, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, userID, req.ProductID, req.ProductTitle, req.ProductImage, req.Price, now); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "收藏失败"})
		return
	}
	db.Exec("UPDATE products SET fav_count = fav_count + 1 WHERE id = ?", req.ProductID)
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

// ========== 点赞 ==========

func (h *SocialHandler) LikeToggle(c *gin.Context) {
	userID := c.GetString("userId")
	productID := c.Param("id")

	db := h.Store.GetDB()

	// 检查是否已点赞
	var existID string
	err := db.QueryRow("SELECT user_id FROM user_likes WHERE user_id = ? AND product_id = ?", userID, productID).Scan(&existID)
	if err == nil {
		// 已点赞，取消点赞
		db.Exec("DELETE FROM user_likes WHERE user_id = ? AND product_id = ?", userID, productID)
		db.Exec("UPDATE products SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?", productID)
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已取消点赞", Data: gin.H{"liked": false}})
		return
	}

	// 未点赞，添加点赞
	now := time.Now()
	db.Exec("INSERT INTO user_likes (user_id, product_id, created_at) VALUES (?, ?, ?)", userID, productID, now)
	db.Exec("UPDATE products SET like_count = like_count + 1 WHERE id = ?", productID)
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "点赞成功", Data: gin.H{"liked": true}})
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
