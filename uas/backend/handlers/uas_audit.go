package handlers

import (
	"strconv"

	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// AuditHandler 审核管理
type AuditHandler struct {
	store *store.Store
}

func NewAuditHandler(s *store.Store) *AuditHandler {
	return &AuditHandler{store: s}
}

// List 审核列表（合并自然人+法人）
func (h *AuditHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	userType := c.DefaultQuery("userType", "all") // all/personal/corp
	auditStatus := c.Query("auditStatus")

	db := h.store.GetDB()

	type AuditItem struct {
		ID          int64  `json:"id"`
		UserType    string `json:"userType"`
		Username    string `json:"username"`
		RealName    string `json:"realName"`
		Phone       string `json:"phone"`
		AuditStatus int    `json:"auditStatus"`
		AuditRemark string `json:"auditRemark"`
		CreateTime  string `json:"createTime"`
	}

	var list []AuditItem
	var total int64

	// 自然人
	if userType == "all" || userType == "personal" {
		where := "WHERE del_flag = 0 AND audit_status > 0"
		args := []interface{}{}
		if auditStatus != "" {
			where += " AND audit_status = ?"
			args = append(args, auditStatus)
		}

		var t1 int64
		db.QueryRow("SELECT COUNT(*) FROM u_user "+where, args...).Scan(&t1)
		total += t1

		if userType == "personal" {
			args2 := append([]interface{}{}, args...)
			args2 = append(args2, pageSize, offset)
			rows, _ := db.Query(
				"SELECT id, phone, real_name, audit_status, audit_remark, create_time FROM u_user "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
				args2...,
			)
			if rows != nil {
				for rows.Next() {
					var u AuditItem
					var realName, auditRemark, phone string
					rows.Scan(&u.ID, &phone, &realName, &u.AuditStatus, &auditRemark, &u.CreateTime)
					u.UserType = "personal"
					u.Username = phone
					u.RealName = realName
					u.Phone = phone
					u.AuditRemark = auditRemark
					list = append(list, u)
				}
				rows.Close()
			}
		}
	}

	// 法人
	if userType == "all" || userType == "corp" {
		where := "WHERE del_flag = 0 AND audit_status > 0"
		args := []interface{}{}
		if auditStatus != "" {
			where += " AND audit_status = ?"
			args = append(args, auditStatus)
		}

		var t2 int64
		db.QueryRow("SELECT COUNT(*) FROM u_corp_user "+where, args...).Scan(&t2)
		total += t2

		if userType == "corp" {
			args2 := append([]interface{}{}, args...)
			args2 = append(args2, pageSize, offset)
			rows, _ := db.Query(
				"SELECT id, username, corp_name, phone, audit_status, audit_remark, create_time FROM u_corp_user "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
				args2...,
			)
			if rows != nil {
				for rows.Next() {
					var u AuditItem
					var username, corpName, phone, auditRemark string
					rows.Scan(&u.ID, &username, &corpName, &phone, &u.AuditStatus, &auditRemark, &u.CreateTime)
					u.UserType = "corp"
					u.Username = username
					u.RealName = corpName
					u.Phone = phone
					u.AuditRemark = auditRemark
					list = append(list, u)
				}
				rows.Close()
			}
		}
	}

	if list == nil {
		list = []AuditItem{}
	}
	utils.SuccessPage(c, total, list)
}

// AuditUser 审核自然人用户
func (h *AuditHandler) AuditUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		AuditStatus int    `json:"auditStatus"` // 2-通过 3-驳回
		AuditRemark string `json:"auditRemark"`
	}
	c.ShouldBindJSON(&req)

	if req.AuditStatus != 2 && req.AuditStatus != 3 {
		utils.BadRequest(c, "审核状态非法")
		return
	}

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE u_user SET audit_status = ?, audit_remark = ?, audit_time = NOW() WHERE id = ?",
		req.AuditStatus, req.AuditRemark, id,
	)
	if err != nil {
		utils.Error(c, "审核失败")
		return
	}
	utils.SuccessMsg(c, "审核成功", nil)
}

// AuditCorp 审核法人用户
func (h *AuditHandler) AuditCorp(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		AuditStatus int    `json:"auditStatus"`
		AuditRemark string `json:"auditRemark"`
	}
	c.ShouldBindJSON(&req)

	if req.AuditStatus != 2 && req.AuditStatus != 3 {
		utils.BadRequest(c, "审核状态非法")
		return
	}

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE u_corp_user SET audit_status = ?, audit_remark = ?, audit_time = NOW() WHERE id = ?",
		req.AuditStatus, req.AuditRemark, id,
	)
	if err != nil {
		utils.Error(c, "审核失败")
		return
	}
	utils.SuccessMsg(c, "审核成功", nil)
}
