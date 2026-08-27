package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterExcalidrawRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	excalidraw := api.Group("/excalidraw")
	{
		excalidraw.POST("", createRateLimiter.Middleware(), h.ExcalidrawHandler.Create)
		excalidraw.GET("/:id", h.ExcalidrawHandler.Get)
		excalidraw.GET("/:id/creator", h.ExcalidrawHandler.GetByCreator)
		excalidraw.PUT("/:id", h.ExcalidrawHandler.Update)
		excalidraw.DELETE("/:id", h.ExcalidrawHandler.Delete)
		excalidraw.GET("/admin/list", h.ExcalidrawHandler.List)
		excalidraw.GET("/admin/:id", h.ExcalidrawHandler.AdminGet)
		excalidraw.DELETE("/admin/:id", h.ExcalidrawHandler.AdminDelete)
	}
}
