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

// UASCorpHandler 法人用户管理
type UASCorpHandler struct {
	store *store.Store
}

func NewUASCorpHandler(s *store.Store) *UASCorpHandler {
	return &UASCorpHandler{store: s}
}

func (h *UASCorpHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("pageNum"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 10)
	offset := (page - 1) * pageSize

	phone := c.Query("phone")
	creditCode := c.Query("creditCode")
	corpName := c.Query("corpName")
	status := c.Query("status")

	db := h.store.GetDB()
	where := "WHERE del_flag = 0"
	args := []interface{}{}
	if phone != "" {
		where += " AND phone LIKE ?"
		args = append(args, "%"+phone+"%")
	}
	if creditCode != "" {
		where += " AND credit_code LIKE ?"
		args = append(args, "%"+creditCode+"%")
	}
	if corpName != "" {
		where += " AND corp_name LIKE ?"
		args = append(args, "%"+corpName+"%")
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM u_corp_user "+where, args...).Scan(&total)

	rows, err := db.Query(
		"SELECT id, username, corp_type, corp_name, credit_code, legal_person, legal_id_card, agent_name, agent_id_card, phone, status, audit_status, create_time FROM u_corp_user "+
			where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}
	defer rows.Close()

	type CorpItem struct {
		ID          int64  `json:"id"`
		Username    string `json:"username"`
		CorpType    string `json:"corpType"`
		CorpName    string `json:"corpName"`
		CreditCode  string `json:"creditCode"`
		LegalPerson string `json:"legalPerson"`
		LegalIDCard string `json:"legalIdCard"`
		AgentName   string `json:"agentName"`
		AgentIDCard string `json:"agentIdCard"`
		Phone       string `json:"phone"`
		Status      int    `json:"status"`
		AuditStatus int    `json:"auditStatus"`
		CreateTime  string `json:"createTime"`
	}

	var list []CorpItem
	for rows.Next() {
		var u CorpItem
		var corpType, legalPerson, legalIDCard, agentName, agentIDCard, phone sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &corpType, &u.CorpName, &u.CreditCode, &legalPerson, &legalIDCard, &agentName, &agentIDCard, &phone, &u.Status, &u.AuditStatus, &u.CreateTime); err != nil {
			continue
		}
		u.CorpType = corpType.String
		u.LegalPerson = legalPerson.String
		u.LegalIDCard = utils.MaskIDCard(legalIDCard.String)
		u.AgentName = agentName.String
		u.AgentIDCard = utils.MaskIDCard(agentIDCard.String)
		u.Phone = phone.String
		list = append(list, u)
	}
	if list == nil {
		list = []CorpItem{}
	}

	utils.SuccessPage(c, total, list)
}

func (h *UASCorpHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()

	var username, corpName, creditCode string
	var corpType, legalPerson, legalIDCard, agentName, agentIDCard, phone, auditRemark sql.NullString
	var status, auditStatus int
	var createTime string

	err := db.QueryRow(
		"SELECT username, corp_type, corp_name, credit_code, legal_person, legal_id_card, agent_name, agent_id_card, phone, status, audit_status, audit_remark, create_time FROM u_corp_user WHERE id = ? AND del_flag = 0",
		id,
	).Scan(&username, &corpType, &corpName, &creditCode, &legalPerson, &legalIDCard, &agentName, &agentIDCard, &phone, &status, &auditStatus, &auditRemark, &createTime)

	if err != nil {
		utils.Error(c, "用户不存在")
		return
	}

	utils.Success(c, gin.H{
		"id":          id,
		"username":    username,
		"corpType":    corpType.String,
		"corpName":    corpName,
		"creditCode":  creditCode,
		"legalPerson": legalPerson.String,
		"legalIdCard": utils.MaskIDCard(legalIDCard.String),
		"agentName":   agentName.String,
		"agentIdCard": utils.MaskIDCard(agentIDCard.String),
		"phone":       phone.String,
		"status":      status,
		"auditStatus": auditStatus,
		"auditRemark": auditRemark.String,
		"createTime":  createTime,
	})
}

func (h *UASCorpHandler) Create(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		Password    string `json:"password"`
		CorpType    string `json:"corpType"`
		CorpName    string `json:"corpName" binding:"required"`
		CreditCode  string `json:"creditCode" binding:"required"`
		LegalPerson string `json:"legalPerson"`
		LegalIDCard string `json:"legalIdCard"`
		AgentName   string `json:"agentName"`
		AgentIDCard string `json:"agentIdCard"`
		Phone       string `json:"phone"`
		Status      int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "用户名、法人名称、信用代码必填")
		return
	}

	if req.Password == "" {
		req.Password = "123456"
	}
	if req.CorpType == "" {
		req.CorpType = "enterprise"
	}
	if req.Status == 0 {
		req.Status = 1
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	db := h.store.GetDB()
	_, err := db.Exec(
		"INSERT INTO u_corp_user (username, password, corp_type, corp_name, credit_code, legal_person, legal_id_card, agent_name, agent_id_card, phone, status, audit_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 2)",
		req.Username, string(hash), req.CorpType, req.CorpName, req.CreditCode, req.LegalPerson, req.LegalIDCard, req.AgentName, req.AgentIDCard, req.Phone, req.Status,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			utils.Error(c, "用户名或信用代码已存在")
			return
		}
		utils.Error(c, "新增失败")
		return
	}
	utils.SuccessMsg(c, "新增成功", nil)
}

func (h *UASCorpHandler) Update(c *gin.Context) {
	var req struct {
		ID          int64  `json:"id" binding:"required"`
		Username    string `json:"username"`
		CorpType    string `json:"corpType"`
		CorpName    string `json:"corpName"`
		CreditCode  string `json:"creditCode"`
		LegalPerson string `json:"legalPerson"`
		LegalIDCard string `json:"legalIdCard"`
		AgentName   string `json:"agentName"`
		AgentIDCard string `json:"agentIdCard"`
		Phone       string `json:"phone"`
		Status      int    `json:"status"`
	}
	c.ShouldBindJSON(&req)

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE u_corp_user SET username=?, corp_type=?, corp_name=?, credit_code=?, legal_person=?, legal_id_card=?, agent_name=?, agent_id_card=?, phone=?, status=? WHERE id=? AND del_flag=0",
		req.Username, req.CorpType, req.CorpName, req.CreditCode, req.LegalPerson, req.LegalIDCard, req.AgentName, req.AgentIDCard, req.Phone, req.Status, req.ID,
	)
	if err != nil {
		utils.Error(c, "修改失败")
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

func (h *UASCorpHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE u_corp_user SET del_flag = 1 WHERE id = ?", id)
	if err != nil {
		utils.Error(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "删除成功", nil)
}

func (h *UASCorpHandler) ChangeStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status int `json:"status"`
	}
	c.ShouldBindJSON(&req)
	db := h.store.GetDB()
	_, err := db.Exec("UPDATE u_corp_user SET status = ? WHERE id = ?", req.Status, id)
	if err != nil {
		utils.Error(c, "修改状态失败")
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}
