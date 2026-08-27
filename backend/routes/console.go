package routes

import "github.com/gin-gonic/gin"

func RegisterConsoleRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	api.GET("/console/settings", h.ConsoleHandler.GetSettings)
	api.POST("/console/settings", h.ConsoleHandler.SaveSettings)
	api.POST("/console/verify", h.ConsoleHandler.VerifyPassword)
}
