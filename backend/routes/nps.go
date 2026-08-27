package routes

import "github.com/gin-gonic/gin"

func RegisterNPSRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	nps := api.Group("/nps")
	{
		nps.GET("/status", h.NpsHandler.Status)
		nps.GET("/tunnels", h.NpsHandler.ListTunnels)
		nps.POST("/tunnels", h.NpsHandler.AddTunnel)
		nps.DELETE("/tunnels/:id", h.NpsHandler.DeleteTunnel)
	}
}
