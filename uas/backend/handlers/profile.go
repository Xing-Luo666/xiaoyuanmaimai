package handlers

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	"uas/config"
	"uas/middleware"
	"uas/store"
	"uas/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ProfileHandler 个人中心
type ProfileHandler struct {
	store *store.Store
	cfg   *config.Config
}

func NewProfileHandler(s *store.Store, cfg *config.Config) *ProfileHandler {
	return &ProfileHandler{store: s, cfg: cfg}
}

// UpdateProfile 修改个人信息
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Sex      int    `json:"sex"`
	}
	c.ShouldBindJSON(&req)

	db := h.store.GetDB()
	_, err := db.Exec(
		"UPDATE sys_user SET nickname=?, email=?, phone=?, sex=? WHERE id=?",
		req.Nickname, req.Email, req.Phone, req.Sex, userID,
	)
	if err != nil {
		utils.Error(c, "修改失败")
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

// GetProfile 获取个人信息
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := h.store.GetDB()

	var (
		id         int64
		username   string
		nickname   string
		email      string
		phone      string
		sex        int
		avatar     string
		deptID     sql.NullInt64
		deptName   string
		roleName   string
		createTime string
	)
	err := db.QueryRow(`
		SELECT u.id, u.username, IFNULL(u.nickname,''), IFNULL(u.email,''), IFNULL(u.phone,''), u.sex, IFNULL(u.avatar,''), u.dept_id,
		       IFNULL(d.dept_name, ''), '管理员', u.create_time
		FROM sys_user u
		LEFT JOIN sys_dept d ON u.dept_id = d.id
		WHERE u.id = ?
	`, userID).Scan(&id, &username, &nickname, &email, &phone, &sex, &avatar, &deptID, &deptName, &roleName, &createTime)
	if err != nil {
		utils.Error(c, "查询失败")
		return
	}

	var deptIDPtr *int64
	if deptID.Valid {
		v := deptID.Int64
		deptIDPtr = &v
	}

	utils.Success(c, gin.H{
		"id":         id,
		"username":   username,
		"nickname":   nickname,
		"email":      email,
		"phone":      phone,
		"sex":        sex,
		"avatar":     avatar,
		"deptId":     deptIDPtr,
		"deptName":   deptName,
		"roleName":   roleName,
		"createTime": createTime,
		"loginDate":  "",
	})
}

// ChangePassword 修改密码
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	db := h.store.GetDB()
	var passwordHash string
	err := db.QueryRow("SELECT password FROM sys_user WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		utils.Error(c, "用户不存在")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.OldPassword)); err != nil {
		utils.Error(c, "原密码错误")
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	_, err = db.Exec("UPDATE sys_user SET password = ? WHERE id = ?", string(hash), userID)
	if err != nil {
		utils.Error(c, "修改失败")
		return
	}
	utils.SuccessMsg(c, "修改成功", nil)
}

// UploadAvatar 上传头像
func (h *ProfileHandler) UploadAvatar(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "请选择文件")
		return
	}

	// 校验文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		utils.BadRequest(c, "仅支持 jpg/jpeg/png/gif 格式")
		return
	}

	// 限制大小（2MB）
	if file.Size > 2*1024*1024 {
		utils.BadRequest(c, "文件大小不能超过2MB")
		return
	}

	// 保存
	filename := "avatar_" + time.Now().Format("20060102150405") + ext
	savePath := filepath.Join("uploads", "avatar", filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		utils.Error(c, "上传失败: "+err.Error())
		return
	}

	// 校验文件是否存在
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		utils.Error(c, "文件保存失败")
		return
	}

	avatarURL := "/uploads/avatar/" + filename
	userID := middleware.GetUserID(c)
	db := h.store.GetDB()
	_, _ = db.Exec("UPDATE sys_user SET avatar = ? WHERE id = ?", avatarURL, userID)

	utils.SuccessMsg(c, "上传成功", gin.H{"avatar": avatarURL})
}
