package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterShortURLRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	shorturl := api.Group("/shorturl")
	{
		shorturl.POST("", createRateLimiter.Middleware(), h.ShortURLHandler.Create)
		shorturl.GET("/list", h.ShortURLHandler.List)
		shorturl.GET("/:id/stats", h.ShortURLHandler.GetStats)
		shorturl.PUT("/:id", h.ShortURLHandler.Update)
		shorturl.DELETE("/:id", h.ShortURLHandler.Delete)
	}
}
