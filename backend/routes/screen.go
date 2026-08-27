package routes

import "github.com/gin-gonic/gin"

func RegisterScreenRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	screen := api.Group("/screen")
	{
		screen.GET("/turn-credentials", h.ScreenHandler.ScreenTurnCredentials)
		screen.GET("/sessions/mine", h.ScreenHandler.ScreenMySessions)
		screen.POST("/sessions", h.ScreenHandler.ScreenCreate)
		screen.GET("/sessions/:id/info", h.ScreenHandler.ScreenInfo)
		screen.POST("/sessions/:id/check-password", h.ScreenHandler.ScreenCheckPassword)
		screen.DELETE("/sessions/:id", h.ScreenHandler.ScreenStop)
		screen.GET("/sessions/:id/ws", h.ScreenHandler.ScreenSignalingWS)
		screen.GET("/sessions/:id/relay-host", h.ScreenHandler.ScreenRelayHostWS)
		screen.GET("/sessions/:id/relay-viewer", h.ScreenHandler.ScreenRelayViewerWS)
	}
}
