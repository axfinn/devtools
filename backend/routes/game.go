package routes

import "github.com/gin-gonic/gin"

func RegisterGameRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	game := api.Group("/game")
	{
		game.POST("/arcade/rooms", h.GameHandler.CreateArcadeRoom)
		game.POST("/arcade/rooms/:id/join", h.GameHandler.JoinArcadeRoom)
		game.GET("/arcade/rooms/:id", h.GameHandler.GetArcadeRoom)
		game.GET("/arcade/ws/:id", h.GameHandler.ArcadeWS)
	}
}
