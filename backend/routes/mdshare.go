package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterMDShareRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	mdshare := api.Group("/mdshare")
	{
		mdshare.POST("", createRateLimiter.Middleware(), h.MdShareHandler.Create)
		mdshare.GET("/:id", h.MdShareHandler.Get)
		mdshare.GET("/:id/creator", h.MdShareHandler.GetByCreator)
		mdshare.PUT("/:id", h.MdShareHandler.Update)
		mdshare.DELETE("/:id", h.MdShareHandler.Delete)
		mdshare.GET("/admin/list", h.MdShareHandler.List)
		mdshare.GET("/admin/:id", h.MdShareHandler.AdminGet)
		mdshare.DELETE("/admin/:id", h.MdShareHandler.AdminDelete)
	}
}
