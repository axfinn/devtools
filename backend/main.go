package main

import (
	"log"
	"os"
	"time"

	"devtools/handlers"
	"devtools/middleware"
	"devtools/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 配置
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/paste.db"
	}

	// 确保数据目录存在
	os.MkdirAll("./data", 0755)

	// 初始化数据库
	db, err := models.NewDB(dbPath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer db.Close()

	// 定期清理过期数据
	go func() {
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			count, err := db.CleanExpired()
			if err == nil && count > 0 {
				log.Printf("已清理 %d 条过期数据", count)
			}
		}
	}()

	// 初始化 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS 配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 全局限流（每 IP 每分钟 60 次请求）
	rateLimiter := middleware.NewRateLimiter(60, time.Minute)
	r.Use(rateLimiter.Middleware())

	// 内容大小限制（1MB）
	r.Use(middleware.ContentSizeLimiter(1024 * 1024))

	// 处理器
	pasteHandler := handlers.NewPasteHandler(db)

	// API 路由
	api := r.Group("/api")
	{
		// 粘贴板 API
		paste := api.Group("/paste")
		{
			paste.POST("", pasteHandler.Create)
			paste.GET("/:id", pasteHandler.Get)
			paste.GET("/:id/info", pasteHandler.GetInfo)
		}

		// 健康检查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	// 静态文件（生产环境）
	r.Static("/assets", "./dist/assets")
	r.StaticFile("/", "./dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})

	log.Printf("服务器启动在端口 %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
