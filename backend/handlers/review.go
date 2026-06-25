package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"school-trade/models"
	"school-trade/store"
)

type ReviewHandler struct {
	Store *store.DBStore
}

// 评价写入：买家对已完成订单的商品评价
// rating: 1-10 (1=0.5星, 2=1星, ..., 10=5星)
func (h *ReviewHandler) Write(c *gin.Context) {
	userID := c.GetString("userId")
	orderID := c.Param("id")

	var req models.ReviewWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	if req.Rating < 1 || req.Rating > 10 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "评分需在1-10之间（半星step）"})
		return
	}
	if len(req.Content) > 500 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "评价内容不超过500字"})
		return
	}

	db := h.Store.GetDB()

	// 验证订单属于当前用户且已完成
	var buyerID, sellerID, sellerName, status, productID, productTitle, productImage, specName string
	var buyerName sql.NullString
	err := db.QueryRow(`SELECT buyer_id, seller_id, seller_name, status, product_id, product_title, product_image, spec_name, buyer_name FROM orders WHERE id = ?`, orderID).
		Scan(&buyerID, &sellerID, &sellerName, &status, &productID, &productTitle, &productImage, &specName, &buyerName)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "订单不存在"})
		return
	}
	if status != "completed" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "只能评价已完成的订单"})
		return
	}
	if userID != buyerID {
		// 仅买家可评价商品（卖家评价另开接口）
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "仅买家可评价商品"})
		return
	}

	// 检查是否已评价
	var existID string
	if err := db.QueryRow("SELECT id FROM reviews WHERE order_id = ? AND reviewer_id = ?", orderID, userID).Scan(&existID); err == nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "该订单已评价"})
		return
	}

	// 获取评价者昵称与头像
	var reviewerName, reviewerAvatar string
	_ = db.QueryRow("SELECT nickname, COALESCE(avatar, '') FROM users WHERE id = ?", userID).Scan(&reviewerName, &reviewerAvatar)

	imagesJSON, _ := json.Marshal(req.Images)
	if string(imagesJSON) == "null" {
		imagesJSON = []byte("[]")
	}

	id := genID("rev")
	now := time.Now()
	_, err = db.Exec(`INSERT INTO reviews (id, order_id, reviewer_id, reviewer_name, target_id, product_id, product_title, product_image, spec_name, rating, content, images, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, orderID, userID, reviewerName, sellerID, productID, productTitle, productImage, specName, req.Rating, req.Content, string(imagesJSON), now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "评价失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "评价成功", Data: gin.H{"id": id}})
}

// 追评：评价后追加内容（仅一次）
func (h *ReviewHandler) Append(c *gin.Context) {
	userID := c.GetString("userId")
	reviewID := c.Param("id")

	var req models.ReviewAppendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	if len(req.Content) == 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "追评内容不能为空"})
		return
	}
	if len(req.Content) > 500 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "追评内容不超过500字"})
		return
	}

	db := h.Store.GetDB()

	var reviewerID, oldAppend string
	err := db.QueryRow("SELECT reviewer_id, COALESCE(append_content, '') FROM reviews WHERE id = ?", reviewID).Scan(&reviewerID, &oldAppend)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "评价不存在"})
		return
	}
	if reviewerID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权追评"})
		return
	}
	if oldAppend != "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "已追评过，无法再次追评"})
		return
	}

	now := time.Now()
	_, err = db.Exec("UPDATE reviews SET append_content = ?, append_at = ? WHERE id = ?", req.Content, now, reviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "追评失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "追评成功"})
}

// 删除评价：仅本人可删除
func (h *ReviewHandler) Delete(c *gin.Context) {
	userID := c.GetString("userId")
	reviewID := c.Param("id")

	db := h.Store.GetDB()

	var reviewerID string
	err := db.QueryRow("SELECT reviewer_id FROM reviews WHERE id = ?", reviewID).Scan(&reviewerID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "评价不存在"})
		return
	}
	if reviewerID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权删除"})
		return
	}

	_, err = db.Exec("DELETE FROM reviews WHERE id = ?", reviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已删除"})
}

// 商品评价列表（分页）
func (h *ReviewHandler) ProductReviews(c *gin.Context) {
	productID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	db := h.Store.GetDB()

	// 总数
	var total int
	_ = db.QueryRow("SELECT COUNT(*) FROM reviews WHERE product_id = ?", productID).Scan(&total)

	// 列表（含评价者头像）
	rows, err := db.Query(`SELECT r.id, r.order_id, r.reviewer_id, COALESCE(r.reviewer_name, ''), r.target_id, r.product_id,
		r.product_title, r.product_image, r.spec_name, r.rating, r.content, COALESCE(r.append_content, ''), r.append_at,
		COALESCE(r.images, '[]'), r.created_at, COALESCE(u.avatar, '')
		FROM reviews r LEFT JOIN users u ON r.reviewer_id = u.id
		WHERE r.product_id = ? ORDER BY r.created_at DESC LIMIT ? OFFSET ?`,
		productID, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var list []models.Review
	for rows.Next() {
		var r models.Review
		var appendAt sql.NullTime
		var imagesStr sql.NullString
		var reviewerAvatar string
		if err := rows.Scan(&r.ID, &r.OrderID, &r.ReviewerID, &r.ReviewerName, &r.TargetID, &r.ProductID,
			&r.ProductTitle, &r.ProductImage, &r.SpecName, &r.Rating, &r.Content, &r.AppendContent, &appendAt,
			&imagesStr, &r.CreatedAt, &reviewerAvatar); err != nil {
			continue
		}
		if appendAt.Valid {
			t := appendAt.Time
			r.AppendAt = &t
			r.HasAppend = true
		}
		r.ReviewerAvatar = reviewerAvatar
		if imagesStr.Valid && imagesStr.String != "" {
			_ = json.Unmarshal([]byte(imagesStr.String), &r.Images)
		}
		if r.Images == nil {
			r.Images = []string{}
		}
		list = append(list, r)
	}
	if list == nil {
		list = []models.Review{}
	}

	// 评分统计
	var ratingAvg float64
	if total > 0 {
		_ = db.QueryRow("SELECT AVG(rating) FROM reviews WHERE product_id = ?", productID).Scan(&ratingAvg)
		ratingAvg = ratingAvg / 2.0 // 1-10 → 0-5
		// 保留1位小数
		ratingAvg = float64(int(ratingAvg*10)) / 10.0
	} else {
		ratingAvg = 5.0 // 默认5星
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: models.PageData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, Message: fmt.Sprintf("%.1f", ratingAvg)})
}

// 商品评分摘要（用于商品列表/详情页快速展示）
func (h *ReviewHandler) ProductRating(c *gin.Context) {
	productID := c.Param("id")
	db := h.Store.GetDB()

	var total int
	var ratingSum sql.NullFloat64
	_ = db.QueryRow("SELECT COUNT(*), COALESCE(SUM(rating), 0) FROM reviews WHERE product_id = ?", productID).Scan(&total, &ratingSum)

	ratingAvg := 5.0
	if total > 0 && ratingSum.Valid {
		ratingAvg = ratingSum.Float64 / float64(total) / 2.0
		ratingAvg = float64(int(ratingAvg*10)) / 10.0
	}

	// 近30天销量（按订单创建时间，status=completed 或 shipped）
	var sold30d int
	_ = db.QueryRow(`SELECT COALESCE(SUM(quantity), 0) FROM orders WHERE product_id = ? AND status IN ('completed','shipped') AND created_at >= ?`,
		productID, time.Now().AddDate(0, 0, -30)).Scan(&sold30d)

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{
		"ratingAvg":   ratingAvg,
		"ratingCount": total,
		"sold30d":     sold30d,
	}})
}

// 卖家店铺信息（综合评分 = 该卖家所有商品评分均分的平均）
func (h *ReviewHandler) ShopInfo(c *gin.Context) {
	sellerID := c.Param("id")
	db := h.Store.GetDB()

	// 卖家信息
	var sellerName, sellerAvatar string
	err := db.QueryRow("SELECT COALESCE(nickname, ''), COALESCE(avatar, '') FROM users WHERE id = ?", sellerID).Scan(&sellerName, &sellerAvatar)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "卖家不存在"})
		return
	}

	// 该卖家所有商品的评分均分
	var shopRatingSum sql.NullFloat64
	var ratedProductCount int
	_ = db.QueryRow(`SELECT COALESCE(SUM(sub.avg_rating), 0), COUNT(*) FROM (
			SELECT AVG(rating)/2.0 AS avg_rating FROM reviews r
			INNER JOIN products p ON r.product_id = p.id
			WHERE p.seller_id = ? GROUP BY r.product_id
		) sub`, sellerID).Scan(&shopRatingSum, &ratedProductCount)

	shopRating := 5.0
	if ratedProductCount > 0 && shopRatingSum.Valid {
		shopRating = shopRatingSum.Float64 / float64(ratedProductCount)
		shopRating = float64(int(shopRating*10)) / 10.0
	}

	// 评论总数
	var reviewCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM reviews r INNER JOIN products p ON r.product_id = p.id WHERE p.seller_id = ?`, sellerID).Scan(&reviewCount)

	// 在售商品数
	var productCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM products WHERE seller_id = ? AND status = 'selling'", sellerID).Scan(&productCount)

	// 店铺近30天销量
	var sold30d int
	_ = db.QueryRow(`SELECT COALESCE(SUM(o.quantity), 0) FROM orders o INNER JOIN products p ON o.product_id = p.id
		WHERE p.seller_id = ? AND o.status IN ('completed','shipped') AND o.created_at >= ?`,
		sellerID, time.Now().AddDate(0, 0, -30)).Scan(&sold30d)

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: models.ShopInfo{
		SellerID:     sellerID,
		SellerName:   sellerName,
		SellerAvatar: sellerAvatar,
		ShopRating:   shopRating,
		ReviewCount:  reviewCount,
		ProductCount: productCount,
		Sold30d:      sold30d,
	}})
}

