package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterSkillsRoutes(api *gin.RouterGroup, h *RouteHandlers, skillsGuard *middleware.SkillsGuard) {
	skills := api.Group("/skills", skillsGuard.Middleware())
	{
		skills.GET("/manifest", h.SkillsHandler.GetManifest)
		skills.GET("/install", h.SkillsHandler.GetInstall)
		skills.GET("/install.sh", h.SkillsHandler.GetInstallShell)
		skills.GET("/mcp", h.SkillsHandler.MCPGetHandler)
		skills.POST("/mcp", h.SkillsHandler.MCPPostHandler)
		skills.POST("/invoke", h.SkillsHandler.Invoke)
	}
	api.GET("/.well-known/skills", h.SkillsHandler.GetDirectory)
}
