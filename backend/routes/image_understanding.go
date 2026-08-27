package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterImageUnderstandingRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	img := api.Group("/image-understanding")
	{
		img.GET("/tools", h.ImageUnderstandingHandler.ListTools)
		img.POST("/describe", createRateLimiter.Middleware(), h.ImageUnderstandingHandler.Describe)
		img.POST("/describe-file", createRateLimiter.Middleware(), h.ImageUnderstandingHandler.DescribeFromUpload)
		img.POST("/sse/create", createRateLimiter.Middleware(), h.ImageUnderstandingHandler.CreateSseTask)
		img.POST("/sse/create-file", createRateLimiter.Middleware(), h.ImageUnderstandingHandler.CreateSseTaskFromFile)
		img.GET("/sse/task/:id", h.ImageUnderstandingHandler.GetSseTask)
		img.GET("/sse/stream/:id", h.ImageUnderstandingHandler.StreamSseTask)
		img.POST("/qwen-vision", createRateLimiter.Middleware(), h.AiGatewayHandler.InternalQwenVision)
		img.GET("/qwen-vision/logs", h.AiGatewayHandler.AdminListQwenVisionLogs)
		img.POST("/minimax-vision", createRateLimiter.Middleware(), h.AiGatewayHandler.InternalMinimaxVision)
	}
}
