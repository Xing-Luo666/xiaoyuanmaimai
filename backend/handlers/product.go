package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"school-trade/models"
	"school-trade/store"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	Store *store.DBStore
}

func NewProductHandler(s *store.DBStore) *ProductHandler {
	return &ProductHandler{Store: s}
}

func (h *ProductHandler) List(c *gin.Context) {
	db := h.Store.GetDB()

	keyword := c.Query("keyword")
	category := c.Query("category")
	campus := c.Query("campus")
	sellerID := c.Query("sellerId")
	condition := c.Query("condition")
	hasImage := c.Query("hasImage")
	priceMin := c.Query("priceMin")
	priceMax := c.Query("priceMax")
	sortBy := c.DefaultQuery("sortBy", "newest")
	status := c.DefaultQuery("status", "selling")

	query := "SELECT id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at FROM products WHERE 1=1"
	args := []interface{}{}

	if status != "all" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if keyword != "" {
		query += " AND (title LIKE ? OR description LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}
	if category != "" && category != "all" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if campus != "" && campus != "all" {
		query += " AND campus = ?"
		args = append(args, campus)
	}
	if sellerID != "" {
		query += " AND seller_id = ?"
		args = append(args, sellerID)
	}
	if condition != "" && condition != "all" {
		query += " AND cond = ?"
		args = append(args, condition)
	}
	if hasImage == "1" {
		query += " AND images NOT LIKE '%default-product.gif%' AND images != '[]' AND images != ''"
	}
	if priceMin != "" {
		if v, err := strconv.ParseFloat(priceMin, 64); err == nil {
			query += " AND price >= ?"
			args = append(args, v)
		}
	}
	if priceMax != "" {
		if v, err := strconv.ParseFloat(priceMax, 64); err == nil {
			query += " AND price <= ?"
			args = append(args, v)
		}
	}

	switch sortBy {
	case "price_asc":
		query += " ORDER BY price ASC"
	case "price_desc":
		query += " ORDER BY price DESC"
	case "popular":
		query += " ORDER BY view_count DESC"
	default:
		query += " ORDER BY created_at DESC"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			continue
		}
		// 附加评分与30天销量，让前端可显示
		attachRatingAndSold(db, &p)
		products = append(products, p)
	}

	// 评分/销量排序需在内存中处理（聚合字段不在 products 表）
	switch sortBy {
	case "rating_desc":
		sort.SliceStable(products, func(i, j int) bool {
			if products[i].RatingAvg != products[j].RatingAvg {
				return products[i].RatingAvg > products[j].RatingAvg
			}
			return products[i].RatingCount > products[j].RatingCount
		})
	case "sold_desc":
		sort.SliceStable(products, func(i, j int) bool {
			return products[i].Sold30d > products[j].Sold30d
		})
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "成功",
		Data: models.PageData{
			List:     products,
			Total:    len(products),
			Page:     1,
			PageSize: 20,
		},
	})
}

func (h *ProductHandler) Get(c *gin.Context) {
	id := c.Param("id")
	db := h.Store.GetDB()

	p, err := scanProductRow(db.QueryRow("SELECT id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at FROM products WHERE id = ?", id))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "商品不存在"})
		return
	}

	if _, err := db.Exec("UPDATE products SET view_count = view_count + 1 WHERE id = ?", id); err == nil {
		p.ViewCount++
	}

	// 附加评分与30天销量（与列表接口保持一致）
	attachRatingAndSold(db, &p)

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "成功", Data: p})
}

