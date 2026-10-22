package handlers

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"devtools/middleware"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// SSRuleHandler 管理「Shadowsocks 规则文件」的编辑与公开分发。
//
// 本模块刻意保持极简:全站只有一份规则文件,页面填域名列表、生成规则正文,
// 公开链接给客户端直接拉。不做多套规则、不做规则类型建模。
type SSRuleHandler struct {
	db            *models.DB
	adminPassword string
}

func NewSSRuleHandler(db *models.DB, adminPassword string) *SSRuleHandler {
	return &SSRuleHandler{db: db, adminPassword: adminPassword}
}

// ssRuleFileNames 公开链接允许的文件名。
//
// 同一份内容用多个扩展名暴露:不同客户端对「规则文件 URL」的后缀要求不一致,
// 内容本身是纯文本,换个后缀就能被另一批客户端接受。
//
// 这里**刻意不含 .pac**:PAC 是 JavaScript,把 ACL 正文当 .pac 发出去,
// 客户端会静默拿到一个语法错误的脚本,报错信息完全指不到这里。
var ssRuleFileNames = []string{"rules.acl", "rules.txt", "rules.list", "rules.conf"}

// ssRuleFileContentType 全部是纯文本。ACL / txt / list / conf 对客户端而言
// 都是「一行一条的文本」,没有各自的 MIME 类型。
const ssRuleFileContentType = "text/plain; charset=utf-8"

// SSRuleNotFoundBody 是 /ss/ 前缀下所有「不该存在」形态的统一响应体。
//
// 必须是**同一个字面量**:分发端的白名单拒绝、token 不存在、模块关闭,
// 以及路由没匹配上落到 NoRoute 的那些形态,外部观察者看到的话必须完全一样
// —— 能区分「token 错」和「文件名错」本身就是信息泄露。
//
// 导出是因为 backend/app_http.go 的 NoRoute 分支也要用同一句 —— 两处各拼一个
// 字符串字面量,早晚拼错。
const SSRuleNotFoundBody = "规则文件不存在"

// ssRuleRateLimitWindow 限流窗口。全部端点统一按「每分钟」计。
const ssRuleRateLimitWindow = time.Minute

// 各端点的限流阈值(次 / 分钟 / IP)。
//
// 刻意**按端点分桶**而不是全模块共用一个桶:共用会让正常保存操作被自己的
// 读操作挤掉(打开页面读一次 + 改一次就吃掉大半配额)。
const (
	// SSRuleVerifyRateLimitPerMinute 管理端密码门。
	SSRuleVerifyRateLimitPerMinute = 10
	// SSRuleRegenerateRateLimitPerMinute 换发 token。
	SSRuleRegenerateRateLimitPerMinute = 10
	// SSRuleReadRateLimitPerMinute 读取当前规则。
	SSRuleReadRateLimitPerMinute = 30
	// SSRuleWriteRateLimitPerMinute 保存规则。
	SSRuleWriteRateLimitPerMinute = 20
	// SSRuleDistributeRateLimitPerMinute 公开分发端。
	SSRuleDistributeRateLimitPerMinute = 120
)

// ssRuleIsAllowedFile 校验文件名是否在白名单内。
func ssRuleIsAllowedFile(name string) bool {
	for _, n := range ssRuleFileNames {
		if n == name {
			return true
		}
	}
	return false
}

// checkAdmin 校验管理密码。
//
// **只认** header X-Super-Admin-Password(已在 CORS 白名单里),不接受 query 兜底:
// 密码一旦进了 URL,就会留在 Cloudflare 访问日志、浏览器历史、Referer 头里,
// 而这些都不是能随时擦掉的存储。
//
// 用定长比较而非 ==,避免时序侧信道。
func (h *SSRuleHandler) checkAdmin(c *gin.Context) bool {
	if h.adminPassword == "" {
		return false // 未配置密码 = 功能关闭,而不是「无密码放行」
	}
	given := c.GetHeader("X-Super-Admin-Password")
	return subtle.ConstantTimeCompare([]byte(given), []byte(h.adminPassword)) == 1
}

