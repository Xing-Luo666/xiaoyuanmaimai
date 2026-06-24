package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"school-trade/middleware"
	"school-trade/models"
	"school-trade/store"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func genID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

type AuthHandler struct {
	Store *store.DBStore
}

func NewAuthHandler(s *store.DBStore) *AuthHandler {
	return &AuthHandler{Store: s}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	// 用户名校验：4-20位字母或数字
	if len(req.Username) < 4 || len(req.Username) > 20 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "用户名需为4-20位字母或数字"})
		return
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(req.Username) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "用户名只能包含字母和数字"})
		return
	}

	// 密码校验：至少6位
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "密码至少需要6位"})
		return
	}

	// 手机号校验（如果填写了）
	if req.Phone != "" {
		if !regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(req.Phone) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "手机号格式不正确"})
			return
		}
	}

	// 邮箱校验（如果填写了）
	if req.Email != "" {
		if !regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`).MatchString(req.Email) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "邮箱格式不正确"})
			return
		}
	}

	// 昵称校验
	if len(req.Nickname) < 1 || len(req.Nickname) > 30 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "昵称长度为1-30个字符"})
		return
	}

	db := h.Store.GetDB()

	var existUsername string
	err := db.QueryRow("SELECT username FROM users WHERE username = ?", req.Username).Scan(&existUsername)
	if err == nil {
		c.JSON(http.StatusConflict, models.APIResponse{Code: 409, Message: "用户名已存在"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "注册失败"})
		return
	}

	now := time.Now()
	user := models.User{
		ID:        genID("u"),
		Username:  req.Username,
		Password:  string(hashedPassword),
		Nickname:  req.Nickname,
		Phone:     req.Phone,
		Email:     req.Email,
		Role:      "student",
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = db.Exec(
		"INSERT INTO users (id, username, password, nickname, avatar, phone, email, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		user.ID, user.Username, user.Password, user.Nickname, user.Avatar, user.Phone, user.Email, user.Role, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "保存用户失败"})
		return
	}

	token, expiresAt, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "生成令牌失败"})
		return
	}

	// 设置 HttpOnly Cookie
	c.SetCookie("sso_token", token, int(time.Until(time.Unix(expiresAt, 0)).Seconds()), "/", "", false, true)

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "注册成功",
		Data: models.SSOToken{
			Token:     token,
			UserID:    user.ID,
			Username:  user.Username,
			Role:      user.Role,
			ExpiresAt: expiresAt,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	db := h.Store.GetDB()

	user, err := scanUserRow(db.QueryRow("SELECT id, username, password, nickname, avatar, phone, email, role, created_at, updated_at FROM users WHERE username = ?", req.Username))
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Code: 401, Message: "用户名或密码错误"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Code: 401, Message: "用户名或密码错误"})
		return
	}

	token, expiresAt, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "生成令牌失败"})
		return
	}

	// 设置 HttpOnly Cookie
	c.SetCookie("sso_token", token, int(time.Until(time.Unix(expiresAt, 0)).Seconds()), "/", "", false, true)

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "登录成功",
		Data: models.SSOToken{
			Token:     token,
			UserID:    user.ID,
			Username:  user.Username,
			Role:      user.Role,
			ExpiresAt: expiresAt,
		},
	})
}

func (h *AuthHandler) VerifyToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	claims, err := middleware.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Code: 401, Message: "令牌无效"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "令牌有效",
		Data: gin.H{
			"userId":   claims.UserID,
			"username": claims.Username,
			"role":     claims.Role,
		},
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Code: 401, Message: "未登录"})
		return
	}

	db := h.Store.GetDB()
	user, err := scanUserRow(db.QueryRow("SELECT id, username, password, nickname, avatar, phone, email, role, created_at, updated_at FROM users WHERE id = ?", userID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "用户不存在"})
		return
	}

	user.Password = ""
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "成功", Data: user})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userId")

	var req struct {
		Nickname string `json:"nickname"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	db := h.Store.GetDB()

	user, err := scanUserRow(db.QueryRow("SELECT id, username, password, nickname, avatar, phone, email, role, created_at, updated_at FROM users WHERE id = ?", userID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "用户不存在"})
		return
	}

	now := time.Now()
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Phone != "" {
		if !regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(req.Phone) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "手机号格式不正确"})
			return
		}
		user.Phone = req.Phone
	}
	if req.Email != "" {
		if !regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`).MatchString(req.Email) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "邮箱格式不正确"})
			return
		}
		user.Email = req.Email
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	user.UpdatedAt = now

	_, err = db.Exec("UPDATE users SET nickname=?, phone=?, email=?, avatar=?, updated_at=? WHERE id=?",
		user.Nickname, user.Phone, user.Email, user.Avatar, user.UpdatedAt, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新失败"})
		return
	}

	user.Password = ""
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "更新成功", Data: user})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("userId")

	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "新密码至少6位"})
		return
	}

	db := h.Store.GetDB()
	var hashedPwd string
	err := db.QueryRow("SELECT password FROM users WHERE id = ?", userID).Scan(&hashedPwd)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "用户不存在"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "原密码错误"})
		return
	}

	newHashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "密码加密失败"})
		return
	}

	_, err = db.Exec("UPDATE users SET password = ?, updated_at = ? WHERE id = ?", string(newHashed), time.Now(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "修改失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "密码修改成功"})
}
