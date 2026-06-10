package handlers

import (
	"encoding/json"
	"net/http"
	"school-trade/models"
	"school-trade/store"
	"time"

	"github.com/gin-gonic/gin"
)

type BannerHandler struct {
	Store *store.DBStore
}

func NewBannerHandler(s *store.DBStore) *BannerHandler {
	return &BannerHandler{Store: s}
}

type Banner struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	ImageURL  string    `json:"imageUrl"`
	LinkURL   string    `json:"linkUrl"`
	SortOrder int       `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// 默认轮播卡片（无自定义图片时使用）
var defaultBanners = []Banner{
	{ID: "b-default-1", Title: "🎓 校园闲置交易集市", Subtitle: "学长学姐好物转让 · 品质有保障 · 校内面交更安心", SortOrder: 0},
	{ID: "b-default-2", Title: "📱 数码好物 学生价", Subtitle: "iPhone · iPad · 耳机 · 相机 — 应有尽有", SortOrder: 1},
	{ID: "b-default-3", Title: "📖 二手书籍 白菜价", Subtitle: "教材教辅 · 考研资料 · 课外读物", SortOrder: 2},
	{ID: "b-default-4", Title: "🏠 寝室好物 大甩卖", Subtitle: "台灯 · 收纳 · 小电器 · 生活用品", SortOrder: 3},
}

// GetBanners 公开接口 — 返回轮播卡片列表
func (h *BannerHandler) GetBanners(c *gin.Context) {
	db := h.Store.GetDB()
	rows, err := db.Query("SELECT id, title, subtitle, image_url, link_url, sort_order, created_at, updated_at FROM banners ORDER BY sort_order ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var list []Banner
	for rows.Next() {
		var b Banner
		rows.Scan(&b.ID, &b.Title, &b.Subtitle, &b.ImageURL, &b.LinkURL, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt)
		list = append(list, b)
	}

	// 数据库为空时返回默认卡片
	if len(list) == 0 {
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: defaultBanners})
		return
	}

	if list == nil {
		list = []Banner{}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: list})
}

// AdminListBanners 管理员 — 列表
func (h *BannerHandler) AdminListBanners(c *gin.Context) {
	h.GetBanners(c)
}

// AdminCreateBanner 管理员 — 新增
func (h *BannerHandler) AdminCreateBanner(c *gin.Context) {
	var req Banner
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	db := h.Store.GetDB()
	now := time.Now()
	id := genID("b")
	_, err := db.Exec("INSERT INTO banners (id, title, subtitle, image_url, link_url, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, req.Title, req.Subtitle, req.ImageURL, req.LinkURL, req.SortOrder, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "创建失败"})
		return
	}
	req.ID = id
	req.CreatedAt = now
	req.UpdatedAt = now
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: req})
}

// AdminUpdateBanner 管理员 — 更新
func (h *BannerHandler) AdminUpdateBanner(c *gin.Context) {
	id := c.Param("id")
	var req Banner
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	db := h.Store.GetDB()
	now := time.Now()
	_, err := db.Exec("UPDATE banners SET title=?, subtitle=?, image_url=?, link_url=?, sort_order=?, updated_at=? WHERE id=?",
		req.Title, req.Subtitle, req.ImageURL, req.LinkURL, req.SortOrder, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "更新成功"})
}

// AdminDeleteBanner 管理员 — 删除
func (h *BannerHandler) AdminDeleteBanner(c *gin.Context) {
	id := c.Param("id")
	db := h.Store.GetDB()
	_, err := db.Exec("DELETE FROM banners WHERE id=?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "删除成功"})
}

// AdminResetBanners 管理员 — 重置为默认（清空数据库并保存默认配置）
func (h *BannerHandler) AdminResetBanners(c *gin.Context) {
	db := h.Store.GetDB()
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "事务启动失败"})
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM banners"); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "清空失败"})
		return
	}

	var req struct {
		Banners []Banner `json:"banners"`
	}
	c.ShouldBindJSON(&req)

	banners := req.Banners
	if len(banners) == 0 {
		banners = defaultBanners
	}
	now := time.Now()
	for _, b := range banners {
		id := b.ID
		if id == "" {
			id = genID("b")
		}
		tx.Exec("INSERT INTO banners (id, title, subtitle, image_url, link_url, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			id, b.Title, b.Subtitle, b.ImageURL, b.LinkURL, b.SortOrder, now, now)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "保存失败"})
		return
	}

	// 重新查询返回
	rows, err := db.Query("SELECT id, title, subtitle, image_url, link_url, sort_order, created_at, updated_at FROM banners ORDER BY sort_order ASC")
	if err != nil {
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: banners})
		return
	}
	defer rows.Close()
	var list []Banner
	for rows.Next() {
		var b Banner
		rows.Scan(&b.ID, &b.Title, &b.Subtitle, &b.ImageURL, &b.LinkURL, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt)
		list = append(list, b)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: list})
}

// 上传图片时用
func init() {
	// 确保 json 包被引用（实际已被上方使用）
	_ = json.Marshal
}