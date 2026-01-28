package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"devtools/utils"

	"github.com/gin-gonic/gin"
)

// ErrorHandler 全局错误处理中间件
// 捕获 panic 并转换为统一的错误响应
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录 panic 堆栈信息
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())

				// 返回内部错误响应
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":  http.StatusInternalServerError,
					"error": "内部服务器错误",
				})

				c.Abort()
			}
		}()

		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// 如果是 AppError，使用统一的错误响应
			if appErr, ok := err.(*utils.AppError); ok {
				if !c.Writer.Written() {
					c.JSON(appErr.Code, appErr)
				}
			} else {
				// 其他错误，返回通用内部错误
				if !c.Writer.Written() {
					c.JSON(http.StatusInternalServerError, utils.ErrInternal)
				}
			}

			c.Abort()
		}
	}
}