func (h *ProductHandler) Create(c *gin.Context) {
	userID := c.GetString("userId")
	username := c.GetString("username")

	var req struct {
		Title       string               `json:"title" binding:"required"`
		Description string               `json:"description"`
		Category    string               `json:"category" binding:"required"`
		Price       float64              `json:"price" binding:"required"`
		OriPrice    float64              `json:"oriPrice"`
		Images      []string             `json:"images"`
		Specs       []models.ProductSpec `json:"specs"`
		Condition   string               `json:"condition"`
		Campus      string               `json:"campus"`
		Building    string               `json:"building"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	if len(req.Specs) == 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "请至少添加一个商品规格"})
		return
	}

	now := time.Now()
	if len(req.Images) == 0 {
		req.Images = []string{"/resources/default-product.gif"}
	}
	product := models.Product{
		ID:          genID("p"),
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Price:       req.Price,
		OriPrice:    req.OriPrice,
		Images:      req.Images,
		Specs:       req.Specs,
		Condition:   req.Condition,
		Campus:      req.Campus,
		Building:    req.Building,
		SellerID:    userID,
		SellerName:  username,
		Status:      "selling",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	imagesJSON, _ := json.Marshal(product.Images)
	specsJSON, _ := json.Marshal(product.Specs)

	db := h.Store.GetDB()
	_, err := db.Exec(
		"INSERT INTO products (id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		product.ID, product.Title, product.Description, product.Category, product.Price, product.OriPrice, string(imagesJSON), string(specsJSON),
		product.Condition, product.Campus, product.Building, product.SellerID, product.SellerName, product.Status, product.ViewCount, product.LikeCount, product.FavCount, product.CreatedAt, product.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "发布失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "发布成功", Data: product})
}

func (h *ProductHandler) Update(c *gin.Context) {
	userID := c.GetString("userId")
	id := c.Param("id")

	var req struct {
		Title       string               `json:"title"`
		Description string               `json:"description"`
		Category    string               `json:"category"`
		Price       float64              `json:"price"`
		OriPrice    float64              `json:"oriPrice"`
		Images      []string             `json:"images"`
		Specs       []models.ProductSpec `json:"specs"`
		Condition   string               `json:"condition"`
		Campus      string               `json:"campus"`
		Building    string               `json:"building"`
		Status      string               `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	db := h.Store.GetDB()

	p, err := scanProductRow(db.QueryRow("SELECT id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at FROM products WHERE id = ?", id))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "商品不存在"})
		return
	}

	if p.SellerID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作此商品"})
		return
	}

	now := time.Now()
	if req.Title != "" {
		p.Title = req.Title
	}
	if req.Description != "" {
		p.Description = req.Description
	}
	if req.Category != "" {
		p.Category = req.Category
	}
	if req.Price > 0 {
		p.Price = req.Price
	}
	if req.OriPrice > 0 {
		p.OriPrice = req.OriPrice
	}
	if req.Images != nil {
		p.Images = req.Images
	}
	if req.Specs != nil {
		p.Specs = req.Specs
	}
	if req.Condition != "" {
		p.Condition = req.Condition
	}
	if req.Campus != "" {
		p.Campus = req.Campus
	}
	if req.Building != "" {
		p.Building = req.Building
	}
	if req.Status != "" {
		validStatuses := map[string]bool{"selling": true, "sold_out": true, "reserved": true, "hidden": true}
		if !validStatuses[req.Status] {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "无效的商品状态"})
			return
		}
		p.Status = req.Status
	}
	p.UpdatedAt = now

	imagesJSON, _ := json.Marshal(p.Images)
	specsJSON, _ := json.Marshal(p.Specs)
	_, err = db.Exec(
		"UPDATE products SET title=?, description=?, category=?, price=?, ori_price=?, images=?, specs=?, cond=?, campus=?, building=?, status=?, updated_at=? WHERE id=?",
		p.Title, p.Description, p.Category, p.Price, p.OriPrice, string(imagesJSON), string(specsJSON), p.Condition, p.Campus, p.Building, p.Status, p.UpdatedAt, p.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "更新成功", Data: p})
}

func (h *ProductHandler) Delete(c *gin.Context) {
	userID := c.GetString("userId")
	id := c.Param("id")

	db := h.Store.GetDB()

	p, err := scanProductRow(db.QueryRow("SELECT id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at FROM products WHERE id = ?", id))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "商品不存在"})
		return
	}
	if p.SellerID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作此商品"})
		return
	}

	// 删除商品及其关联数据（事务保证一致性）
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if _, err := tx.Exec("DELETE FROM favorites WHERE product_id = ?", id); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除收藏失败"})
		return
	}
	if _, err := tx.Exec("DELETE FROM cart_items WHERE product_id = ?", id); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除购物车项失败"})
		return
	}
	if _, err := tx.Exec("DELETE FROM history WHERE product_id = ?", id); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除浏览历史失败"})
		return
	}
	if _, err := tx.Exec("DELETE FROM reviews WHERE product_id = ?", id); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除评价失败"})
		return
	}
	if _, err := tx.Exec("DELETE FROM products WHERE id = ?", id); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除商品失败"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "删除成功"})
}

