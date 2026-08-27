package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRecipeRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	recipe := api.Group("/recipe")
	{
		recipe.GET("/default", h.RecipeHandler.GetDefault)
		recipe.GET("/detailed", h.RecipeHandler.GetDetailed)
		recipe.POST("", createRateLimiter.Middleware(), h.RecipeHandler.Create)
		recipe.POST("/login", h.RecipeHandler.Login)
		recipe.GET("/:id", h.RecipeHandler.Get)
		recipe.GET("/:id/creator", h.RecipeHandler.GetByCreator)
		recipe.PUT("/:id", h.RecipeHandler.Update)
		recipe.DELETE("/:id", h.RecipeHandler.Delete)
	}
}
