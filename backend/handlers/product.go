package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"school-trade/models"
	"school-trade/store"
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
		products = append(products, p)
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

	db.Exec("UPDATE products SET view_count = view_count + 1 WHERE id = ?", id)
	p.ViewCount++

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
		"INSERT INTO products (id, title, description, category, price, ori_price, images, specs, cond, campus, building, seller_id, seller_name, status, view_count, like_count, fav_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
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

	db.Exec("DELETE FROM products WHERE id = ?", id)
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
	uploadDir := filepath.Join("..", "frontend", "resources")
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
