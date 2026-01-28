package utils

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AppError 应用程序错误类型
type AppError struct {
	Code    int    `json:"code"`           // HTTP 状态码
	Message string `json:"error"`          // 用户友好的错误信息
	Detail  string `json:"detail,omitempty"` // 详细错误信息（可选）
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Detail)
	}
	return e.Message
}

// NewAppError 创建新的应用错误
func NewAppError(code int, message string, detail string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Detail:  detail,
	}
}

// 预定义的常见错误
var (
	ErrInvalidInput       = &AppError{http.StatusBadRequest, "无效的输入数据", ""}
	ErrMissingContent     = &AppError{http.StatusBadRequest, "请输入内容", ""}
	ErrContentTooLarge    = &AppError{http.StatusBadRequest, "内容过大", ""}
	ErrInvalidPassword    = &AppError{http.StatusUnauthorized, "密码错误", ""}
	ErrUnauthorized       = &AppError{http.StatusUnauthorized, "未授权访问", ""}
	ErrNotFound           = &AppError{http.StatusNotFound, "资源不存在", ""}
	ErrExpired            = &AppError{http.StatusGone, "资源已过期", ""}
	ErrTooManyRequests    = &AppError{http.StatusTooManyRequests, "请求过于频繁，请稍后再试", ""}
	ErrForbidden          = &AppError{http.StatusForbidden, "禁止访问", ""}
	ErrInternal           = &AppError{http.StatusInternalServerError, "内部服务器错误", ""}
	ErrDatabaseError      = &AppError{http.StatusInternalServerError, "数据库操作失败", ""}
	ErrPasswordHashFailed = &AppError{http.StatusInternalServerError, "密码处理失败", ""}
)

// RespondError 统一的错误响应函数
func RespondError(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.Code, appErr)
	} else {
		// 未知错误，返回通用内部错误
		c.JSON(http.StatusInternalServerError, ErrInternal)
	}
}

// RespondSuccess 统一的成功响应函数
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// RespondCreated 统一的创建成功响应函数
func RespondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}
