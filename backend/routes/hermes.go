package routes

import "github.com/gin-gonic/gin"

func RegisterHermesRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	hermes := api.Group("/hermes")
	{
		hermes.POST("/verify", h.HermesHandler.VerifyPassword)
		hermes.GET("/status", h.HermesHandler.Status)
		hermes.GET("/models", h.HermesHandler.Models)
		hermes.POST("/chat", h.HermesHandler.Chat)
	}
}
