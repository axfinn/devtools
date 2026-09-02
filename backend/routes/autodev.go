package routes

import (
	"time"

	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAutoDevRoutes 注册 autodev 路由。Admin 写端点(Submit/Ask/Extend/Stop/Terminate/Delete/
// SSHKeyRegenerate)挂 RateLimiter(默认 5/min/IP),防止 admin pwd 撞库后批量触发 Claude/Codex 子进程;
// 其余读/诊断/流式端点不限流,避免误伤调试体验。
func RegisterAutoDevRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	autodev := api.Group("/autodev")
	autodev.POST("/verify", h.AutoDevHandler.VerifyPassword)
	autodev.GET("/capabilities", h.AutoDevHandler.GetCapabilities)

	// 写端点限流:5/min/IP。store 由 createRateLimiter 提供(共享全局 transientStore)。
	writeRL := middleware.NewRateLimiter(5, time.Minute, nil)
	if createRateLimiter != nil {
		writeRL = createRateLimiter
	}
	writeOps := autodev.Group("", writeRL.Middleware())
	{
		writeOps.POST("/tasks", h.AutoDevHandler.Submit)
		writeOps.GET("/tasks", h.AutoDevHandler.List)
		writeOps.GET("/projects", h.AutoDevHandler.ListProjects)
		writeOps.GET("/tasks/:id", h.AutoDevHandler.GetTask)
		writeOps.GET("/tasks/:id/state", h.AutoDevHandler.GetState)
		writeOps.GET("/tasks/:id/files", h.AutoDevHandler.GetFiles)
		writeOps.GET("/tasks/:id/file", h.AutoDevHandler.GetFile)
		writeOps.GET("/tasks/:id/raw", h.AutoDevHandler.GetRawFile)
		writeOps.GET("/tasks/:id/logs", h.AutoDevHandler.GetLogs)
		writeOps.GET("/tasks/:id/download", h.AutoDevHandler.Download)
		writeOps.GET("/tasks/:id/site/*filepath", h.AutoDevHandler.GetSite)
		writeOps.POST("/tasks/:id/stop", h.AutoDevHandler.StopTask)
		writeOps.POST("/tasks/:id/terminate", h.AutoDevHandler.TerminateTask)
		writeOps.DELETE("/tasks/:id", h.AutoDevHandler.DeleteTask)
		writeOps.POST("/ask", h.AutoDevHandler.Ask)
		writeOps.GET("/ask/:id", h.AutoDevHandler.GetAskResult)
		writeOps.POST("/extend", h.AutoDevHandler.Extend)
		writeOps.GET("/init/stream", h.AutoDevHandler.InitProject)
		writeOps.GET("/sshkey", h.AutoDevHandler.GetSSHKey)
		writeOps.POST("/sshkey/regenerate", h.AutoDevHandler.RegenerateSSHKey)
		writeOps.GET("/claude/version", h.AutoDevHandler.GetClaudeVersion)
		writeOps.GET("/claude/cli/test", h.AutoDevHandler.TestClaudeCLI)
		writeOps.GET("/claude/test", h.AutoDevHandler.TestModel)
		writeOps.GET("/claude/update/stream", h.AutoDevHandler.UpdateClaude)
		writeOps.GET("/codex/version", h.AutoDevHandler.GetCodexVersion)
		writeOps.GET("/codex/cli/test", h.AutoDevHandler.TestCodexCLI)
		writeOps.GET("/codex/update/stream", h.AutoDevHandler.UpdateCodex)
		writeOps.GET("/clawtest/version", h.AutoDevHandler.GetClawtestVersion)
		writeOps.GET("/clawtest/update/stream", h.AutoDevHandler.UpdateClawtest)
	}
}
