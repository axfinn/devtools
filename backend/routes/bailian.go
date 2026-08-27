package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterBailianRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	bailian := api.Group("/bailian")
	{
		bailian.GET("/docs", h.BailianHandler.GetDocs)
		bailian.GET("/models", h.BailianHandler.GetModels)
		bailian.POST("/tasks", createRateLimiter.Middleware(), h.BailianHandler.CreateTask)
		bailian.GET("/tasks", h.BailianHandler.ListTasks)
		bailian.GET("/tasks/:id/events", h.BailianHandler.GetTaskEvents)
		bailian.POST("/tasks/:id/poll", h.BailianHandler.PollTask)
		bailian.GET("/tasks/:id", h.BailianHandler.GetTask)
		bailian.POST("/generate", createRateLimiter.Middleware(), h.BailianHandler.OpenAPICreateTask)
	}
}
