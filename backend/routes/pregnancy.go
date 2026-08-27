package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPregnancyRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	pregnancy := api.Group("/pregnancy")
	{
		pregnancy.POST("", createRateLimiter.Middleware(), h.PregnancyHandler.Create)
		pregnancy.POST("/login", h.PregnancyHandler.Login)
		pregnancy.GET("/:id", h.PregnancyHandler.Get)
		pregnancy.GET("/:id/creator", h.PregnancyHandler.GetByCreator)
		pregnancy.GET("/:id/device", h.PregnancyHandler.GetByDevice)
		pregnancy.PUT("/:id", h.PregnancyHandler.Update)
		pregnancy.DELETE("/:id", h.PregnancyHandler.Delete)
	}
}
