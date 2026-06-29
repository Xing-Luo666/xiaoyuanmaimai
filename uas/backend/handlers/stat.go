package handlers

import (
	"time"

	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
)

// StatHandler 统计分析
type StatHandler struct {
	store *store.Store
}

func NewStatHandler(s *store.Store) *StatHandler {
	return &StatHandler{store: s}
}

// Account 账户统计
func (h *StatHandler) Account(c *gin.Context) {
	db := h.store.GetDB()

	var personalTotal, personalL1, personalL2, personalL3, personalAudit int64
	var corpTotal, corpAudit int64
	var appTotal, appActive int64

	db.QueryRow("SELECT COUNT(*) FROM u_user WHERE del_flag = 0").Scan(&personalTotal)
	db.QueryRow("SELECT COUNT(*) FROM u_user WHERE del_flag = 0 AND auth_level = 'L1'").Scan(&personalL1)
	db.QueryRow("SELECT COUNT(*) FROM u_user WHERE del_flag = 0 AND auth_level = 'L2'").Scan(&personalL2)
	db.QueryRow("SELECT COUNT(*) FROM u_user WHERE del_flag = 0 AND auth_level = 'L3'").Scan(&personalL3)
	db.QueryRow("SELECT COUNT(*) FROM u_user WHERE del_flag = 0 AND audit_status = 1").Scan(&personalAudit)
	db.QueryRow("SELECT COUNT(*) FROM u_corp_user WHERE del_flag = 0").Scan(&corpTotal)
	db.QueryRow("SELECT COUNT(*) FROM u_corp_user WHERE del_flag = 0 AND audit_status = 1").Scan(&corpAudit)
	db.QueryRow("SELECT COUNT(*) FROM u_app WHERE del_flag = 0").Scan(&appTotal)
	db.QueryRow("SELECT COUNT(*) FROM u_app WHERE del_flag = 0 AND status = 1").Scan(&appActive)

	utils.Success(c, gin.H{
		"personalTotal": personalTotal,
		"personalL1":    personalL1,
		"personalL2":    personalL2,
		"personalL3":    personalL3,
		"personalAudit": personalAudit,
		"corpTotal":     corpTotal,
		"corpAudit":     corpAudit,
		"appTotal":      appTotal,
		"appActive":     appActive,
	})
}

// Login 登录统计
func (h *StatHandler) Login(c *gin.Context) {
	db := h.store.GetDB()
	days := 7

	type DayStat struct {
		Date  string `json:"date"`
		Total int64  `json:"total"`
		Ok    int64  `json:"ok"`
		Fail  int64  `json:"fail"`
	}

	var list []DayStat
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		start := date + " 00:00:00"
		end := date + " 23:59:59"

		var total, ok, fail int64
		db.QueryRow("SELECT COUNT(*) FROM u_login_log WHERE login_time BETWEEN ? AND ?", start, end).Scan(&total)
		db.QueryRow("SELECT COUNT(*) FROM u_login_log WHERE login_time BETWEEN ? AND ? AND login_result = 1", start, end).Scan(&ok)
		db.QueryRow("SELECT COUNT(*) FROM u_login_log WHERE login_time BETWEEN ? AND ? AND login_result = 0", start, end).Scan(&fail)
		list = append(list, DayStat{Date: date, Total: total, Ok: ok, Fail: fail})
	}

	utils.Success(c, list)
}

// API 接口统计（简化：查询审计日志按操作类型分组）
func (h *StatHandler) API(c *gin.Context) {
	db := h.store.GetDB()
	rows, err := db.Query(
		"SELECT DATE(oper_time) as date, COUNT(*) as total FROM sys_audit_log WHERE oper_time >= DATE_SUB(NOW(), INTERVAL 7 DAY) GROUP BY DATE(oper_time) ORDER BY date",
	)
	if err != nil {
		utils.Success(c, []interface{}{})
		return
	}
	defer rows.Close()

	type APIStat struct {
		Date  string `json:"date"`
		Total int64  `json:"total"`
	}
	var list []APIStat
	for rows.Next() {
		var s APIStat
		var date []byte
		rows.Scan(&date, &s.Total)
		s.Date = string(date)
		list = append(list, s)
	}
	if list == nil {
		list = []APIStat{}
	}
	utils.Success(c, list)
}

// SMS 消息统计（简化）
func (h *StatHandler) SMS(c *gin.Context) {
	db := h.store.GetDB()
	days := 7
	type SMSStat struct {
		Date  string `json:"date"`
		Total int64  `json:"total"`
	}
	var list []SMSStat
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		start := date + " 00:00:00"
		end := date + " 23:59:59"
		var total int64
		db.QueryRow("SELECT COUNT(*) FROM sys_sms_log WHERE send_time BETWEEN ? AND ?", start, end).Scan(&total)
		list = append(list, SMSStat{Date: date, Total: total})
	}
	utils.Success(c, list)
}

