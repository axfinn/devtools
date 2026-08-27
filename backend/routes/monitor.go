package routes

import "github.com/gin-gonic/gin"

func RegisterMonitorRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	monitor := api.Group("/monitor")
	{
		monitor.GET("/verify", h.MonitoringHandler.Verify)
		monitor.GET("/overview", h.MonitoringHandler.Overview)
		monitor.GET("/responses", h.MonitoringHandler.Responses)
		monitor.GET("/logs", h.MonitoringHandler.Logs)
		monitor.GET("/ai", h.MonitoringHandler.AI)
		monitor.GET("/service", h.MonitoringHandler.Service)
		monitor.GET("/sessions", h.MonitoringHandler.Sessions)
		monitor.POST("/archive", h.MonitoringHandler.Archive)
		monitor.DELETE("/logs", h.MonitoringHandler.DeleteLogs)
	}
}