// 旧接口兼容：用户评价列表
func (h *ReviewHandler) UserReviews(c *gin.Context) {
	userID := c.Param("userId")
	db := h.Store.GetDB()

	rows, err := db.Query(`SELECT r.id, r.order_id, r.reviewer_id, COALESCE(r.reviewer_name, ''), r.target_id, r.product_id,
		r.product_title, r.product_image, r.spec_name, r.rating, r.content, COALESCE(r.append_content, ''), r.append_at,
		COALESCE(r.images, '[]'), r.created_at, COALESCE(u.avatar, '')
		FROM reviews r LEFT JOIN users u ON r.reviewer_id = u.id
		WHERE r.target_id = ? ORDER BY r.created_at DESC LIMIT 50`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var list []models.Review
	for rows.Next() {
		var r models.Review
		var appendAt sql.NullTime
		var imagesStr sql.NullString
		var reviewerAvatar string
		_ = rows.Scan(&r.ID, &r.OrderID, &r.ReviewerID, &r.ReviewerName, &r.TargetID, &r.ProductID,
			&r.ProductTitle, &r.ProductImage, &r.SpecName, &r.Rating, &r.Content, &r.AppendContent, &appendAt,
			&imagesStr, &r.CreatedAt, &reviewerAvatar)
		if appendAt.Valid {
			t := appendAt.Time
			r.AppendAt = &t
			r.HasAppend = true
		}
		r.ReviewerAvatar = reviewerAvatar
		if imagesStr.Valid && imagesStr.String != "" {
			_ = json.Unmarshal([]byte(imagesStr.String), &r.Images)
		}
		if r.Images == nil {
			r.Images = []string{}
		}
		list = append(list, r)
	}
	if list == nil {
		list = []models.Review{}
	}

	totalCount := len(list)
	goodCount := 0
	for _, r := range list {
		if r.Rating >= 6 { // 半星制：6=3星以上算好评
			goodCount++
		}
	}
	goodRate := 0.0
	if totalCount > 0 {
		goodRate = float64(goodCount) / float64(totalCount) * 100
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{
		"reviews":    list,
		"totalCount": totalCount,
		"goodRate":   goodRate,
	}})
}

// 旧接口兼容：订单是否已评价
func (h *ReviewHandler) OrderReviewed(c *gin.Context) {
	userID := c.GetString("userId")
	orderID := c.Param("id")

	db := h.Store.GetDB()
	var existID string
	err := db.QueryRow("SELECT id FROM reviews WHERE order_id = ? AND reviewer_id = ?", orderID, userID).Scan(&existID)
	reviewed := err == nil

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{"reviewed": reviewed}})
}
