package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterMusicRoutes 音乐库接口(均在 /api/music 下)。
// 播放/列表公开只读;上传走独立限流。
func RegisterMusicRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	m := api.Group("/music")
	{
		m.GET("/tracks", h.MusicHandler.ListTracks)
		m.GET("/tracks/:id", h.MusicHandler.GetTrack)
		m.GET("/stats", h.MusicHandler.GetStats)
		m.POST("/tracks", createRateLimiter.Middleware(), h.MusicHandler.UploadTrack)
		m.PATCH("/tracks/:id", h.MusicHandler.UpdateTrack)
		m.POST("/tracks/:id/favorite", h.MusicHandler.ToggleFavorite)
		m.POST("/tracks/:id/played", h.MusicHandler.MarkPlayed)
		m.POST("/tracks/:id/cover", createRateLimiter.Middleware(), h.MusicHandler.UploadCover)
		m.DELETE("/tracks/:id", h.MusicHandler.DeleteTrack)
		m.GET("/stream/:id", h.MusicHandler.StreamTrack)
		m.GET("/cover/:id", h.MusicHandler.ServeCover)
	}
}
