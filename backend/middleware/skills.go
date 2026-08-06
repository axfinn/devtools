package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"devtools/config"
	"devtools/state"

	"github.com/gin-gonic/gin"
)

// SkillsGuard Skills 路由 guard —— 默认未启用(配置打开才放行);打开后:
//   1. 校验 Origin 白名单(若配置了 AllowedOrigins)
//   2. POST 走总配额限流(按 IP 滑动窗口)
//   3. 主动读 X-Forwarded-For 第 1 个 IP,避免依赖 c.ClientIP()(后者在反代场景下
//      需要先 SetTrustedProxies 才正确,当前项目未配置 — 显式读最稳)
type SkillsGuard struct {
	enabled            bool
	rateLimitPerMinute int
	totalLimiter       *RateLimiter
	writeLimiter       *RateLimiter
	allowedOrigins     []string
}

// NewSkillsGuard 构造 guard;store 通常传 TransientStore(Redis 或内存降级)
func NewSkillsGuard(cfg config.SkillsConfig, store state.TransientStore) *SkillsGuard {
	total := cfg.RateLimitPerMinute
	if total <= 0 {
		total = 60
	}
	write := cfg.WriteRateLimitPerMinute
	if write <= 0 {
		write = 5
	}
	return &SkillsGuard{
		enabled:            cfg.Enabled,
		rateLimitPerMinute: total,
		totalLimiter:       NewRateLimiter(total, time.Minute, store),
		writeLimiter:       NewRateLimiter(write, time.Minute, store),
		allowedOrigins:     cfg.AllowedOrigins,
	}
}

// GetClientIP 主动按 X-Forwarded-For > X-Real-IP > c.ClientIP() 顺序取值。
// 因为当前 engine 未 SetTrustedProxies,c.ClientIP() 在反代环境拿到的是反代 IP,
// 所有请求会聚到同一 key 上,使限流完全失效 —— Skills 关键路径必须自己解析。
func GetClientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		for _, p := range strings.Split(xff, ",") {
			if ip := strings.TrimSpace(p); ip != "" {
				return ip
			}
		}
	}
	if xri := strings.TrimSpace(c.GetHeader("X-Real-IP")); xri != "" {
		return xri
	}
	return c.ClientIP()
}

func (g *SkillsGuard) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !g.enabled {
			c.JSON(http.StatusNotFound, gin.H{"error": "Skills 接口未启用,请在 config.yaml 设置 skills.enabled=true", "code": 404})
			c.Abort()
			return
		}

		// Origin 白名单(非空时启用)
		if len(g.allowedOrigins) > 0 {
			origin := strings.TrimSpace(c.GetHeader("Origin"))
			if origin == "" || !g.originAllowed(origin) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Origin 不在白名单", "code": 403})
				c.Abort()
				return
			}
		}

		// GET 不限流(manifest / 初始化握手都是无成本元数据)
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		// POST 走总配额
		if !g.totalLimiter.Allow("skills:total:" + GetClientIP(c)) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Skills 接口调用过于频繁,请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// CheckWriteLimit 写库类 skill 走专项更严限流(默认 5/min/IP per skill),
// 在 skills handler Invoke 内分发到具体 skill 后调用。
func (g *SkillsGuard) CheckWriteLimit(skillName, ip string) error {
	if !g.writeLimiter.Allow("skills:write:" + skillName + ":" + ip) {
		return errors.New("write-class skill " + skillName + " 已超过单 IP 频率限制")
	}
	return nil
}

// AllowedWriteTools 返回被识别为写库类的 skill name 集合,默认 ["shorturl_create","paste_create"]
func (g *SkillsGuard) AllowedWriteTools() map[string]bool {
	return map[string]bool{
		"shorturl_create": true,
		"paste_create":    true,
	}
}

// IsWriteClass 检查某个 skill name 是否属于写库类(skills handler 用于决定是否走 CheckWriteLimit)
func (g *SkillsGuard) IsWriteClass(skillName string) bool {
	tools := g.AllowedWriteTools()
	_, ok := tools[skillName]
	return ok
}

func (g *SkillsGuard) originAllowed(origin string) bool {
	for _, o := range g.allowedOrigins {
		if o == "*" || strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}

// RateLimitPerMinute 暴露给其他模块(测试、调试)读取
func (g *SkillsGuard) RateLimitPerMinute() int { return g.rateLimitPerMinute }
