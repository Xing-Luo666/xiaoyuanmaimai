package handlers

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"school-trade/models"
	"school-trade/store"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	Store *store.DBStore
}

const maxAvatarBytes = 512 * 1024 // 512KB

// UploadAvatar 上传用户头像
// 接收前端裁剪后的方形图片（base64 或 multipart），后端再压缩到 512KB 以内
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetString("userId")

	var imgData []byte
	var ext string

	// 优先解析 multipart 文件
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		if header.Size > 5*1024*1024 {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "原始图片不能超过5MB"})
			return
		}
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(file); err != nil {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "读取图片失败"})
			return
		}
		imgData = buf.Bytes()
		ext = strings.ToLower(filepath.Ext(header.Filename))
	} else {
		// 兼容 base64 JSON 上传
		var req struct {
			Image    string `json:"image"` // dataURL 或纯 base64
			ImageB64 string `json:"imageB64"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			raw := req.Image
			if raw == "" {
				raw = req.ImageB64
			}
			if raw == "" {
				c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "请上传图片"})
				return
			}
			// 处理 dataURL 前缀
			if strings.HasPrefix(raw, "data:") {
				if idx := strings.Index(raw, ";base64,"); idx > 0 {
					ext = "." + strings.TrimPrefix(raw[5:idx], "image/")
					raw = raw[idx+8:]
				} else {
					c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "dataURL 格式错误"})
					return
				}
			}
			b, err := base64.StdEncoding.DecodeString(raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "base64 解码失败"})
				return
			}
			imgData = b
		} else {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "请上传图片"})
			return
		}
	}

	if ext == "" {
		ext = ".png"
	}
	if !map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true, ".bmp": true}[ext] {
		ext = ".png"
	}

	// 解码为 image.Image
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "图片解码失败: " + err.Error()})
		return
	}

	// 压缩到 512KB 以内（逐步降低 JPEG 质量）
	compressed, finalExt, err := compressImageToSize(img, maxAvatarBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "图片压缩失败: " + err.Error()})
		return
	}

	// 保存到 resources/avatars/
	uploadDir := filepath.Join("..", "frontend", "resources", "avatars")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		// 兼容从 backend 目录直接运行的情况
		altDir := filepath.Join("frontend", "resources", "avatars")
		if _, e := os.Stat(altDir); e == nil || !os.IsNotExist(e) {
			uploadDir = altDir
		}
	}
	os.MkdirAll(uploadDir, 0755)

	fileName := fmt.Sprintf("avatar_%s_%d%s", userID, time.Now().UnixNano(), finalExt)
	savePath := filepath.Join(uploadDir, fileName)
	if err := os.WriteFile(savePath, compressed, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "保存头像失败"})
		return
	}

	// 返回 URL
	avatarURL := "/resources/avatars/" + fileName

	// 更新 users 表
	db := h.Store.GetDB()
	if _, err := db.Exec("UPDATE users SET avatar = ?, updated_at = ? WHERE id = ?", avatarURL, time.Now(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新头像失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "上传成功", Data: models.AvatarUploadResponse{URL: avatarURL}})
}

// GetProfile 获取当前用户信息（含头像）
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()

	var u models.User
	var avatar sql.NullString
	err := db.QueryRow("SELECT id, username, nickname, COALESCE(avatar, ''), phone, COALESCE(email, ''), role, created_at FROM users WHERE id = ?", userID).
		Scan(&u.ID, &u.Username, &u.Nickname, &avatar, &u.Phone, &u.Email, &u.Role, &u.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "用户不存在"})
		return
	}
	if avatar.Valid {
		u.Avatar = avatar.String
	}
	if u.Avatar == "" {
		u.Avatar = "/resources/default-avatar.svg" // 默认头像
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: u})
}

// compressImageToSize 压缩图片到指定大小以内，返回 (字节数组, 扩展名, err)
// 策略：1) PNG 无损；2) 不够则转 JPEG 质量 90→70→50→30→10；3) 仍超限则缩小尺寸
func compressImageToSize(img image.Image, maxBytes int) ([]byte, string, error) {
	// 先尝试 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err == nil && buf.Len() <= maxBytes {
		return buf.Bytes(), ".png", nil
	}

	// JPEG 渐进质量
	qualities := []int{90, 80, 70, 60, 50, 40, 30, 20, 10}
	for _, q := range qualities {
		buf.Reset()
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			continue
		}
		if buf.Len() <= maxBytes {
			return buf.Bytes(), ".jpg", nil
		}
	}

	// 仍超限：缩小尺寸再试
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	scaleFactors := []float64{0.8, 0.6, 0.5, 0.4, 0.3, 0.2}
	for _, s := range scaleFactors {
		newW, newH := int(float64(w)*s), int(float64(h)*s)
		if newW < 32 || newH < 32 {
			break
		}
		resized := resizeImage(img, newW, newH)
		for _, q := range []int{70, 50, 30} {
			buf.Reset()
			if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: q}); err == nil && buf.Len() <= maxBytes {
				return buf.Bytes(), ".jpg", nil
			}
		}
	}

	// 最后兜底：用最低质量 JPEG
	buf.Reset()
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 10}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), ".jpg", nil
}

// resizeImage 简单近邻采样缩放
func resizeImage(src image.Image, newW, newH int) image.Image {
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		sy := y * srcH / newH
		for x := 0; x < newW; x++ {
			sx := x * srcW / newW
			dst.Set(x, y, src.At(bounds.Min.X+sx, bounds.Min.Y+sy))
		}
	}
	return dst
}
