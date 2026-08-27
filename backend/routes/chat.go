package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	chat := api.Group("/chat")
	{
		chat.POST("/room", createRateLimiter.Middleware(), h.ChatHandler.CreateRoom)
		chat.GET("/rooms", h.ChatHandler.GetRooms)
		chat.GET("/room/:id", h.ChatHandler.GetRoom)
		chat.GET("/room/:id/messages", h.ChatHandler.GetRoomMessages)
		chat.POST("/room/:id/join", h.ChatHandler.JoinRoom)
		chat.GET("/room/:id/ws", h.ChatHandler.HandleWebSocket)
		chat.POST("/upload", createRateLimiter.Middleware(), h.ChatHandler.UploadImage)
		chat.Static("/uploads", "./data/uploads")
		chat.GET("/admin/rooms", h.ChatHandler.AdminListRooms)
		chat.DELETE("/admin/room/:id", h.ChatHandler.AdminDeleteRoom)
		chat.GET("/room/:id/bot", h.ChatHandler.GetBotConfig)
		chat.POST("/room/:id/bot", h.ChatHandler.AddBot)
		chat.DELETE("/room/:id/bot", h.ChatHandler.RemoveBot)
		chat.POST("/room/:id/bot/stop", h.ChatHandler.StopBot)
	}
}
