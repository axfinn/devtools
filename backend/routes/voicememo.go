package routes

import "github.com/gin-gonic/gin"

func RegisterVoiceMemoRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	voicememo := api.Group("/voicememo")
	{
		voicememo.POST("/upload", h.VoiceMemoHandler.Upload)
		voicememo.GET("/list", h.VoiceMemoHandler.List)
		voicememo.GET("/task/:taskId/list", h.VoiceMemoHandler.ListByTask)
		voicememo.GET("/:id/audio", h.VoiceMemoHandler.ServeMemoAudio)
		voicememo.POST("/:id/transcribe", h.VoiceMemoHandler.Transcribe)
		voicememo.POST("/:id/summarize", h.VoiceMemoHandler.Summarize)
		voicememo.POST("/:id/planner-task", h.VoiceMemoHandler.CreatePlannerTask)
		voicememo.PUT("/:id", h.VoiceMemoHandler.Update)
		voicememo.GET("/:id", h.VoiceMemoHandler.Get)
		voicememo.DELETE("/:id", h.VoiceMemoHandler.Delete)
	}
}
