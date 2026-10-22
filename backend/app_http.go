package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devtools/config"
	"devtools/handlers"
	"devtools/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func newHTTPRouter(rt *appRuntime, handlers *routeHandlers) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	// gin.Default() 等价于 gin.New() + gin.Logger() + gin.Recovery()。
	// 这里显式拆开写,是为了在 Logger **之前**插一个抹掉 /ss/ 分发 token 的
	// 中间件 —— gin 的 Logger 在进入时就把 URL.Path 抓进局部变量,
	// c.Next() 之后才拿它拼日志,事后在 handler 里补救已经来不及。
	//
	// 中间件顺序与 gin.Default() 完全一致(Logger → Recovery → 业务中间件),
	// 只是最前面多了一个**只作用于 /ss/ 路径**的遮罩,其它模块的日志行为不变。
	router := gin.New()
	router.Use(redactSSRuleTokenPath())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 响应压缩（gzip level 5，平衡压缩比与 CPU）。
	// 见 docs/plans/finn-29-first-paint-perf.md §C 与 FINN-41 评审结论。
	// 排除规则在 middleware.ResponseCompression 内集中维护。
	router.Use(middleware.ResponseCompression(5))

	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-Super-Admin-Password"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))
	router.Use(middleware.ContentSizeLimiter(200 * 1024 * 1024))
	requestLogger := middleware.NewRequestLogger(rt.db, rt.cfg.Monitoring.Enabled)
	router.Use(requestLogger.Middleware())

	createRateLimiter := middleware.NewRateLimiter(10, time.Minute, rt.transientStore)
	api := router.Group("/api")
	setupRoutes(api, createRateLimiter, handlers)

	registerPublicRoutes(router, handlers)
	registerStaticRoutes(router, "./dist")

	return router
}

// ssRuleTokenMask 是 /ss/ 分发路径里 token 的替身。
//
// token 等同于凭据:拿到它就能拉走规则文件正文。访问日志(gin 的 Logger →
// stdout)在接上日志聚合/落盘之后,日志里留着的 token 就是一份可用的凭据副本。
const ssRuleTokenMask = "[redacted]"

// redactSSRuleTokenPath 把 /ss/<token>/... 里的 token 换成占位符。
//
// 为什么必须动 c.Request.URL.Path:
//
//   - gin 的 Logger() 在**进入时**就把 c.Request.URL.Path 抓进局部变量,
//     c.Next() 返回之后才拿它拼日志 ⇒ 在 handler 里做任何补救都来不及;
//   - gin v1.9.1 的 LoggerConfig.SkipPaths 是**精确匹配**(不是前缀),
//     对 /ss/<token>/<file> 这种参数化路径根本匹配不上;
//   - 路由组级别的自定义 logger 同样没用 —— 全局 Logger 是先执行的,
//     路由组里的 handler 只能看到「已经打出来的日志」。
//
// 为什么改写 URL.Path 不会影响业务:gin 在 ServeHTTP 里先完成路由匹配,
// c.Param("token") 在 handler 链执行之前就解析好了 —— handler 取 token
// 用的是 c.Param,不是 URL.Path。
//
// 遮罩后的路径保留 "/ss/" 前缀与末段文件名:
//
//   - 前缀必须保留 —— registerStaticRoutes 的 NoRoute 里「/ss/ 一律 404」
//     那条判断靠它,前缀丢了 404 会掉进 SPA fallback 变成 200 + index.html;
//   - 末段文件名来自公开白名单(rules.acl / rules.txt / rules.list /
//     rules.conf),不是机密,留着才有排查价值。
func redactSSRuleTokenPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		if p := c.Request.URL.Path; strings.HasPrefix(p, "/ss/") {
			c.Request.URL.Path = maskSSRuleToken(p)
		}
		c.Next()
	}
}

// maskSSRuleToken 把 /ss/ 路径里的 token 段抹掉,只留末段文件名。
//
// 前置条件:path 以 "/ss/" 开头 ⇒ segments 至少是 ["", "ss", "<token>"]。
//
// token 恒在 "/ss/" 之后的**第一段**(路由形态 /ss/:token/:filename),
// 末段是文件名。中间段(编码斜杠等畸形形态)一并抹掉,避免残余。
func maskSSRuleToken(path string) string {
	segments := strings.Split(path, "/")
	tail := ""
	// 只有存在第 4 段("/ss/<token>/<file>")时,末段才可能是文件名;
	// "/ss/<token>" 形态没有文件名可留。
	if len(segments) > 3 {
		if last := segments[len(segments)-1]; last != "" {
			tail = "/" + last
		}
	}
	return "/ss/" + ssRuleTokenMask + tail
}

func registerPublicRoutes(router *gin.Engine, handlers *routeHandlers) {
	router.GET("/s/:id", handlers.shortURLHandler.Redirect)
	router.GET("/sub/proxy", handlers.proxyHandler.DownloadSubscription)
	router.GET("/sub/proxy/:type", handlers.proxyHandler.DownloadSubscription)
	router.Any("/mock/:id", handlers.mockAPIHandler.Execute)
	registerSSRulePublicRoutes(router, handlers.ssRuleHandler)
}

