package routes

import (
	"devtools/handlers"
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterOCRRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	api.POST("/ocr", createRateLimiter.Middleware(), handlers.NewOCRHandler().Extract)
}
