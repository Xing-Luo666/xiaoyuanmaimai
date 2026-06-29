package utils

import (
	"math/rand"
	"strings"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateAppID 生成AppId（KK790 + 10位随机字符）
func GenerateAppID() string {
	return "KK790" + randomString(11)
}

// GenerateAppSecret 生成32位Secret
func GenerateAppSecret() string {
	return randomString(32)
}

// GenerateSM4Secret 生成SM4秘钥（24位base64格式）
func GenerateSM4Secret() string {
	return randomString(24) + "=="
}

// GenerateAuthCode 生成OAuth2授权码（32位）
func GenerateAuthCode() string {
	return randomString(32)
}

// GenerateAccessToken 生成Access Token（64位）
func GenerateAccessToken() string {
	return randomString(64)
}

// MaskString 字符串脱敏（保留首尾n位）
func MaskString(s string, head, tail int) string {
	if len(s) <= head+tail {
		return strings.Repeat("*", len(s))
	}
	return s[:head] + strings.Repeat("*", len(s)-head-tail) + s[len(s)-tail:]
}

// MaskPhone 手机号脱敏：保留前3后4
func MaskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// MaskIDCard 身份证脱敏：保留前6后4
func MaskIDCard(idCard string) string {
	if len(idCard) < 10 {
		return idCard
	}
	return idCard[:6] + "********" + idCard[len(idCard)-4:]
}

// MaskName 姓名脱敏：保留姓
func MaskName(name string) string {
	if len(name) <= 1 {
		return name
	}
	return string(name[0]) + strings.Repeat("*", len(name)-1)
}

func randomString(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[r.Intn(len(letterBytes))]
	}
	return string(b)
}
