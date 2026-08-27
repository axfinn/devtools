package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterMockAPIRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	mockapi := api.Group("/mockapi")
	{
		mockapi.POST("", createRateLimiter.Middleware(), h.MockAPIHandler.Create)
		mockapi.GET("/:id", h.MockAPIHandler.Get)
		mockapi.GET("/:id/logs", h.MockAPIHandler.GetLogs)
		mockapi.PUT("/:id", h.MockAPIHandler.Update)
		mockapi.DELETE("/:id", h.MockAPIHandler.Delete)
	}
}
