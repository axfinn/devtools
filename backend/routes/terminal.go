package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterTerminalRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	terminal := api.Group("/terminal")
	{
		terminal.POST("", createRateLimiter.Middleware(), h.TerminalHandler.Create)
		terminal.POST("/login", h.TerminalHandler.Login)
		terminal.GET("/list", h.TerminalHandler.List)
		terminal.GET("/admin/list", h.TerminalHandler.AdminList)
		terminal.DELETE("/admin/:id", h.TerminalHandler.AdminDelete)
		terminal.GET("/:id", h.TerminalHandler.Get)
		terminal.GET("/:id/creator", h.TerminalHandler.GetByCreator)
		terminal.GET("/:id/history", h.TerminalHandler.GetHistory)
		terminal.POST("/:id/resume", h.TerminalHandler.Resume)
		terminal.POST("/:id/disconnect", h.TerminalHandler.Disconnect)
		terminal.PUT("/:id", h.TerminalHandler.Update)
		terminal.DELETE("/:id", h.TerminalHandler.Delete)
		terminal.GET("/:id/ws", h.TerminalHandler.HandleWebSocket)
	}
}