// registerSSRulePublicRoutes 注册 SS 规则文件的公开分发路由。
//
// 为什么要拆成独立函数:HTTP 级回归用例(package main)要复用**同一份**注册逻辑,
// 否则测试里手抄一遍路由就等于没有测到真实接线(接线漂移不会被发现)。
func registerSSRulePublicRoutes(router *gin.Engine, h *handlers.SSRuleHandler) {
	// nil 守卫:与 routes.RegisterSSRuleRoutes 对齐。
	//
	// 生产路径上 handler 恒非 nil(app_handlers.go 无条件构造),所以这不是活 bug;
	// 但将来若给 routeHandlers 加条件构造,这里会 nil 解引用 panic。
	//
	// 不挂载 ≠ 放行:未注册的 /ss/ 路径会落到 registerStaticRoutes 的 NoRoute,
	// 那里的「/ss/ 前缀一律 404」判断会兜住它们 ⇒ 客户端拿到 404,
	// 而不是 SPA fallback 的 200 + index.html。
	if h == nil {
		return
	}

	// 分发端独立限流桶:120 次/分钟/IP。
	distributeRL := handlers.NewSSRuleLimiter(handlers.SSRuleDistributeRateLimitPerMinute)

	// 正常分发端点。必须显式注册:非 /api 前缀的路径没注册就会落到 NoRoute
	// 被 SPA fallback 兜成 index.html,客户端拿到的是一坨 HTML,
	// 而报错信息完全指不到这里。
	router.GET("/ss/:token/:filename", distributeRL, h.Download)

	// 显式注册尾斜杠 / 空 filename / 少一段的形态并直返 404,用来"关闭"
	// /ss/ 前缀的 RedirectTrailingSlash 自动重定向。不注册的话 gin 会 301 到
	// 无尾斜杠形态,尾斜杠载荷就拿不到同款 404。
	//
	// 注意这三条必须指向 404 handler 而**不是** Download:尾斜杠是独立的空段,
	// 不进 :filename 参数,指过去会返回 200 + 规则正文 —— 比 301 更糟。
	router.GET("/ss/:token/:filename/", distributeRL, h.NotFound)
	router.GET("/ss/:token/", distributeRL, h.NotFound)
	router.GET("/ss/:token", distributeRL, h.NotFound)
}

func registerStaticRoutes(router *gin.Engine, distDir string) {
	// /assets 里的 chunk 是 hashed 文件名(immutable),可以永久缓存。
	// 但 index.html 必须每次校验,否则浏览器拿旧 index.html 引用旧 chunk → 用户看到老 bug。
	router.Static("/neon", distDir+"/neon")
	router.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.File(distDir + "/index.html")
	})
	router.StaticFile("/alipay.jpeg", distDir+"/alipay.jpeg")
	router.StaticFile("/wxpay.jpeg", distDir+"/wxpay.jpeg")
	router.StaticFile("/pregnancy-shortcut-192.png", distDir+"/pregnancy-shortcut-192.png")
	router.StaticFile("/pregnancy-shortcut-512.png", distDir+"/pregnancy-shortcut-512.png")

	router.GET("/manifest.webmanifest", func(c *gin.Context) {
		c.Header("Content-Type", "application/manifest+json; charset=utf-8")
		c.File(distDir + "/manifest.webmanifest")
	})
	router.GET("/sw.js", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File(distDir + "/sw.js")
	})

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/api" || strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在", "path": path})
			return
		}

		// /ss/ 前缀一律 404「规则文件不存在」。
		// 不加这条 → 未匹配上的 /ss/ 路径会落进下面的 SPA fallback,
		// 客户端拿到 200 + index.html,把一坨 HTML 当规则文件用 ——
		// 静默失败,比 404 更难排查。
		//
		// 只拦 /ss/ 前缀,不影响其它路径的 SPA 兜底行为。
		if strings.HasPrefix(path, "/ss/") {
			c.String(http.StatusNotFound, handlers.SSRuleNotFoundBody)
			return
		}

		// hashed chunk 文件名带 hash 即可长缓存;其他文件(包含 index.html 兜底)
		// 必须每次校验,避免浏览器拿旧版引用旧 chunk。
		if strings.HasPrefix(path, "/assets/") && (strings.Contains(path, "-") || strings.Contains(path, ".")) {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		}

		cleanPath := filepath.Clean(strings.TrimPrefix(path, "/"))
		if cleanPath != "." && cleanPath != "" && !strings.HasPrefix(cleanPath, "..") {
			filePath := filepath.Join(distDir, cleanPath)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				c.File(filePath)
				return
			}
		}

		c.File(distDir + "/index.html")
	})
}

func startTunnelProxyServer(cfg *config.Config, proxyHandler *handlers.ProxyHandler) {
	if cfg.Proxy.TunnelPort == "" {
		return
	}
	if cfg.Proxy.AdminPassword == "" {
		log.Printf("警告：proxy.tunnel_port 已配置但 proxy.admin_password 为空，代理端口将拒绝所有连接")
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in background goroutine: %v", r)
			}
		}()
		tunnelSrv := &http.Server{
			Addr: ":" + cfg.Proxy.TunnelPort,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				proxyHandler.Tunnel(w, req)
			}),
		}
		log.Printf("独立代理端口启动在 %s（供 NPS 映射）", cfg.Proxy.TunnelPort)
		if err := tunnelSrv.ListenAndServe(); err != nil {
			log.Printf("独立代理端口启动失败: %v", err)
		}
	}()
}

func newMainHTTPServer(port string, router *gin.Engine, proxyHandler *handlers.ProxyHandler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Minute,
		IdleTimeout:       120 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodConnect {
				proxyHandler.Tunnel(w, req)
				return
			}
			router.ServeHTTP(w, req)
		}),
	}
}
