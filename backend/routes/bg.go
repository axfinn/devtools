package routes

import (
	"devtools/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterBackgroundRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	api.GET("/bg", handlers.GetBackgroundImages)
	api.POST("/bg/cache", handlers.CacheBackgroundImages)
	api.POST("/bg/replace", handlers.ReplaceRandomImages)
	api.GET("/bg/random", handlers.GetRandomBackground)
	api.GET("/bg/cached/:filename", handlers.ServeCachedBackground)
}
