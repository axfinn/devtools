package routes

import (
	"devtools/handlers"
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// ssRuleMaxBodySize 管理端请求体上限:1MB。
//
// 入参是域名列表,生成器自身的上限是 512KB,1MB 足够宽松。
// 收紧是为了别让全站默认的 200MB 上限把内存吃掉。
const ssRuleMaxBodySize = 1 << 20

// RegisterSSRuleRoutes 挂载 /api/ssrules/* 管理端点。
//
// 第三个参数是全站共享的限流器,本模块**刻意不用**(故以 `_` 接收):
//   - 它的键是 c.ClientIP(),gin 默认 trustedProxies=0.0.0.0/0 ⇒ 取 XFF 最左项
//     ⇒ 客户端可伪造限流桶 key,轮换该头即无限绕过;
//   - 它是全站唯一实例,计数桶与其它模块共用,别人被打爆会把本模块一起拖下水。
//
// 本模块自建独立限流器(见 handlers.NewSSRuleLimiter),**绝不回落到它**。
//
// 公开的规则文件分发(/ss/:token/:filename)不在这里 —— 它不走 /api 分组,
// 而是在 main 包的 registerSSRulePublicRoutes 里注册,见 app_http.go。
func RegisterSSRuleRoutes(api *gin.RouterGroup, h *RouteHandlers, _ *middleware.RateLimiter) {
	if h.SSRuleHandler == nil {
		return
	}

	// 请求体上限 1MB:本模块的入参是域名列表,实际用不到 512KB(生成器自身的上限),
	// 但 1MB 足够宽松,不必卡太死。收紧是为了别让 200MB 的默认上限把内存吃掉。
	g := api.Group("/ssrules", middleware.ContentSizeLimiter(ssRuleMaxBodySize))
	{
		// 密码门也要限流:这是本模块唯一的密码校验点,
		// 不限流等于给了无限次爆破机会。
		g.POST("/verify",
			handlers.NewSSRuleLimiter(handlers.SSRuleVerifyRateLimitPerMinute),
			h.SSRuleHandler.Verify)
		// 读端点同样必须限流:绕过 verify 直接打 GET 也是爆破路径。
		g.GET("",
			handlers.NewSSRuleLimiter(handlers.SSRuleReadRateLimitPerMinute),
			h.SSRuleHandler.Get)
		g.POST("",
			handlers.NewSSRuleLimiter(handlers.SSRuleWriteRateLimitPerMinute),
			h.SSRuleHandler.Save)
		g.POST("/regenerate-token",
			handlers.NewSSRuleLimiter(handlers.SSRuleRegenerateRateLimitPerMinute),
			h.SSRuleHandler.RegenerateToken)
	}
}
