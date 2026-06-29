package handlers

import (
	"database/sql"
	"strconv"

	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// SysUserHandler 系统管理员
type SysUserHandler struct {
	store *store.Store
}

func NewSysUserHandler(s *store.Store) *SysUserHandler {
	return &SysUserHandler{store: s}
}

func (h *SysUserHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	username := c.Query("username")
	phone := c.Query("phone")
	status := c.Query("status")

	db := h.store.GetDB()
	where := "WHERE del_flag = 0"
	args := []interface{}{}
	if username != "" {
		where += " AND username LIKE ?"
		args = append(args, "%"+username+"%")
	}
	if phone != "" {
		where += " AND phone LIKE ?"
		args = append(args, "%"+phone+"%")
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM sys_user "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, username, nickname, email, phone, sex, avatar, status, remark, create_time FROM sys_user "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type UserItem struct {
		ID         int64  `json:"id"`
		Username   string `json:"username"`
		Nickname   string `json:"nickname"`
		Email      string `json:"email"`
		Phone      string `json:"phone"`
		Sex        int    `json:"sex"`
		Avatar     string `json:"avatar"`
		Status     int    `json:"status"`
		Remark     string `json:"remark"`
		CreateTime string `json:"createTime"`
	}
	var list []UserItem
	for rows.Next() {
		var u UserItem
		var email, phone, avatar, remark sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &u.Nickname, &email, &phone, &u.Sex, &avatar, &u.Status, &remark, &u.CreateTime); err != nil {
			continue
		}
		u.Email = email.String
		u.Phone = phone.String
		u.Avatar = avatar.String
		u.Remark = remark.String
		list = append(list, u)
	}
	if list == nil {
		list = []UserItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *SysUserHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	var u struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Sex      int    `json:"sex"`
		Avatar   string `json:"avatar"`
		Status   int    `json:"status"`
		Remark   string `json:"remark"`
	}
	var email, phone, avatar, remark sql.NullString
	err := db.QueryRow(
		"SELECT id, username, nickname, email, phone, sex, avatar, status, remark FROM sys_user WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.Nickname, &email, &phone, &u.Sex, &avatar, &u.Status, &remark)
	if err != nil {
		utils.Error(c, "用户不存在")
		return
	}
	u.Email = email.String
	u.Phone = phone.String
	u.Avatar = avatar.String
	u.Remark = remark.String

	// 查询角色
	var roleIDs []int64
	rows, _ := db.Query("SELECT role_id FROM sys_user_role WHERE user_id = ?", id)
	if rows != nil {
		for rows.Next() {
			var rid int64
			rows.Scan(&rid)
			roleIDs = append(roleIDs, rid)
		}
		rows.Close()
	}
	utils.Success(c, gin.H{
		"id":       u.ID,
		"username": u.Username,
		"nickname": u.Nickname,
		"email":    u.Email,
		"phone":    u.Phone,
		"sex":      u.Sex,
		"avatar":   u.Avatar,
		"status":   u.Status,
		"remark":   u.Remark,
		"roleIds":  roleIDs,
	})
}

func (h *SysUserHandler) Create(c *gin.Context) {
	var req struct {
		Username string  `json:"username" binding:"required"`
		Password string  `json:"password" binding:"required"`
		Nickname string  `json:"nickname"`
		Email    string  `json:"email"`
		Phone    string  `json:"phone"`
		Sex      int     `json:"sex"`
		Status   int     `json:"status"`
		Remark   string  `json:"remark"`
		RoleIDs  []int64 `json:"roleIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "用户名、密码必填")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	db := h.store.GetDB()
	res, err := db.Exec(
		"INSERT INTO sys_user (username, password, nickname, email, phone, sex, status, remark) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		req.Username, string(hash), req.Nickname, req.Email, req.Phone, req.Sex, req.Status, req.Remark,
	)
	if err != nil {
		utils.Error(c, "新增失败: "+err.Error())
		return
	}
	uid, _ := res.LastInsertId()
	// 保存角色关联
	for _, rid := range req.RoleIDs {
		db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)", uid, rid)
	}
	utils.SuccessMsg(c, "新增成功", nil)
}

func (h *SysUserHandler) Update(c *gin.Context) {
	var req struct {
		ID       int64   `json:"id" binding:"required"`
		Nickname string  `json:"nickname"`
		Email    string  `json:"email"`
		Phone    string  `json:"phone"`
		Sex      int     `json:"sex"`
		Status   int     `json:"status"`
		Remark   string  `json:"remark"`
		RoleIDs  []int64 `json:"roleIds"`
	}
	c.ShouldBindJSON(&req)

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE sys_user SET nickname=?, email=?, phone=?, sex=?, status=?, remark=? WHERE id=?",
		req.Nickname, req.Email, req.Phone, req.Sex, req.Status, req.Remark, req.ID,
	)
	if err != nil {
		utils.Error(c, "修改失败")
		return
	}
	// 更新角色
	db.Exec("DELETE FROM sys_user_role WHERE user_id = ?", req.ID)
	for _, rid := range req.RoleIDs {
		db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)", req.ID, rid)
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

func (h *SysUserHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE sys_user SET del_flag = 1 WHERE id = ?", id)
	if err != nil {
		utils.Error(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "删除成功", nil)
}

func (h *SysUserHandler) ResetPwd(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	c.ShouldBindJSON(&req)
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE sys_user SET password = ? WHERE id = ?", string(hash), id)
	if err != nil {
		utils.Error(c, "重置失败")
		return
	}
	utils.SuccessMsg(c, "重置成功", nil)
}

func (h *SysUserHandler) ChangeStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status int `json:"status"`
	}
	c.ShouldBindJSON(&req)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE sys_user SET status = ? WHERE id = ?", req.Status, id)
	if err != nil {
		utils.Error(c, "修改状态失败")
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}
