package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// R 统一响应结构（仿若依 {code, msg, data}）
type R struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// AjaxResult 兼容若依命名
type AjaxResult = R

const (
	CodeSuccess      = 200
	CodeError        = 500
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeBadRequest   = 400
)

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, R{Code: CodeSuccess, Msg: "操作成功", Data: data})
}

// SuccessMsg 成功响应（自定义消息）
func SuccessMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, R{Code: CodeSuccess, Msg: msg, Data: data})
}

// Error 失败响应
func Error(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, R{Code: CodeError, Msg: msg})
}

// Errorcode 失败响应（自定义code）
func Errorcode(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, R{Code: code, Msg: msg})
}

// BadRequest 参数错误
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, R{Code: CodeBadRequest, Msg: msg})
}

// Unauthorized 未登录
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, R{Code: CodeUnauthorized, Msg: msg})
}

// Forbidden 无权限
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, R{Code: CodeForbidden, Msg: msg})
}

// PageResult 分页结果
type PageResult struct {
	Total int64       `json:"total"`
	Rows  interface{} `json:"rows"`
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
}

// SuccessPage 分页响应
func SuccessPage(c *gin.Context, total int64, rows interface{}) {
	c.JSON(http.StatusOK, PageResult{Total: total, Rows: rows, Code: CodeSuccess, Msg: "查询成功"})
}
