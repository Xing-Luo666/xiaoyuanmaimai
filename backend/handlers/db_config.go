package handlers

import (
	"database/sql"
	"net/http"
	"school-trade/models"
	"school-trade/store"

	"github.com/gin-gonic/gin"
)

type DBConfigHandler struct {
	DBStore *store.DBStore
	CfgFile string
}

func NewDBConfigHandler(s *store.DBStore, cfgFile string) *DBConfigHandler {
	return &DBConfigHandler{DBStore: s, CfgFile: cfgFile}
}

func (h *DBConfigHandler) GetConfig(c *gin.Context) {
	cfg := h.DBStore.GetConfig()
	// 不返回密码，避免敏感信息泄露
	passwordMasked := ""
	if cfg.Password != "" {
		passwordMasked = "******"
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"host":      cfg.Host,
			"port":      cfg.Port,
			"user":      cfg.User,
			"password":  passwordMasked,
			"dbName":    cfg.DBName,
			"connected": h.isConnected(),
		},
	})
}

func (h *DBConfigHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		Host     string `json:"host" binding:"required"`
		Port     string `json:"port" binding:"required"`
		User     string `json:"user" binding:"required"`
		Password string `json:"password"`
		DBName   string `json:"dbName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	// 若前端提交的是占位密码（"******"），保留原密码
	password := req.Password
	if password == "" || password == "******" {
		password = h.DBStore.GetConfig().Password
	}

	cfg := store.DBConfig{
		Host:     req.Host,
		Port:     req.Port,
		User:     req.User,
		Password: password,
		DBName:   req.DBName,
	}

	if err := h.DBStore.Reconnect(h.CfgFile, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "连接数据库失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "数据库配置已更新并重新连接成功",
	})
}

func (h *DBConfigHandler) isConnected() bool {
	db := h.DBStore.GetDB()
	if db == nil {
		return false
	}
	return db.Ping() == nil
}

func scanUserRow(row *sql.Row) (models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Avatar,
		&u.Phone, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}
