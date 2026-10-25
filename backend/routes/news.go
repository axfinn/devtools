package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterNewsRoutes 新闻模块接口(均在 /api/news 下)。
func RegisterNewsRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	news := api.Group("/news")
	{
		news.GET("/latest", h.NewsHandler.GetLatest)
		news.GET("/list", h.NewsHandler.ListDates)
		news.GET("/config", h.NewsHandler.GetConfig)
		news.GET("/:date", h.NewsHandler.GetByDate)
		news.POST("/generate", createRateLimiter.Middleware(), h.NewsHandler.Generate)
		// 智能体(WorkBuddy 本地定时任务)采编后发布/归档的入口。
		news.POST("/publish", createRateLimiter.Middleware(), h.NewsHandler.Publish)
	}
}
