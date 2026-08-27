package routes

import "github.com/gin-gonic/gin"

func RegisterEdgeTTSRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	edgeTTS := api.Group("/edge-tts")
	{
		edgeTTS.GET("/voices", h.EdgeTTSHandler.ListVoices)
		edgeTTS.POST("/tts", h.EdgeTTSHandler.Synthesize)
		edgeTTS.GET("/audio/:filename", h.EdgeTTSHandler.ServeAudioFile)
		edgeTTS.GET("/health", h.EdgeTTSHandler.Health)
		edgeTTS.POST("/convert", h.EdgeTTSHandler.ConvertAudioFormat)
	}
}
