package handlers

import (
	"database/sql"
	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// LogHandler 日志管理
type LogHandler struct {
	store *store.Store
}

func NewLogHandler(s *store.Store) *LogHandler {
	return &LogHandler{store: s}
}

// LoginLogList 登录日志列表
func (h *LogHandler) LoginLogList(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	username := c.Query("username")
	loginIP := c.Query("loginIp")

	db := h.store.GetDB()
	where := "WHERE 1=1"
	args := []interface{}{}
	if username != "" {
		where += " AND username LIKE ?"
		args = append(args, "%"+username+"%")
	}
	if loginIP != "" {
		where += " AND login_ip LIKE ?"
		args = append(args, "%"+loginIP+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM u_login_log "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, user_id, username, login_type, login_ip, login_result, fail_reason, user_agent, login_time FROM u_login_log "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type LoginLogItem struct {
		ID          int64  `json:"id"`
		UserID      *int64 `json:"userId"`
		Username    string `json:"username"`
		LoginType   string `json:"loginType"`
		LoginIP     string `json:"loginIp"`
		LoginResult int    `json:"loginResult"`
		FailReason  string `json:"failReason"`
		UserAgent   string `json:"userAgent"`
		LoginTime   string `json:"loginTime"`
	}

	var list []LoginLogItem
	for rows.Next() {
		var l LoginLogItem
		var username, loginType, loginIP, failReason, userAgent sql.NullString
		if err := rows.Scan(&l.ID, &l.UserID, &username, &loginType, &loginIP, &l.LoginResult, &failReason, &userAgent, &l.LoginTime); err != nil {
			continue
		}
		l.Username = username.String
		l.LoginType = loginType.String
		l.LoginIP = loginIP.String
		l.FailReason = failReason.String
		l.UserAgent = userAgent.String
		list = append(list, l)
	}
	if list == nil {
		list = []LoginLogItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *LogHandler) CleanLoginLog(c *gin.Context) {
	db := h.store.GetDB()
	_, err := db.Exec("TRUNCATE TABLE u_login_log")
	if err != nil {
		utils.Error(c, "清空失败")
		return
	}
	utils.SuccessMsg(c, "清空成功", nil)
}

// AuditLogList 审计日志列表
func (h *LogHandler) AuditLogList(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	operName := c.Query("operName")
	operType := c.Query("operType")

	db := h.store.GetDB()
	where := "WHERE 1=1"
	args := []interface{}{}
	if operName != "" {
		where += " AND oper_name LIKE ?"
		args = append(args, "%"+operName+"%")
	}
	if operType != "" {
		where += " AND oper_type LIKE ?"
		args = append(args, "%"+operType+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM sys_audit_log "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, oper_name, oper_type, oper_content, oper_ip, oper_time FROM sys_audit_log "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type AuditLogItem struct {
		ID          int64  `json:"id"`
		OperName    string `json:"operName"`
		OperType    string `json:"operType"`
		OperContent string `json:"operContent"`
		OperIP      string `json:"operIp"`
		OperTime    string `json:"operTime"`
	}
	var list []AuditLogItem
	for rows.Next() {
		var a AuditLogItem
		var operName, operType, operContent, operIP sql.NullString
		if err := rows.Scan(&a.ID, &operName, &operType, &operContent, &operIP, &a.OperTime); err != nil {
			continue
		}
		a.OperName = operName.String
		a.OperType = operType.String
		a.OperContent = operContent.String
		a.OperIP = operIP.String
		list = append(list, a)
	}
	if list == nil {
		list = []AuditLogItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *LogHandler) CleanAuditLog(c *gin.Context) {
	db := h.store.GetDB()
	_, err := db.Exec("TRUNCATE TABLE sys_audit_log")
	if err != nil {
		utils.Error(c, "清空失败")
		return
	}
	utils.SuccessMsg(c, "清空成功", nil)
}

// SmsLogList 短信日志列表
func (h *LogHandler) SmsLogList(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	phone := c.Query("phone")

	db := h.store.GetDB()
	where := "WHERE 1=1"
	args := []interface{}{}
	if phone != "" {
		where += " AND phone LIKE ?"
		args = append(args, "%"+phone+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM sys_sms_log "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, phone, template, content, send_result, send_time FROM sys_sms_log "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type SmsLogItem struct {
		ID         int64  `json:"id"`
		Phone      string `json:"phone"`
		Template   string `json:"template"`
		Content    string `json:"content"`
		SendResult string `json:"sendResult"`
		SendTime   string `json:"sendTime"`
	}
	var list []SmsLogItem
	for rows.Next() {
		var s SmsLogItem
		var template, content, sendResult sql.NullString
		if err := rows.Scan(&s.ID, &s.Phone, &template, &content, &sendResult, &s.SendTime); err != nil {
			continue
		}
		s.Template = template.String
		s.Content = content.String
		s.SendResult = sendResult.String
		list = append(list, s)
	}
	if list == nil {
		list = []SmsLogItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *LogHandler) CleanSmsLog(c *gin.Context) {
	db := h.store.GetDB()
	_, err := db.Exec("TRUNCATE TABLE sys_sms_log")
	if err != nil {
		utils.Error(c, "清空失败")
		return
	}
	utils.SuccessMsg(c, "清空成功", nil)
}

// OperLogList 操作日志列表（基于审计日志）
func (h *LogHandler) OperLogList(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize
	operName := c.Query("operName")
	module := c.Query("module")
	operType := c.Query("operType")

	db := h.store.GetDB()
	where := "WHERE 1=1"
	args := []interface{}{}
	if operName != "" {
		where += " AND oper_name LIKE ?"
		args = append(args, "%"+operName+"%")
	}
	if module != "" {
		where += " AND oper_type LIKE ?"
		args = append(args, "%"+module+"%")
	}
	if operType != "" {
		where += " AND oper_type LIKE ?"
		args = append(args, "%"+operType+"%")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM sys_audit_log "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, oper_name, oper_type, oper_content, oper_ip, oper_time FROM sys_audit_log "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type OperLogItem struct {
		ID            int64  `json:"id"`
		OperName      string `json:"operName"`
		Module        string `json:"module"`
		OperType      string `json:"operType"`
		Description   string `json:"description"`
		RequestMethod string `json:"requestMethod"`
		OperIP        string `json:"operIp"`
		CostTime      int    `json:"costTime"`
		OperTime      string `json:"operTime"`
		OperUrl       string `json:"operUrl"`
		OperParam     string `json:"operParam"`
		JsonResult    string `json:"jsonResult"`
	}
	var list []OperLogItem
	for rows.Next() {
		var l OperLogItem
		var content string
		rows.Scan(&l.ID, &l.OperName, &l.OperType, &content, &l.OperIP, &l.OperTime)
		l.Description = content
		l.Module = l.OperType
		list = append(list, l)
	}
	if list == nil {
		list = []OperLogItem{}
	}
	utils.SuccessPage(c, total, list)
}

func (h *LogHandler) CleanOperLog(c *gin.Context) {
	db := h.store.GetDB()
	_, err := db.Exec("TRUNCATE TABLE sys_audit_log")
	if err != nil {
		utils.Error(c, "清空失败")
		return
	}
	utils.SuccessMsg(c, "清空成功", nil)
}
