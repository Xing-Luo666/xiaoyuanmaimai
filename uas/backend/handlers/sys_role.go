package handlers

import (
	"database/sql"
	"strconv"

	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// RoleHandler 角色管理
type RoleHandler struct {
	store *store.Store
}

func NewRoleHandler(s *store.Store) *RoleHandler {
	return &RoleHandler{store: s}
}

func (h *RoleHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	roleName := c.Query("roleName")
	roleKey := c.Query("roleKey")

	db := h.store.GetDB()
	where := "WHERE del_flag = 0"
	args := []interface{}{}
	if roleName != "" {
		where += " AND role_name LIKE ?"
		args = append(args, "%"+roleName+"%")
	}
	if roleKey != "" {
		where += " AND role_key LIKE ?"
		args = append(args, "%"+roleKey+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM sys_role "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, role_name, role_key, role_sort, status, remark, create_time FROM sys_role "+
			where+" ORDER BY role_sort ASC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type RoleItem struct {
		ID         int64  `json:"id"`
		RoleName   string `json:"roleName"`
		RoleKey    string `json:"roleKey"`
		RoleSort   int    `json:"roleSort"`
		Status     int    `json:"status"`
		Remark     string `json:"remark"`
		CreateTime string `json:"createTime"`
	}
	var list []RoleItem
	for rows.Next() {
		var r RoleItem
		var remark sql.NullString
		rows.Scan(&r.ID, &r.RoleName, &r.RoleKey, &r.RoleSort, &r.Status, &remark, &r.CreateTime)
		r.Remark = remark.String
		list = append(list, r)
	}
	if list == nil {
		list = []RoleItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	var r struct {
		ID       int64  `json:"id"`
		RoleName string `json:"roleName"`
		RoleKey  string `json:"roleKey"`
		RoleSort int    `json:"roleSort"`
		Status   int    `json:"status"`
		Remark   string `json:"remark"`
	}
	var remark sql.NullString
	err := db.QueryRow(
		"SELECT id, role_name, role_key, role_sort, status, remark FROM sys_role WHERE id = ?",
		id,
	).Scan(&r.ID, &r.RoleName, &r.RoleKey, &r.RoleSort, &r.Status, &remark)
	if err != nil {
		utils.Error(c, "角色不存在")
		return
	}
	r.Remark = remark.String

	// 查询已分配菜单
	var menuIDs []int64
	rows, _ := db.Query("SELECT menu_id FROM sys_role_menu WHERE role_id = ?", id)
	if rows != nil {
		for rows.Next() {
			var mid int64
			rows.Scan(&mid)
			menuIDs = append(menuIDs, mid)
		}
		rows.Close()
	}
	utils.Success(c, gin.H{
		"id":       r.ID,
		"roleName": r.RoleName,
		"roleKey":  r.RoleKey,
		"roleSort": r.RoleSort,
		"status":   r.Status,
		"remark":   r.Remark,
		"menuIds":  menuIDs,
	})
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req struct {
		RoleName string  `json:"roleName" binding:"required"`
		RoleKey  string  `json:"roleKey" binding:"required"`
		RoleSort int     `json:"roleSort"`
		Status   int     `json:"status"`
		Remark   string  `json:"remark"`
		MenuIDs  []int64 `json:"menuIds"`
	}
	c.ShouldBindJSON(&req)
	if req.Status == 0 {
		req.Status = 1
	}
	db := h.store.GetDB()
	res, err := db.Exec(
		"INSERT INTO sys_role (role_name, role_key, role_sort, status, remark) VALUES (?, ?, ?, ?, ?)",
		req.RoleName, req.RoleKey, req.RoleSort, req.Status, req.Remark,
	)
	if err != nil {
		utils.Error(c, "新增失败")
		return
	}
	rid, _ := res.LastInsertId()
	for _, mid := range req.MenuIDs {
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", rid, mid)
	}
	utils.SuccessMsg(c, "新增成功", nil)
}

func (h *RoleHandler) Update(c *gin.Context) {
	var req struct {
		ID       int64   `json:"id" binding:"required"`
		RoleName string  `json:"roleName"`
		RoleKey  string  `json:"roleKey"`
		RoleSort int     `json:"roleSort"`
		Status   int     `json:"status"`
		Remark   string  `json:"remark"`
		MenuIDs  []int64 `json:"menuIds"`
	}
	c.ShouldBindJSON(&req)
	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE sys_role SET role_name=?, role_key=?, role_sort=?, status=?, remark=? WHERE id=?",
		req.RoleName, req.RoleKey, req.RoleSort, req.Status, req.Remark, req.ID,
	)
	if err != nil {
		utils.Error(c, "修改失败")
		return
	}
	db.Exec("DELETE FROM sys_role_menu WHERE role_id = ?", req.ID)
	for _, mid := range req.MenuIDs {
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", req.ID, mid)
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE sys_role SET del_flag = 1 WHERE id = ?", id)
	if err != nil {
		utils.Error(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "删除成功", nil)
}
