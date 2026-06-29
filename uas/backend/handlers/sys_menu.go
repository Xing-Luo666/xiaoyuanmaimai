package handlers

import (
	"database/sql"
	"strconv"

	"uas/models"
	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// MenuHandler 菜单管理
type MenuHandler struct {
	store *store.Store
}

func NewMenuHandler(s *store.Store) *MenuHandler {
	return &MenuHandler{store: s}
}

func (h *MenuHandler) List(c *gin.Context) {
	menuName := c.Query("menuName")
	db := h.store.GetDB()
	where := "WHERE del_flag = 0"
	args := []interface{}{}
	if menuName != "" {
		where += " AND menu_name LIKE ?"
		args = append(args, "%"+menuName+"%")
	}
	rows, err := db.Query(
		"SELECT id, menu_name, parent_id, menu_sort, path, component, menu_type, visible, perms, icon, create_time FROM sys_menu "+
			where+" ORDER BY parent_id, menu_sort",
		args...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	var menus []models.SysMenu
	for rows.Next() {
		var m models.SysMenu
		var path, component, perms, icon sql.NullString
		var createTime string
		if err := rows.Scan(&m.ID, &m.MenuName, &m.ParentID, &m.MenuSort, &path, &component, &m.MenuType, &m.Visible, &perms, &icon, &createTime); err != nil {
			continue
		}
		m.Path = path.String
		m.Component = component.String
		m.Perms = perms.String
		m.Icon = icon.String
		menus = append(menus, m)
	}
	if menus == nil {
		menus = []models.SysMenu{}
	}
	// 返回树形结构
	tree := buildMenuTree(menus, 0)
	utils.Success(c, tree)
}

func (h *MenuHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	var m models.SysMenu
	var path, component, perms, icon sql.NullString
	err := db.QueryRow(
		"SELECT id, menu_name, parent_id, menu_sort, path, component, menu_type, visible, perms, icon FROM sys_menu WHERE id = ?",
		id,
	).Scan(&m.ID, &m.MenuName, &m.ParentID, &m.MenuSort, &path, &component, &m.MenuType, &m.Visible, &perms, &icon)
	if err != nil {
		utils.Error(c, "菜单不存在")
		return
	}
	m.Path = path.String
	m.Component = component.String
	m.Perms = perms.String
	m.Icon = icon.String
	utils.Success(c, m)
}

func (h *MenuHandler) Create(c *gin.Context) {
	var req struct {
		MenuName  string `json:"menuName" binding:"required"`
		ParentID  int64  `json:"parentId"`
		MenuSort  int    `json:"menuSort"`
		Path      string `json:"path"`
		Component string `json:"component"`
		MenuType  string `json:"menuType"`
		Visible   int    `json:"visible"`
		Perms     string `json:"perms"`
		Icon      string `json:"icon"`
	}
	c.ShouldBindJSON(&req)
	if req.MenuType == "" {
		req.MenuType = "C"
	}
	if req.Visible == 0 {
		req.Visible = 1
	}
	db := h.store.GetDB()
	_, err := db.Exec(
		"INSERT INTO sys_menu (menu_name, parent_id, menu_sort, path, component, menu_type, visible, perms, icon) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		req.MenuName, req.ParentID, req.MenuSort, req.Path, req.Component, req.MenuType, req.Visible, req.Perms, req.Icon,
	)
	if err != nil {
		utils.Error(c, "新增失败")
		return
	}
	utils.SuccessMsg(c, "新增成功", nil)
}

func (h *MenuHandler) Update(c *gin.Context) {
	var req struct {
		ID        int64  `json:"id" binding:"required"`
		MenuName  string `json:"menuName"`
		ParentID  int64  `json:"parentId"`
		MenuSort  int    `json:"menuSort"`
		Path      string `json:"path"`
		Component string `json:"component"`
		MenuType  string `json:"menuType"`
		Visible   int    `json:"visible"`
		Perms     string `json:"perms"`
		Icon      string `json:"icon"`
	}
	c.ShouldBindJSON(&req)
	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE sys_menu SET menu_name=?, parent_id=?, menu_sort=?, path=?, component=?, menu_type=?, visible=?, perms=?, icon=? WHERE id=?",
		req.MenuName, req.ParentID, req.MenuSort, req.Path, req.Component, req.MenuType, req.Visible, req.Perms, req.Icon, req.ID,
	)
	if err != nil {
		utils.Error(c, "修改失败")
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

func (h *MenuHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	// 检查是否有子菜单
	var count int64
	db.QueryRow("SELECT COUNT(*) FROM sys_menu WHERE parent_id = ? AND del_flag = 0", id).Scan(&count)
	if count > 0 {
		utils.Error(c, "存在子菜单，不允许删除")
		return
	}
	_, err := db.Exec("UPDATE sys_menu SET del_flag = 1 WHERE id = ?", id)
	if err != nil {
		utils.Error(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "删除成功", nil)
}

// TreeSelect 获取菜单树（用于角色分配菜单）
func (h *MenuHandler) TreeSelect(c *gin.Context) {
	db := h.store.GetDB()
	rows, err := db.Query(
		"SELECT id, menu_name, parent_id FROM sys_menu WHERE del_flag = 0 AND menu_type IN ('M', 'C') ORDER BY parent_id, menu_sort",
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type TreeNode struct {
		ID       int64      `json:"id"`
		Label    string     `json:"label"`
		Children []TreeNode `json:"children"`
	}

	var nodes []struct {
		ID       int64
		Label    string
		ParentID int64
	}
	for rows.Next() {
		var n struct {
			ID       int64
			Label    string
			ParentID int64
		}
		rows.Scan(&n.ID, &n.Label, &n.ParentID)
		nodes = append(nodes, n)
	}

	var build func(pid int64) []TreeNode
	build = func(pid int64) []TreeNode {
		var tree []TreeNode
		for _, n := range nodes {
			if n.ParentID == pid {
				t := TreeNode{ID: n.ID, Label: n.Label}
				t.Children = build(n.ID)
				tree = append(tree, t)
			}
		}
		return tree
	}

	utils.Success(c, build(0))
}

// RoleMenuTreeSelect 获取菜单树+角色已选菜单
func (h *MenuHandler) RoleMenuTreeSelect(c *gin.Context) {
	roleID, _ := strconv.ParseInt(c.Param("roleId"), 10, 64)
	db := h.store.GetDB()

	// 查询角色已选菜单ID
	var checkedKeys []int64
	rows, _ := db.Query("SELECT menu_id FROM sys_role_menu WHERE role_id = ?", roleID)
	if rows != nil {
		for rows.Next() {
			var mid int64
			rows.Scan(&mid)
			checkedKeys = append(checkedKeys, mid)
		}
		rows.Close()
	}
	if checkedKeys == nil {
		checkedKeys = []int64{}
	}

	// 复用 TreeSelect 逻辑
	h.TreeSelect(c)
}
