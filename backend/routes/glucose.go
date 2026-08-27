package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterGlucoseRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	glucose := api.Group("/glucose")
	{
		glucose.POST("", createRateLimiter.Middleware(), h.GlucoseHandler.Create)
		glucose.POST("/login", h.GlucoseHandler.Login)
		glucose.GET("/:id", h.GlucoseHandler.Get)
		glucose.DELETE("/:id", h.GlucoseHandler.Delete)
		glucose.PUT("/:id/extend", h.GlucoseHandler.Extend)
		glucose.GET("/:id/records", h.GlucoseHandler.GetRecords)
		glucose.POST("/:id/records", h.GlucoseHandler.CreateRecord)
		glucose.PUT("/:id/records/:recordId", h.GlucoseHandler.UpdateRecord)
		glucose.DELETE("/:id/records/:recordId", h.GlucoseHandler.DeleteRecord)
		glucose.GET("/:id/records/:recordId/history", h.GlucoseHandler.GetRecordHistory)
		glucose.GET("/:id/history", h.GlucoseHandler.GetProfileHistory)
		glucose.POST("/:id/import", h.GlucoseHandler.ImportRecords)
		glucose.GET("/:id/stats", h.GlucoseHandler.GetStats)
		glucose.POST("/:id/voice-parse", h.GlucoseHandler.VoiceParse)
	}
}
