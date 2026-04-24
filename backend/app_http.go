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

	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))
	router.Use(middleware.ContentSizeLimiter(200 * 1024 * 1024))

	createRateLimiter := middleware.NewRateLimiter(10, time.Minute, rt.transientStore)
	api := router.Group("/api")
	setupRoutes(api, createRateLimiter, handlers)

	registerPublicRoutes(router, handlers)
	registerStaticRoutes(router, "./dist")

	return router
}

func registerPublicRoutes(router *gin.Engine, handlers *routeHandlers) {
	router.GET("/s/:id", handlers.shortURLHandler.Redirect)
	router.Any("/mock/:id", handlers.mockAPIHandler.Execute)
}

func registerStaticRoutes(router *gin.Engine, distDir string) {
	router.Static("/assets", distDir+"/assets")
	router.Static("/neon", distDir+"/neon")
	router.StaticFile("/", distDir+"/index.html")
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
		Addr: ":" + port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodConnect {
				proxyHandler.Tunnel(w, req)
				return
			}
			router.ServeHTTP(w, req)
		}),
	}
}
