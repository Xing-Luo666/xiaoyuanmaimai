package handlers

import (
	"database/sql"
	"strconv"
	"strings"

	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// UASUserHandler 自然人用户管理
type UASUserHandler struct {
	store *store.Store
}

func NewUASUserHandler(s *store.Store) *UASUserHandler {
	return &UASUserHandler{store: s}
}

// List 分页查询自然人用户
func (h *UASUserHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	phone := c.Query("phone")
	status := c.Query("status")
	idCard := c.Query("idCard")

	db := h.store.GetDB()
	where := "WHERE del_flag = 0"
	args := []interface{}{}
	if phone != "" {
		where += " AND phone LIKE ?"
		args = append(args, "%"+phone+"%")
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	if idCard != "" {
		where += " AND id_card_no LIKE ?"
		args = append(args, "%"+idCard+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM u_user "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, phone, real_name, id_card_type, id_card_no, auth_level, nickname, email, status, audit_status, audit_remark, audit_time, create_time, update_time FROM u_user "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败: "+err.Error())
		return
	}
	defer rows.Close()

	type UserItem struct {
		ID          int64   `json:"id"`
		Phone       string  `json:"phone"`
		RealName    string  `json:"realName"`
		IDCardType  int     `json:"idCardType"`
		IDCardNo    string  `json:"idCardNo"`
		AuthLevel   string  `json:"authLevel"`
		Nickname    string  `json:"nickname"`
		Email       string  `json:"email"`
		Status      int     `json:"status"`
		AuditStatus int     `json:"auditStatus"`
		AuditRemark string  `json:"auditRemark"`
		AuditTime   *string `json:"auditTime"`
		CreateTime  string  `json:"createTime"`
		UpdateTime  string  `json:"updateTime"`
	}

	var list []UserItem
	for rows.Next() {
		var u UserItem
		var idCard, nickname, email, auditRemark sql.NullString
		var auditTime sql.NullTime
		if err := rows.Scan(&u.ID, &u.Phone, &u.RealName, &u.IDCardType, &idCard, &u.AuthLevel, &nickname, &email, &u.Status, &u.AuditStatus, &auditRemark, &auditTime, &u.CreateTime, &u.UpdateTime); err != nil {
			continue
		}
		u.IDCardNo = utils.MaskIDCard(idCard.String)
		u.Nickname = nickname.String
		u.Email = email.String
		u.AuditRemark = auditRemark.String
		if auditTime.Valid {
			u.AuditTime = strPtr(auditTime.Time.Format("2006-01-02 15:04:05"))
		}
		list = append(list, u)
	}
	if list == nil {
		list = []UserItem{}
	}

	utils.SuccessPage(c, total, list)
}

// Get 查询单个用户
func (h *UASUserHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()

	var phone, realName, idCard, authLevel, nickname, email, auditRemark string
	var idCardType, status, auditStatus int
	var auditTime sql.NullTime
	var createTime, updateTime string

	err := db.QueryRow(
		"SELECT phone, real_name, id_card_type, id_card_no, auth_level, nickname, email, status, audit_status, audit_remark, audit_time, create_time, update_time FROM u_user WHERE id = ? AND del_flag = 0",
		id,
	).Scan(&phone, &realName, &idCardType, &idCard, &authLevel, &nickname, &email, &status, &auditStatus, &auditRemark, &auditTime, &createTime, &updateTime)

	if err != nil {
		utils.Error(c, "用户不存在")
		return
	}

	utils.Success(c, gin.H{
		"id":          id,
		"phone":       phone,
		"realName":    realName,
		"idCardType":  idCardType,
		"idCardNo":    utils.MaskIDCard(idCard),
		"authLevel":   authLevel,
		"nickname":    nickname,
		"email":       email,
		"status":      status,
		"auditStatus": auditStatus,
		"auditRemark": auditRemark,
		"auditTime":   nullTimeStr(auditTime),
		"createTime":  createTime,
		"updateTime":  updateTime,
	})
}

// Create 新增用户
func (h *UASUserHandler) Create(c *gin.Context) {
	var req struct {
		Phone      string `json:"phone" binding:"required"`
		Password   string `json:"password"`
		RealName   string `json:"realName"`
		IDCardType int    `json:"idCardType"`
		IDCardNo   string `json:"idCardNo"`
		AuthLevel  string `json:"authLevel"`
		Nickname   string `json:"nickname"`
		Email      string `json:"email"`
		Status     int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "手机号必填")
		return
	}

	if req.Password == "" {
		req.Password = "123456"
	}
	if req.AuthLevel == "" {
		req.AuthLevel = "L1"
	}
	if req.Status == 0 {
		req.Status = 1
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	db := h.store.GetDB()
	_, err := db.Exec(
		"INSERT INTO u_user (phone, password, real_name, id_card_type, id_card_no, auth_level, nickname, email, status, audit_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 2)",
		req.Phone, string(hash), req.RealName, req.IDCardType, req.IDCardNo, req.AuthLevel, req.Nickname, req.Email, req.Status,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			utils.Error(c, "手机号已存在")
			return
		}
		utils.Error(c, "新增失败: "+err.Error())
		return
	}
	utils.SuccessMsg(c, "新增成功", nil)
}

// Update 修改用户
func (h *UASUserHandler) Update(c *gin.Context) {
	var req struct {
		ID         int64  `json:"id" binding:"required"`
		Phone      string `json:"phone"`
		RealName   string `json:"realName"`
		IDCardType int    `json:"idCardType"`
		IDCardNo   string `json:"idCardNo"`
		AuthLevel  string `json:"authLevel"`
		Nickname   string `json:"nickname"`
		Email      string `json:"email"`
		Status     int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "id必填")
		return
	}

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE u_user SET phone=?, real_name=?, id_card_type=?, id_card_no=?, auth_level=?, nickname=?, email=?, status=? WHERE id=? AND del_flag=0",
		req.Phone, req.RealName, req.IDCardType, req.IDCardNo, req.AuthLevel, req.Nickname, req.Email, req.Status, req.ID,
	)
	if err != nil {
		utils.Error(c, "修改失败: "+err.Error())
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

// Delete 删除用户
func (h *UASUserHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE u_user SET del_flag = 1 WHERE id = ?", id)
	if err != nil {
		utils.Error(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "删除成功", nil)
}

// ChangeStatus 修改状态
func (h *UASUserHandler) ChangeStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status int `json:"status"`
	}
	c.ShouldBindJSON(&req)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE u_user SET status = ? WHERE id = ?", req.Status, id)
	if err != nil {
		utils.Error(c, "修改状态失败")
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

// 工具函数
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func strPtr(s string) *string {
	return &s
}

func nullTimeStr(t sql.NullTime) string {
	if t.Valid {
		return t.Time.Format("2006-01-02 15:04:05")
	}
	return ""
}
