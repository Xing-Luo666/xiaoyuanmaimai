package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"uas/config"
	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// OAuthHandler OAuth2 授权码模式
type OAuthHandler struct {
	store *store.Store
	cfg   *config.Config
	// 内存存储授权码和token（生产环境建议用Redis）
	codes  map[string]*AuthCode
	tokens map[string]*AccessToken
}

// AuthCode 授权码
type AuthCode struct {
	Code      string
	AppID     string
	UserID    int64
	UserType  string
	Scope     string
	Redirect  string
	ExpiresAt time.Time
}

// AccessToken 访问令牌
type AccessToken struct {
	Token     string
	AppID     string
	UserID    int64
	UserType  string
	Scope     string
	ExpiresAt time.Time
}

func NewOAuthHandler(s *store.Store, cfg *config.Config) *OAuthHandler {
	return &OAuthHandler{
		store:  s,
		cfg:    cfg,
		codes:  make(map[string]*AuthCode),
		tokens: make(map[string]*AccessToken),
	}
}

// Authorize 授权端点
// GET /api/oauth/authorize?client_id=xxx&redirect_uri=xxx&response_type=code&state=xxx&scope=xxx
func (h *OAuthHandler) Authorize(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	responseType := c.Query("response_type")
	state := c.Query("state")
	scope := c.DefaultQuery("scope", "userinfo")

	if responseType != "code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_response_type", "error_description": "仅支持 code 模式"})
		return
	}

	if clientID == "" || redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "client_id 和 redirect_uri 必填"})
		return
	}

	// 校验应用
	appHandler := NewAppHandler(h.store, h.cfg)
	if _, err := appHandler.VerifyApp(clientID, redirectURI); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client", "error_description": "应用不存在或回调地址不匹配"})
		return
	}

	// 查询应用名称
	var appName, appType, description string
	db := h.store.GetDB()
	db.QueryRow("SELECT app_name, app_type, description FROM u_app WHERE app_id = ?", clientID).
		Scan(&appName, &appType, &description)

	// 返回授权页所需信息，前端展示授权确认页
	utils.Success(c, gin.H{
		"appId":       clientID,
		"appName":     appName,
		"appType":     appType,
		"redirectUri": redirectURI,
		"scope":       scope,
		"state":       state,
		"description": description,
	})
}

// AuthorizeConfirm 用户确认授权，返回code
// POST /api/oauth/authorize
func (h *OAuthHandler) AuthorizeConfirm(c *gin.Context) {
	var req struct {
		ClientID    string `json:"client_id" binding:"required"`
		RedirectURI string `json:"redirect_uri" binding:"required"`
		State       string `json:"state"`
		Scope       string `json:"scope"`
		UserID      int64  `json:"user_id" binding:"required"`
		UserType    string `json:"user_type"`
		// 兼容直接传入 token 的方式
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}

	if req.UserType == "" {
		req.UserType = "personal"
	}

	// 校验应用
	appHandler := NewAppHandler(h.store, h.cfg)
	_, err := appHandler.VerifyApp(req.ClientID, req.RedirectURI)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	// 生成授权码
	code := utils.GenerateAuthCode()
	h.codes[code] = &AuthCode{
		Code:      code,
		AppID:     req.ClientID,
		UserID:    req.UserID,
		UserType:  req.UserType,
		Scope:     req.Scope,
		Redirect:  req.RedirectURI,
		ExpiresAt: time.Now().Add(time.Duration(h.cfg.OAuth2.CodeExpireSeconds) * time.Second),
	}

	// 记录授权
	grantHandler := NewGrantHandler(h.store)
	if !grantHandler.CheckGrant(req.UserID, req.UserType, req.ClientID) {
		grantHandler.CreateGrant(req.UserID, req.UserType, req.ClientID)
	}

	// 拼接回调URL
	redirectURL := req.RedirectURI
	sep := "?"
	if strings.Contains(redirectURL, "?") {
		sep = "&"
	}
	redirectURL += sep + "code=" + code
	if req.State != "" {
		redirectURL += "&state=" + req.State
	}

	utils.Success(c, gin.H{
		"code":         code,
		"redirect_url": redirectURL,
		"expires_in":   h.cfg.OAuth2.CodeExpireSeconds,
	})
}

// Token Token端点
// POST /api/oauth/token
func (h *OAuthHandler) Token(c *gin.Context) {
	grantType := c.PostForm("grant_type")
	code := c.PostForm("code")
	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")
	redirectURI := c.PostForm("redirect_uri")

	if grantType != "authorization_code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	if code == "" || clientID == "" || clientSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "参数缺失"})
		return
	}

	// 校验应用密钥
	db := h.store.GetDB()
	var dbID int64
	var dbSecret string
	var status int
	err := db.QueryRow(
		"SELECT id, app_secret, status FROM u_app WHERE app_id = ? AND del_flag = 0",
		clientID,
	).Scan(&dbID, &dbSecret, &status)

	if err != nil || status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	if dbSecret != clientSecret {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client", "error_description": "密钥不匹配"})
		return
	}

	// 校验授权码
	authCode, ok := h.codes[code]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "授权码不存在"})
		return
	}

	if time.Now().After(authCode.ExpiresAt) {
		delete(h.codes, code)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "授权码已过期"})
		return
	}

	if authCode.AppID != clientID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "授权码与应用不匹配"})
		return
	}

	if redirectURI != "" && redirectURI != authCode.Redirect {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "redirect_uri 不一致"})
		return
	}

	// 生成 access_token
	token := utils.GenerateAccessToken()
	h.tokens[token] = &AccessToken{
		Token:     token,
		AppID:     clientID,
		UserID:    authCode.UserID,
		UserType:  authCode.UserType,
		Scope:     authCode.Scope,
		ExpiresAt: time.Now().Add(time.Duration(h.cfg.OAuth2.TokenExpireSeconds) * time.Second),
	}

	// 授权码一次性使用
	delete(h.codes, code)

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   h.cfg.OAuth2.TokenExpireSeconds,
		"scope":        authCode.Scope,
	})
}

