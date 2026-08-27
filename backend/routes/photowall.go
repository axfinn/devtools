package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPhotoWallRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	photowall := api.Group("/photowall")
	{
		photowall.POST("/profile", createRateLimiter.Middleware(), h.PhotoWallHandler.CreateProfile)
		photowall.POST("/profile/login", h.PhotoWallHandler.LoginProfile)
		photowall.GET("/profile/:id", h.PhotoWallHandler.GetProfile)
		photowall.PUT("/profile/:id", h.PhotoWallHandler.UpdateProfile)
		photowall.DELETE("/profile/:id", h.PhotoWallHandler.DeleteProfile)
		photowall.POST("/profile/:id/items", createRateLimiter.Middleware(), h.PhotoWallHandler.UploadItem)
		photowall.PUT("/profile/:id/items/:itemId", h.PhotoWallHandler.UpdateItem)
		photowall.DELETE("/profile/:id/items/:itemId", h.PhotoWallHandler.DeleteItem)
		photowall.GET("/profile/:id/download", h.PhotoWallHandler.DownloadSelection)
		photowall.GET("/share/:id", h.PhotoWallHandler.GetShare)
		photowall.GET("/files/:filename", h.PhotoWallHandler.ServeFile)
		photowall.GET("/admin/list", h.PhotoWallHandler.AdminList)
		photowall.GET("/admin/:id", h.PhotoWallHandler.AdminGet)
		photowall.PUT("/admin/:id", h.PhotoWallHandler.AdminUpdate)
		photowall.DELETE("/admin/:id", h.PhotoWallHandler.AdminDelete)
	}
}
