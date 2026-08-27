package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterAskitRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	askit := api.Group("/askit/v1")
	{
		auth := askit.Group("/auth")
		{
			auth.POST("/request-code", createRateLimiter.Middleware(), h.AskitSyncHandler.RequestCode)
			auth.POST("/login-code", createRateLimiter.Middleware(), h.AskitSyncHandler.LoginCode)
			auth.POST("/refresh", h.AskitSyncHandler.Refresh)
			auth.GET("/me", h.AskitSyncHandler.AuthMiddleware(), h.AskitSyncHandler.Me)
			auth.POST("/logout", h.AskitSyncHandler.AuthMiddleware(), h.AskitSyncHandler.Logout)
		}
		sync := askit.Group("/sync", h.AskitSyncHandler.AuthMiddleware())
		{
			sync.GET("/pull", h.AskitSyncHandler.Pull)
			sync.POST("/push", h.AskitSyncHandler.Push)
			sync.GET("/snapshot", h.AskitSyncHandler.Snapshot)
		}
		askit.POST("/blob/upload", h.AskitSyncHandler.AuthMiddleware(), h.AskitSyncHandler.BlobUpload)
		askit.GET("/blob/files/:uid/:name", h.AskitSyncHandler.BlobServe)
		askit.POST("/admin/invites", h.AskitSyncHandler.CreateInvites)
		askit.GET("/admin/users", h.AskitSyncHandler.AdminUsersOverview)
	}
}