// UserInfo 用户信息端点
// GET /api/oauth/userinfo
func (h *OAuthHandler) UserInfo(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || token == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	at, ok := h.tokens[token]
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	if time.Now().After(at.ExpiresAt) {
		delete(h.tokens, token)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "error_description": "token expired"})
		return
	}

	// 查询用户信息
	db := h.store.GetDB()
	if at.UserType == "personal" {
		var phone, realName, idCard, nickname string
		var email, authLevel, avatar sql.NullString
		err := db.QueryRow(
			"SELECT phone, real_name, id_card_no, nickname, email, auth_level, avatar FROM u_user WHERE id = ? AND del_flag = 0",
			at.UserID,
		).Scan(&phone, &realName, &idCard, &nickname, &email, &authLevel, &avatar)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_not_found", "error_description": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sub":        fmt.Sprintf("u_%d", at.UserID),
			"user_id":    at.UserID,
			"user_type":  "personal",
			"phone":      phone,
			"real_name":  realName,
			"nickname":   nickname,
			"email":      email.String,
			"avatar":     avatar.String,
			"auth_level": authLevel.String,
			"app_id":     at.AppID,
		})
		return
	}

	// 法人用户
	var username, corpName, creditCode, legalPerson, phone string
	err := db.QueryRow(
		"SELECT username, corp_name, credit_code, legal_person, phone FROM u_corp_user WHERE id = ? AND del_flag = 0",
		at.UserID,
	).Scan(&username, &corpName, &creditCode, &legalPerson, &phone)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_not_found", "error_description": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sub":          fmt.Sprintf("c_%d", at.UserID),
		"user_id":      at.UserID,
		"user_type":    "corp",
		"username":     username,
		"corp_name":    corpName,
		"credit_code":  creditCode,
		"legal_person": legalPerson,
		"phone":        phone,
		"app_id":       at.AppID,
	})
}

// UASLogin UAS用户登录（OAuth2 流程中使用）
// POST /api/uas/login
func (h *OAuthHandler) UASLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"` // 手机号
		Password string `json:"password" binding:"required"`
		UserType string `json:"userType"` // personal / corp，默认 personal
		Code     string `json:"code"`     // 图形验证码
		UUID     string `json:"uuid"`     // 验证码ID
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "账号密码必填")
		return
	}

	// 校验图形验证码，防止暴力登录
	if !verifyCaptcha(req.UUID, req.Code) {
		utils.BadRequest(c, "验证码错误或已过期")
		return
	}

	if req.UserType == "" {
		req.UserType = "personal"
	}

	db := h.store.GetDB()
	var userID int64
	var passwordHash string
	var status int
	var nickname, realName string

	if req.UserType == "personal" {
		err := db.QueryRow(
			"SELECT id, password, status, COALESCE(nickname, real_name), COALESCE(real_name, '') FROM u_user WHERE phone = ? AND del_flag = 0",
			req.Username,
		).Scan(&userID, &passwordHash, &status, &nickname, &realName)
		if err == sql.ErrNoRows {
			utils.Error(c, "账号或密码错误")
			return
		}
		if err != nil {
			utils.Error(c, "查询失败")
			return
		}
	} else {
		err := db.QueryRow(
			"SELECT id, password, status, COALESCE(corp_name, username), '' FROM u_corp_user WHERE username = ? AND del_flag = 0",
			req.Username,
		).Scan(&userID, &passwordHash, &status, &nickname, &realName)
		if err == sql.ErrNoRows {
			utils.Error(c, "账号或密码错误")
			return
		}
		if err != nil {
			utils.Error(c, "查询失败")
			return
		}
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		utils.Error(c, "账号或密码错误")
		return
	}

	if status != 1 {
		utils.Error(c, "账号已禁用")
		return
	}

	// 生成临时token（用于授权页确认身份）
	sessionToken := uuid.New().String()
	// 简化：直接返回userID和token，前端在确认授权时传入user_id
	utils.Success(c, gin.H{
		"token":    sessionToken,
		"userId":   userID,
		"userType": req.UserType,
		"nickname": nickname,
		"realName": realName,
	})
}

// UASUserInfo 获取UAS用户信息（通过sessionToken）
func (h *OAuthHandler) UASUserInfo(c *gin.Context) {
	// 此接口可扩展，目前简单返回
	utils.Success(c, gin.H{"ok": true})
}

// ParseTokenForDebug 内部调试用
func (h *OAuthHandler) parseTokenForDebug(token string) interface{} {
	at, ok := h.tokens[token]
	if !ok {
		return nil
	}
	b, _ := json.Marshal(at)
	return string(b)
}
