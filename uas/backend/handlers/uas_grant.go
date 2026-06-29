package handlers

import (
	"strconv"

	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// GrantHandler 应用授权管理
type GrantHandler struct {
	store *store.Store
}

func NewGrantHandler(s *store.Store) *GrantHandler {
	return &GrantHandler{store: s}
}

// List 授权列表
func (h *GrantHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize

	appID := c.Query("appId")
	userType := c.Query("userType")

	db := h.store.GetDB()
	where := "WHERE g.status = 1"
	args := []interface{}{}
	if appID != "" {
		where += " AND g.app_id = ?"
		args = append(args, appID)
	}
	if userType != "" {
		where += " AND g.user_type = ?"
		args = append(args, userType)
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM u_grant g "+where, args...).Scan(&total)

	query := "SELECT g.id, g.user_id, g.user_type, g.app_id, g.grant_time, g.expire_time, g.status, " +
		"COALESCE(a.app_name, '') as app_name, " +
		"COALESCE(u.real_name, u.phone, COALESCE(c.corp_name, c.username, '')) as user_name " +
		"FROM u_grant g " +
		"LEFT JOIN u_app a ON g.app_id = a.app_id " +
		"LEFT JOIN u_user u ON g.user_type = 'personal' AND g.user_id = u.id " +
		"LEFT JOIN u_corp_user c ON g.user_type = 'corp' AND g.user_id = c.id " +
		where + " ORDER BY g.id DESC LIMIT ? OFFSET ?"

	rows, err := db.Query(query, append(args, pageSize, offset)...)
	if err != nil {
		utils.Error(c, "查询失败: "+err.Error())
		return
	}
	defer rows.Close()

	type GrantItem struct {
		ID         int64  `json:"id"`
		UserID     int64  `json:"userId"`
		UserType   string `json:"userType"`
		AppID      string `json:"appId"`
		AppName    string `json:"appName"`
		UserName   string `json:"userName"`
		GrantTime  string `json:"grantTime"`
		ExpireTime string `json:"expireTime"`
		Status     int    `json:"status"`
	}

	var list []GrantItem
	for rows.Next() {
		var g GrantItem
		var expireTime string
		if err := rows.Scan(&g.ID, &g.UserID, &g.UserType, &g.AppID, &g.GrantTime, &expireTime, &g.Status, &g.AppName, &g.UserName); err != nil {
			continue
		}
		g.ExpireTime = expireTime
		list = append(list, g)
	}
	if list == nil {
		list = []GrantItem{}
	}

	utils.SuccessPage(c, total, list)
}

// Delete 撤销授权
func (h *GrantHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE u_grant SET status = 0 WHERE id = ?", id)
	if err != nil {
		utils.Error(c, "撤销失败")
		return
	}
	utils.SuccessMsg(c, "撤销成功", nil)
}

// CheckGrant 检查用户是否已授权某应用（内部使用）
func (h *GrantHandler) CheckGrant(userID int64, userType, appID string) bool {
	db := h.store.GetDB()
	var count int64
	db.QueryRow(
		"SELECT COUNT(*) FROM u_grant WHERE user_id = ? AND user_type = ? AND app_id = ? AND status = 1",
		userID, userType, appID,
	).Scan(&count)
	return count > 0
}

// CreateGrant 创建授权记录（内部使用）
func (h *GrantHandler) CreateGrant(userID int64, userType, appID string) error {
	db := h.store.GetDB()
	_, err := db.Exec(
		"INSERT INTO u_grant (user_id, user_type, app_id, grant_time, status) VALUES (?, ?, ?, NOW(), 1)",
		userID, userType, appID,
	)
	return err
}
