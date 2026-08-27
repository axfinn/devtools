package routes

import "github.com/gin-gonic/gin"

func RegisterAutoDevRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	autodev := api.Group("/autodev")
	{
		autodev.POST("/verify", h.AutoDevHandler.VerifyPassword)
		autodev.GET("/capabilities", h.AutoDevHandler.GetCapabilities)
		autodev.POST("/tasks", h.AutoDevHandler.Submit)
		autodev.GET("/tasks", h.AutoDevHandler.List)
		autodev.GET("/projects", h.AutoDevHandler.ListProjects)
		autodev.GET("/tasks/:id", h.AutoDevHandler.GetTask)
		autodev.GET("/tasks/:id/state", h.AutoDevHandler.GetState)
		autodev.GET("/tasks/:id/files", h.AutoDevHandler.GetFiles)
		autodev.GET("/tasks/:id/file", h.AutoDevHandler.GetFile)
		autodev.GET("/tasks/:id/raw", h.AutoDevHandler.GetRawFile)
		autodev.GET("/tasks/:id/logs", h.AutoDevHandler.GetLogs)
		autodev.GET("/tasks/:id/download", h.AutoDevHandler.Download)
		autodev.GET("/tasks/:id/site/*filepath", h.AutoDevHandler.GetSite)
		autodev.POST("/tasks/:id/stop", h.AutoDevHandler.StopTask)
		autodev.POST("/tasks/:id/terminate", h.AutoDevHandler.TerminateTask)
		autodev.DELETE("/tasks/:id", h.AutoDevHandler.DeleteTask)
		autodev.POST("/ask", h.AutoDevHandler.Ask)
		autodev.GET("/ask/:id", h.AutoDevHandler.GetAskResult)
		autodev.POST("/extend", h.AutoDevHandler.Extend)
		autodev.GET("/init/stream", h.AutoDevHandler.InitProject)
		autodev.GET("/sshkey", h.AutoDevHandler.GetSSHKey)
		autodev.POST("/sshkey/regenerate", h.AutoDevHandler.RegenerateSSHKey)
		autodev.GET("/claude/version", h.AutoDevHandler.GetClaudeVersion)
		autodev.GET("/claude/cli/test", h.AutoDevHandler.TestClaudeCLI)
		autodev.GET("/claude/test", h.AutoDevHandler.TestModel)
		autodev.GET("/claude/update/stream", h.AutoDevHandler.UpdateClaude)
		autodev.GET("/codex/version", h.AutoDevHandler.GetCodexVersion)
		autodev.GET("/codex/cli/test", h.AutoDevHandler.TestCodexCLI)
		autodev.GET("/codex/update/stream", h.AutoDevHandler.UpdateCodex)
		autodev.GET("/clawtest/version", h.AutoDevHandler.GetClawtestVersion)
		autodev.GET("/clawtest/update/stream", h.AutoDevHandler.UpdateClawtest)
	}
}