// SSRuleClientIP 取本模块的限流键。
//
// 不用 gin 的 c.ClientIP():gin 默认 trustedProxies = ["0.0.0.0/0","::/0"]
// (gin.go:205) + ForwardedByClientIP=true (gin.go:195) +
// RemoteIPHeaders ["X-Forwarded-For","X-Real-IP"] (gin.go:196),
// validateHeader (gin.go:458) 于是直接返回 XFF **最左边**那一项 ——
// 客户端自带 X-Forwarded-For 就能任意伪造限流桶 key,轮换该头即无限绕过。
// 全仓没有 SetTrustedProxies 调用,上面三个默认值全部生效。
//
// 现在的取法:CF 覆写过的头最可信,其次取代理追加的 XFF **最右**一项,
// 最后退到直连来源地址。
func SSRuleClientIP(c *gin.Context) string {
	if v := strings.TrimSpace(c.GetHeader("CF-Connecting-IP")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); v != "" {
		parts := strings.Split(v, ",")
		return strings.TrimSpace(parts[len(parts)-1]) // 最右一项 = 代理追加的那一项
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

// NewSSRuleLimiter 返回本模块专用的限流中间件。
//
// 为什么不用共享的 middleware.RateLimiter.Middleware():
//   - 它的键是 c.ClientIP(),可被客户端伪造(见 SSRuleClientIP 的注释);
//   - 它是全站唯一实例(app_http.go),计数桶与其它模块共用 —— 别人被打爆会把
//     本模块一起拖下水,本模块被打爆也会连累别人。
//
// 共享类型只借用它的「桶」逻辑:Allow(ip) 已导出(ratelimit.go:32),
// 每个 NewRateLimiter(..., nil) 自建一个私有 MemoryStore(ratelimit.go:20-23)
// ⇒ 天然独立计数桶。nil 是刻意的:本模块不落 Redis(Redis 挂了行为不变)。
func NewSSRuleLimiter(limit int) gin.HandlerFunc {
	rl := middleware.NewRateLimiter(limit, ssRuleRateLimitWindow, nil) // nil ⇒ 独立 store
	return func(c *gin.Context) {
		if !rl.Allow(SSRuleClientIP(c)) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

type ssRuleSaveRequest struct {
	Source string `json:"source"`
}

// ssRuleView 是管理接口统一响应体。
//
// Paths 返回相对路径,由前端拼上 window.location.origin —— 服务端不知道
// 用户是从哪个域名访问的(可能走 CF、可能走内网),拼死反而会给出错链接。
type ssRuleView struct {
	Source    string    `json:"source"`
	Content   string    `json:"content"`
	Token     string    `json:"token"`
	UpdatedAt time.Time `json:"updated_at"`
	Paths     []string  `json:"paths"`
	Skipped   []string  `json:"skipped"`
}

func newSSRuleView(file *models.SSRuleFile, skipped []string) ssRuleView {
	paths := make([]string, 0, len(ssRuleFileNames))
	for _, name := range ssRuleFileNames {
		paths = append(paths, "/ss/"+file.Token+"/"+name)
	}
	if skipped == nil {
		skipped = []string{}
	}
	return ssRuleView{
		Source:    file.Source,
		Content:   file.Content,
		Token:     file.Token,
		UpdatedAt: file.UpdatedAt,
		Paths:     paths,
		Skipped:   skipped,
	}
}

// Verify POST /api/ssrules/verify —— 管理页密码门。
func (h *SSRuleHandler) Verify(c *gin.Context) {
	if !h.checkAdmin(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Get GET /api/ssrules —— 读取当前规则文件(含原始输入,用于回显编辑框)。
func (h *SSRuleHandler) Get(c *gin.Context) {
	if !h.checkAdmin(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	file, err := h.db.GetSSRuleFile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取规则文件失败"})
		return
	}

	// 历史内容可能是在校验放宽前存进去的,回显时重跑一遍生成,
	// 让页面上的 skipped 提示与现实一致。
	content, skipped, err := buildACLRules(file.Source)
	if err != nil {
		// 回落行为不变(用库里存的正文 + 空 skipped),但必须留现场:
		// 否则「页面上 skipped 为什么是空的」事后无从查起。
		log.Printf("ssrules: 规则正文重建失败,回落到库内已存内容: %v", err)
		content, skipped = file.Content, []string{}
	}
	file.Content = content

	c.JSON(http.StatusOK, newSSRuleView(file, skipped))
}

// Save POST /api/ssrules —— 保存域名列表,重新生成规则正文。
func (h *SSRuleHandler) Save(c *gin.Context) {
	if !h.checkAdmin(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	var req ssRuleSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	content, skipped, err := buildACLRules(req.Source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := h.db.SaveSSRuleFile(req.Source, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, newSSRuleView(file, skipped))
}

// RegenerateToken POST /api/ssrules/regenerate-token —— 换发 token。
// 公开链接外泄时的止损手段:换完旧链接立刻失效。
func (h *SSRuleHandler) RegenerateToken(c *gin.Context) {
	if !h.checkAdmin(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	file, err := h.db.RegenerateSSRuleToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "换发失败"})
		return
	}

	// 换发 token 不改 source,但页面上的 skipped 提示必须与现实一致:
	// 重跑一遍生成,让当前 source 里的非法行照样报出来(与 Get 的口径相同)。
	content, skipped, err := buildACLRules(file.Source)
	if err != nil {
		// 与 Get 同一口径:回落行为不变,只补可观测性。
		log.Printf("ssrules: 规则正文重建失败,回落到库内已存内容: %v", err)
		content, skipped = file.Content, []string{}
	}
	file.Content = content

	c.JSON(http.StatusOK, newSSRuleView(file, skipped))
}

// Download GET /ss/:token/:filename —— 公开分发规则文件,不需密码。
//
// 安全性依赖 token 的不可枚举性(32 位随机 hex)。客户端拉规则时基本不支持
// 交互式认证,所以这里不能要求密码,只能靠 token。
//
// 这里**不碰文件系统**:文件名过白名单就放行,正文来自数据库。
// 就算白名单被绕过,也没有任何文件读取面(这是路径穿越防护的根,不是补丁)。
func (h *SSRuleHandler) Download(c *gin.Context) {
	// 「未配密码 = 模块关闭」必须在分发端同样成立:
	// 用户存好规则后若把 admin_password 清空以为关了模块,旧链接还在吐内容。
	// 配置层面的「关闭」必须等于数据层面的「关闭」。
	// 响应与「token 不存在」逐字节相同,外部观察者分辨不出模块被关了。
	if h.adminPassword == "" {
		c.String(http.StatusNotFound, SSRuleNotFoundBody)
		return
	}

	filename := c.Param("filename")
	if !ssRuleIsAllowedFile(filename) {
		c.String(http.StatusNotFound, SSRuleNotFoundBody)
		return
	}

	file, err := h.db.GetSSRuleFileByToken(c.Param("token"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.String(http.StatusNotFound, SSRuleNotFoundBody)
			return
		}
		c.String(http.StatusInternalServerError, "读取规则文件失败")
		return
	}

	// no-store:用户改完规则后客户端下次拉取必须拿到新内容,
	// 缓存住会让「改了规则但客户端没生效」变成极难排查的问题。
	c.Header("Content-Type", ssRuleFileContentType)
	c.Header("Cache-Control", "no-store")
	// nosniff:防浏览器 MIME 猜测。正文是纯文本,被当成 HTML 渲染才是事故。
	c.Header("X-Content-Type-Options", "nosniff")
	// Content-Disposition 只拼白名单常量:filename 是外部可控的,拼进响应头
	// 就是注入面。这里恒为 rules.acl,与请求的后缀无关。
	c.Header("Content-Disposition", `attachment; filename="rules.acl"`)
	c.String(http.StatusOK, file.Content)
}

// NotFound 是 /ss/ 前缀下所有「不该存在」形态的统一响应。
//
// 与 Download 内部的白名单拒绝 / token 不存在 / 模块关闭响应逐字节一致。
// 用来显式关掉 /ss/ 前缀的 RedirectTrailingSlash 自动重定向:不注册这些路由的话,
// gin 会 301 到无尾斜杠形态,尾斜杠载荷就拿不到同款 404。
func (h *SSRuleHandler) NotFound(c *gin.Context) {
	c.String(http.StatusNotFound, SSRuleNotFoundBody)
}
