package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPasteRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	paste := api.Group("/paste")
	{
		paste.POST("", createRateLimiter.Middleware(), h.PasteHandler.Create)
		paste.POST("/upload", h.PasteHandler.UploadFile)
		paste.GET("/files/:filename", h.PasteHandler.ServeFile)
		paste.GET("/:id", h.PasteHandler.Get)
		paste.GET("/:id/info", h.PasteHandler.GetInfo)

		paste.POST("/chunk/init", h.PasteHandler.InitChunkUpload)
		paste.POST("/chunk/:file_id", h.PasteHandler.UploadChunk)
		paste.POST("/chunk/:file_id/merge", h.PasteHandler.MergeChunks)
		paste.GET("/chunk/:file_id/status", h.PasteHandler.CheckChunkStatus)

		paste.POST("/analyze", h.PasteHandler.AnalyzeCode)
		paste.GET("/analyze/:file_id", h.PasteHandler.AnalyzeFile)

		paste.POST("/scan", h.PasteHandler.ScanContent)
		paste.GET("/validate/:file_id", h.PasteHandler.ValidateFile)

		paste.GET("/languages", h.PasteHandler.GetSupportedLanguages)
		paste.GET("/content-types", h.PasteHandler.GetSupportedContentTypes)
		paste.GET("/stats", h.PasteHandler.GetStats)
		paste.GET("/search", h.PasteHandler.SearchPastes)

		paste.GET("/admin/list", h.PasteHandler.AdminListPastes)
		paste.GET("/admin/:id", h.PasteHandler.AdminGetPaste)
		paste.PUT("/admin/:id", h.PasteHandler.AdminUpdatePaste)
		paste.DELETE("/admin/:id", h.PasteHandler.AdminDeletePaste)
	}
}
