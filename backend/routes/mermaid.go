package routes

import "github.com/gin-gonic/gin"

func RegisterMermaidRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	mermaid := api.Group("/mermaid")
	{
		mermaid.POST("/projects", h.MermaidHandler.CreateProject)
		mermaid.GET("/projects", h.MermaidHandler.ListProjects)
		mermaid.DELETE("/projects/:id", h.MermaidHandler.DeleteProject)
		mermaid.POST("/projects/:id/versions", h.MermaidHandler.SaveVersion)
		mermaid.GET("/projects/:id/versions", h.MermaidHandler.ListVersions)
		mermaid.POST("/projects/:id/generate", h.MermaidHandler.AIGenerate)
		mermaid.GET("/projects/:id/messages", h.MermaidHandler.ListMessages)
		mermaid.DELETE("/projects/:id/messages", h.MermaidHandler.ClearMessages)
	}
}
