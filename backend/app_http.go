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
	router := gin.Default()

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
