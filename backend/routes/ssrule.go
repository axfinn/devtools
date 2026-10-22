package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterSSRuleRoutes 挂载 /api/ssrules/* 管理端点。
//
// 公开的规则文件分发(/ss/:token/:filename)不在这里 —— 它不走 /api 分组,
// 而是在 main 包的 registerPublicRoutes 里注册,见 app_http.go。
func RegisterSSRuleRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	if h.SSRuleHandler == nil {
		return
	}
	g := api.Group("/ssrules")
	{
		// 密码门也要限流:这是本模块唯一的密码校验点,
		// 不限流等于给了无限次爆破机会。
		g.POST("/verify", createRateLimiter.Middleware(), h.SSRuleHandler.Verify)
		g.GET("", h.SSRuleHandler.Get)
		g.POST("", createRateLimiter.Middleware(), h.SSRuleHandler.Save)
		g.POST("/regenerate-token", createRateLimiter.Middleware(), h.SSRuleHandler.RegenerateToken)
	}
}