// Overview 统计概览（卡片数据）
func (h *StatHandler) Overview(c *gin.Context) {
	db := h.store.GetDB()

	var personalCount, corpCount, appCount, todayGrantCount int64
	db.QueryRow("SELECT COUNT(*) FROM u_user WHERE del_flag = 0").Scan(&personalCount)
	db.QueryRow("SELECT COUNT(*) FROM u_corp_user WHERE del_flag = 0").Scan(&corpCount)
	db.QueryRow("SELECT COUNT(*) FROM u_app WHERE del_flag = 0").Scan(&appCount)

	today := time.Now().Format("2006-01-02")
	start := today + " 00:00:00"
	end := today + " 23:59:59"
	db.QueryRow("SELECT COUNT(*) FROM u_grant WHERE grant_time BETWEEN ? AND ?", start, end).Scan(&todayGrantCount)

	utils.Success(c, gin.H{
		"personalCount":   personalCount,
		"corpCount":       corpCount,
		"appCount":        appCount,
		"todayGrantCount": todayGrantCount,
	})
}

// Trend 近7天授权趋势
func (h *StatHandler) Trend(c *gin.Context) {
	db := h.store.GetDB()
	days := 7

	type TrendItem struct {
		Date       string `json:"date"`
		GrantCount int64  `json:"grantCount"`
		LoginCount int64  `json:"loginCount"`
	}

	var list []TrendItem
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		start := date + " 00:00:00"
		end := date + " 23:59:59"

		var grantCount, loginCount int64
		db.QueryRow("SELECT COUNT(*) FROM u_grant WHERE grant_time BETWEEN ? AND ?", start, end).Scan(&grantCount)
		db.QueryRow("SELECT COUNT(*) FROM u_login_log WHERE login_time BETWEEN ? AND ?", start, end).Scan(&loginCount)
		list = append(list, TrendItem{Date: date, GrantCount: grantCount, LoginCount: loginCount})
	}

	utils.Success(c, list)
}

// AppType 应用类型分布
func (h *StatHandler) AppType(c *gin.Context) {
	db := h.store.GetDB()
	rows, err := db.Query("SELECT app_type, COUNT(*) as cnt FROM u_app WHERE del_flag = 0 GROUP BY app_type")
	if err != nil {
		utils.Success(c, []interface{}{})
		return
	}
	defer rows.Close()

	type Item struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}
	var list []Item
	typeMap := map[string]string{"web": "Web应用", "mobile": "移动应用", "desktop": "桌面应用"}
	for rows.Next() {
		var appType string
		var cnt int64
		rows.Scan(&appType, &cnt)
		name := typeMap[appType]
		if name == "" {
			name = appType
		}
		list = append(list, Item{Name: name, Count: cnt})
	}
	if list == nil {
		list = []Item{}
	}
	utils.Success(c, list)
}

// TopApps 授权量Top10应用
func (h *StatHandler) TopApps(c *gin.Context) {
	db := h.store.GetDB()
	rows, err := db.Query(`
		SELECT a.app_name, COUNT(g.id) as cnt
		FROM u_app a
		LEFT JOIN u_grant g ON a.app_id = g.app_id AND g.status = 1
		WHERE a.del_flag = 0
		GROUP BY a.app_id, a.app_name
		ORDER BY cnt DESC
		LIMIT 10
	`)
	if err != nil {
		utils.Success(c, []interface{}{})
		return
	}
	defer rows.Close()

	type Item struct {
		AppName    string `json:"appName"`
		GrantCount int64  `json:"grantCount"`
	}
	var list []Item
	for rows.Next() {
		var item Item
		rows.Scan(&item.AppName, &item.GrantCount)
		list = append(list, item)
	}
	if list == nil {
		list = []Item{}
	}
	utils.Success(c, list)
}

// ActiveUsers 近7天活跃用户
func (h *StatHandler) ActiveUsers(c *gin.Context) {
	db := h.store.GetDB()
	rows, err := db.Query(`
		SELECT username, COUNT(*) as cnt
		FROM u_login_log
		WHERE login_time >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY username
		ORDER BY cnt DESC
		LIMIT 10
	`)
	if err != nil {
		utils.Success(c, []interface{}{})
		return
	}
	defer rows.Close()

	type Item struct {
		UserName   string `json:"userName"`
		LoginCount int64  `json:"loginCount"`
	}
	var list []Item
	for rows.Next() {
		var item Item
		rows.Scan(&item.UserName, &item.LoginCount)
		list = append(list, item)
	}
	if list == nil {
		list = []Item{}
	}
	utils.Success(c, list)
}