func (h *ProductHandler) MyProducts(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()

	rows, err := db.Query("SELECT id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at FROM products WHERE seller_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			continue
		}
		products = append(products, p)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "成功",
		Data:    models.PageData{List: products, Total: len(products), Page: 1, PageSize: 100},
	})
}

// ShopProducts 查看某卖家的店铺所有在售商品（公开接口，需要登录）
func (h *ProductHandler) ShopProducts(c *gin.Context) {
	sellerID := c.Param("id")
	db := h.Store.GetDB()

	rows, err := db.Query("SELECT id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at FROM products WHERE seller_id = ? AND status = 'selling' ORDER BY created_at DESC", sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			continue
		}
		// 附加评分与销量
		attachRatingAndSold(db, &p)
		products = append(products, p)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "成功",
		Data:    models.PageData{List: products, Total: len(products), Page: 1, PageSize: 100},
	})
}

// attachRatingAndSold 给商品附加评分与30天销量
func attachRatingAndSold(db *sql.DB, p *models.Product) {
	var total int
	var ratingSum sql.NullFloat64
	_ = db.QueryRow("SELECT COUNT(*), COALESCE(SUM(rating), 0) FROM reviews WHERE product_id = ?", p.ID).Scan(&total, &ratingSum)
	if total > 0 && ratingSum.Valid {
		avg := ratingSum.Float64 / float64(total) / 2.0
		p.RatingAvg = float64(int(avg*10)) / 10.0
		p.RatingCount = total
	} else {
		p.RatingAvg = 5.0
		p.RatingCount = 0
	}
	var sold30d int
	_ = db.QueryRow("SELECT COALESCE(SUM(quantity), 0) FROM orders WHERE product_id = ? AND status IN ('completed','shipped') AND created_at >= ?",
		p.ID, time.Now().AddDate(0, 0, -30)).Scan(&sold30d)
	p.Sold30d = sold30d
}

var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".bmp": true, ".heic": true, ".heif": true,
}

// UploadImage 上传商品图片
func (h *ProductHandler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "请选择图片文件"})
		return
	}
	defer file.Close()

	// 检查文件大小（10MB）
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "图片大小不能超过 10MB"})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExts[ext] {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "不支持的图片格式，仅支持 jpg/png/gif/webp/bmp/heic"})
		return
	}

	// 保存在 resources 目录
	// 兼容从 backend/ 或项目根目录启动两种情况
	execDir, _ := os.Getwd()
	uploadDir := filepath.Join(execDir, "..", "frontend", "resources")
	if _, err := os.Stat(filepath.Join(execDir, "frontend")); err == nil {
		uploadDir = filepath.Join(execDir, "frontend", "resources")
	}
	os.MkdirAll(uploadDir, 0755)

	// 生成唯一文件名
	fileName := fmt.Sprintf("upload_%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(uploadDir, fileName)

	dst, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "保存图片失败"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "写入图片失败"})
		return
	}

	// 返回可访问的 URL
	imageURL := "/resources/" + fileName
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "上传成功", Data: gin.H{"url": imageURL}})
}

func scanProduct(rows *sql.Rows) (models.Product, error) {
	var p models.Product
	var imagesStr, specsStr sql.NullString
	var building sql.NullString
	var favCount sql.NullInt64

	// 动态获取列
	cols, _ := rows.Columns()
	numCols := len(cols)
	if numCols >= 19 {
		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Category, &p.Price, &p.OriPrice, &imagesStr, &specsStr,
			&p.Condition, &p.Campus, &building, &p.SellerID, &p.SellerName, &p.Status, &p.ViewCount, &p.LikeCount, &favCount, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return p, err
		}
	} else {
		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Category, &p.Price, &p.OriPrice, &imagesStr,
			&p.Condition, &p.Campus, &p.SellerID, &p.SellerName, &p.Status, &p.ViewCount, &p.LikeCount, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return p, err
		}
	}
	p.Building = building.String
	p.FavCount = int(favCount.Int64)
	if imagesStr.Valid && imagesStr.String != "" {
		json.Unmarshal([]byte(imagesStr.String), &p.Images)
	}
	if p.Images == nil {
		p.Images = []string{}
	}
	if specsStr.Valid && specsStr.String != "" {
		json.Unmarshal([]byte(specsStr.String), &p.Specs)
	}
	if p.Specs == nil {
		p.Specs = []models.ProductSpec{}
	}
	return p, nil
}

