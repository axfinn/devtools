package routes

import (
	"github.com/gin-gonic/gin"
)

func RegisterNFSShareRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	nfsshare := api.Group("/nfsshare")
	{
		nfsshare.GET("/status", h.NfsShareHandler.Status)
		nfsshare.GET("/turn-credentials", h.NfsShareHandler.GetTurnCredentials)
		nfsshare.GET("/:id/info", h.NfsShareHandler.Info)
		nfsshare.POST("/:id/check-password", h.NfsShareHandler.CheckPassword)
		nfsshare.GET("/:id/stream", h.NfsShareHandler.Stream)
		nfsshare.GET("/:id/qualities", h.NfsShareHandler.HLSQualities)
		nfsshare.GET("/:id/hls/:quality/:segment", func(c *gin.Context) {
			if c.Param("segment") == "index.m3u8" {
				h.NfsShareHandler.HLSPlaylist(c)
			} else {
				h.NfsShareHandler.HLSSegment(c)
			}
		})
		nfsshare.GET("/:id", h.NfsShareHandler.Access)
		nfsshare.GET("/:id/watch/ws", h.NfsShareHandler.WatchWS)
		nfsshare.POST("/:id/record", h.NfsShareHandler.UploadRecord)
		nfsshare.GET("/:id/record/:filename", h.NfsShareHandler.ServeRecord)
		nfsshare.POST("", h.NfsShareHandler.Create)
		nfsshare.GET("/admin/browse", h.NfsShareHandler.Browse)
		nfsshare.GET("/admin/raw", h.NfsShareHandler.AdminRaw)
		nfsshare.POST("/admin/login", h.NfsShareHandler.AdminLogin)
		nfsshare.POST("/admin/logout", h.NfsShareHandler.AdminLogout)
		nfsshare.GET("/admin/list", h.NfsShareHandler.AdminList)
		nfsshare.GET("/admin/mounts", h.NfsShareHandler.MountsList)
		nfsshare.POST("/admin/mounts/:name/remount", h.NfsShareHandler.MountsRemount)
		nfsshare.POST("/admin/mounts/:name/umount", h.NfsShareHandler.MountsUmount)
		nfsshare.GET("/admin/:id/logs", h.NfsShareHandler.AdminGetLogs)
		nfsshare.GET("/admin/:id/summary", h.NfsShareHandler.AdminGetSummary)
		nfsshare.GET("/admin/recordings", h.NfsShareHandler.AdminListRecordings)
		nfsshare.PUT("/admin/:id", h.NfsShareHandler.AdminUpdate)
		nfsshare.DELETE("/admin/:id", h.NfsShareHandler.AdminDelete)
		nfsshare.POST("/admin/upload/init", h.NfsShareHandler.UploadInit)
		nfsshare.POST("/admin/upload/:token/chunk", h.NfsShareHandler.UploadChunk)
		nfsshare.POST("/admin/upload/:token/complete", h.NfsShareHandler.UploadComplete)
	}
}