func scanProductRow(row *sql.Row) (models.Product, error) {
	// 使用 QueryRow 无法获取列数，先尝试 19 列新表，失败则回退到 16 列
	var p models.Product
	var imagesStr, specsStr sql.NullString
	var building sql.NullString
	var favCount sql.NullInt64
	err := row.Scan(&p.ID, &p.Title, &p.Description, &p.Category, &p.Price, &p.OriPrice, &imagesStr, &specsStr,
		&p.Condition, &p.Campus, &building, &p.SellerID, &p.SellerName, &p.Status, &p.ViewCount, &p.LikeCount, &favCount, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, err
	}
	p.Building = building.String
	p.FavCount = int(favCount.Int64)
	if imagesStr.Valid {
		json.Unmarshal([]byte(imagesStr.String), &p.Images)
	}
	if p.Images == nil {
		p.Images = []string{}
	}
	if specsStr.Valid {
		json.Unmarshal([]byte(specsStr.String), &p.Specs)
	}
	if p.Specs == nil {
		p.Specs = []models.ProductSpec{}
	}
	return p, nil
}

// Recommend 基于用户行为（浏览/购买/购物车/收藏）推荐商品
// 算法：按行为权重给分类打分，优先推荐高分分类的在售商品；
// 支持 seed 参数实现"换一批"刷新功能
func (h *ProductHandler) Recommend(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()

	// 行为权重
	const (
		wBrowse = 1 // 浏览历史
		wFav    = 2 // 收藏
		wCart   = 3 // 购物车
		wBuy    = 5 // 购买
	)

	// 收集用户各分类的行为得分
	catScore := map[string]int{}

	// 1. 浏览历史
	rows, _ := db.Query(`SELECT p.category, COUNT(*) FROM history h JOIN products p ON h.product_id = p.id WHERE h.user_id = ? GROUP BY p.category`, userID)
	if rows != nil {
		for rows.Next() {
			var cat string
			var cnt int
			rows.Scan(&cat, &cnt)
			catScore[cat] += cnt * wBrowse
		}
		rows.Close()
	}

	// 2. 收藏
	rows, _ = db.Query(`SELECT p.category, COUNT(*) FROM favorites f JOIN products p ON f.product_id = p.id WHERE f.user_id = ? GROUP BY p.category`, userID)
	if rows != nil {
		for rows.Next() {
			var cat string
			var cnt int
			rows.Scan(&cat, &cnt)
			catScore[cat] += cnt * wFav
		}
		rows.Close()
	}

	// 3. 购物车
	rows, _ = db.Query(`SELECT p.category, COUNT(*) FROM cart_items ci JOIN products p ON ci.product_id = p.id WHERE ci.user_id = ? GROUP BY p.category`, userID)
	if rows != nil {
		for rows.Next() {
			var cat string
			var cnt int
			rows.Scan(&cat, &cnt)
			catScore[cat] += cnt * wCart
		}
		rows.Close()
	}

	// 4. 购买记录
	rows, _ = db.Query(`SELECT p.category, COUNT(*) FROM orders o JOIN products p ON o.product_id = p.id WHERE o.buyer_id = ? GROUP BY p.category`, userID)
	if rows != nil {
		for rows.Next() {
			var cat string
			var cnt int
			rows.Scan(&cat, &cnt)
			catScore[cat] += cnt * wBuy
		}
		rows.Close()
	}

	// 构建分类排序（得分降序）
	type catRank struct {
		Cat   string
		Score int
	}
	var ranks []catRank
	for cat, sc := range catScore {
		ranks = append(ranks, catRank{cat, sc})
	}
	sort.SliceStable(ranks, func(i, j int) bool { return ranks[i].Score > ranks[j].Score })

	// seed 参数用于刷新（换一批），每次刷新改变随机偏移
	seedStr := c.DefaultQuery("seed", "0")
	seed, _ := strconv.Atoi(seedStr)
	if seed < 0 {
		seed = 0
	}

	// 排除用户自己的商品，只推荐在售商品
	baseQuery := `SELECT id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at FROM products WHERE status = 'selling' AND seller_id != ?`
	baseArgs := []interface{}{userID}

	var products []models.Product
	seenIDs := map[string]bool{}

	// 用 seed 创建确定性随机数生成器，实现"换一批"
	rng := rand.New(rand.NewSource(int64(seed)))

	if len(ranks) > 0 {
		// 有行为数据：按分类偏好加权查询
		// 取得分最高的前3个分类
		topCats := ranks
		if len(topCats) > 3 {
			topCats = topCats[:3]
		}

		// 查询每个偏好分类的所有在售商品，在应用层用 seed 打乱顺序
		for idx, r := range topCats {
			q := baseQuery + " AND category = ? ORDER BY view_count DESC, created_at DESC"
			args := append(baseArgs, r.Cat)
			rows, err := db.Query(q, args...)
			if err != nil {
				continue
			}
			var catProducts []models.Product
			for rows.Next() {
				p, err := scanProduct(rows)
				if err != nil {
					continue
				}
				if !seenIDs[p.ID] {
					attachRatingAndSold(db, &p)
					catProducts = append(catProducts, p)
				}
			}
			rows.Close()
			// 用 seed+idx 打乱该分类的商品顺序
			catRng := rand.New(rand.NewSource(int64(seed*100 + idx)))
			catRng.Shuffle(len(catProducts), func(i, j int) { catProducts[i], catProducts[j] = catProducts[j], catProducts[i] })
			// 每个偏好分类最多取 3 个
			take := 3
			if take > len(catProducts) {
				take = len(catProducts)
			}
			for k := 0; k < take; k++ {
				products = append(products, catProducts[k])
				seenIDs[catProducts[k].ID] = true
			}
		}

		// 从所有在售商品中补足到 8 个
		if len(products) < 8 {
			// 查询所有未推荐过的在售商品
			excludeIDs := ""
			excludeArgs := []interface{}{}
			for id := range seenIDs {
				if excludeIDs != "" {
					excludeIDs += ","
				}
				excludeIDs += "?"
				excludeArgs = append(excludeArgs, id)
			}
			fillQuery := baseQuery
			fillArgs := append([]interface{}{userID}, excludeArgs...)
			if excludeIDs != "" {
				fillQuery += " AND id NOT IN (" + excludeIDs + ")"
			}
			fillQuery += " ORDER BY view_count DESC, created_at DESC"
			rows, err := db.Query(fillQuery, fillArgs...)
			if err == nil {
				var fillProducts []models.Product
				for rows.Next() {
					p, err := scanProduct(rows)
					if err != nil {
						continue
					}
					attachRatingAndSold(db, &p)
					fillProducts = append(fillProducts, p)
				}
				rows.Close()
				// 用 seed 打乱补足商品
				rng.Shuffle(len(fillProducts), func(i, j int) { fillProducts[i], fillProducts[j] = fillProducts[j], fillProducts[i] })
				need := 8 - len(products)
				if need > len(fillProducts) {
					need = len(fillProducts)
				}
				for k := 0; k < need; k++ {
					products = append(products, fillProducts[k])
				}
			}
		}
	}

	// 无行为数据或推荐结果为空：从所有在售商品随机取 8 个
	if len(products) == 0 {
		q := baseQuery + " ORDER BY view_count DESC, created_at DESC"
		rows, err := db.Query(q, baseArgs)
		if err == nil {
			var allProducts []models.Product
			for rows.Next() {
				p, err := scanProduct(rows)
				if err != nil {
					continue
				}
				attachRatingAndSold(db, &p)
				allProducts = append(allProducts, p)
			}
			rows.Close()
			rng.Shuffle(len(allProducts), func(i, j int) { allProducts[i], allProducts[j] = allProducts[j], allProducts[i] })
			take := 8
			if take > len(allProducts) {
				take = len(allProducts)
			}
			products = allProducts[:take]
		}
	}

	// 最后再打乱一次顺序，让偏好分类商品不总是排在前面
	rng.Shuffle(len(products), func(i, j int) { products[i], products[j] = products[j], products[i] })

	if products == nil {
		products = []models.Product{}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "成功",
		Data: models.PageData{
			List:     products,
			Total:    len(products),
			Page:     1,
			PageSize: 8,
		},
	})
}
