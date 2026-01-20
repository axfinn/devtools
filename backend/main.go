package main

import (
	"log"
	"os"
	"time"

	"devtools/config"
	"devtools/handlers"
	"devtools/middleware"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置文件
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("加载配置文件失败，使用默认配置: %v", err)
		cfg = config.Get()
	}

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

	// 初始化聊天室数据库表
	if err := db.InitChat(); err != nil {
		log.Fatalf("聊天室数据库初始化失败: %v", err)
	}

	// 初始化短链数据库表
	if err := db.InitShortURL(); err != nil {
		log.Fatalf("短链数据库初始化失败: %v", err)
	}

	// 初始化 Mock API 数据库表
	if err := db.InitMockAPI(); err != nil {
		log.Fatalf("Mock API 数据库初始化失败: %v", err)
	}

	// 定期清理过期数据
	go func() {
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			// 清理过期粘贴板
			count, err := db.CleanExpired()
			if err == nil && count > 0 {
				log.Printf("已清理 %d 条过期粘贴板", count)
			}
			// 清理过期聊天室（7天不活跃）
			roomCount, err := db.CleanExpiredRooms(7)
			if err == nil && roomCount > 0 {
				log.Printf("已清理 %d 个过期聊天室", roomCount)
			}
			// 清理过期消息（7天）
			msgCount, err := db.CleanExpiredMessages(7)
			if err == nil && msgCount > 0 {
				log.Printf("已清理 %d 条过期消息", msgCount)
			}
			// 清理过期短链
			err = db.CleanExpiredShortURLs()
			if err == nil {
				log.Printf("已清理过期短链")
			}
			// 清理过期 Mock APIs
			err = db.CleanExpiredMockAPIs()
			if err == nil {
				log.Printf("已清理过期 Mock APIs")
			}
			// 清理过期上传文件（7天）
			uploadCount, err := utils.CleanExpiredUploads("./data/uploads", 7)
			if err == nil && uploadCount > 0 {
				log.Printf("已清理 %d 个过期上传文件", uploadCount)
			}
		}
	}()

	// 初始化 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS 配置
	// 注意: AllowOrigins: "*" 与 AllowCredentials: true 不兼容
	// 如需凭证支持，请设置具体的 AllowOrigins
	r.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))

	// 内容大小限制（55MB，支持50MB视频上传）
	r.Use(middleware.ContentSizeLimiter(55 * 1024 * 1024))

	// 创建限流（仅用于上传，每 IP 每分钟 10 次）
	createRateLimiter := middleware.NewRateLimiter(10, time.Minute)

	// 处理器
	pasteHandler := handlers.NewPasteHandler(db)
	dnsHandler := handlers.NewDNSHandler()
	chatHandler := handlers.NewChatHandler(db)
	shortURLHandler := handlers.NewShortURLHandler(db, cfg.ShortURL.Password)
	mockAPIHandler := handlers.NewMockAPIHandler(db)

	// API 路由
	api := r.Group("/api")
	{
		// 粘贴板 API
		paste := api.Group("/paste")
		{
			// 只有创建操作需要限流
			paste.POST("", createRateLimiter.Middleware(), pasteHandler.Create)
			paste.GET("/:id", pasteHandler.Get)
			paste.GET("/:id/info", pasteHandler.GetInfo)
		}

		// IP/DNS API
		api.GET("/ip", dnsHandler.GetIP)
		api.GET("/dns", dnsHandler.Lookup)

		// 聊天室 API
		chat := api.Group("/chat")
		{
			chat.POST("/room", createRateLimiter.Middleware(), chatHandler.CreateRoom)
			chat.GET("/rooms", chatHandler.GetRooms)
			chat.GET("/room/:id", chatHandler.GetRoom)
			chat.POST("/room/:id/join", chatHandler.JoinRoom)
			chat.GET("/room/:id/ws", chatHandler.HandleWebSocket)
			// 图片上传
			chat.POST("/upload", createRateLimiter.Middleware(), chatHandler.UploadImage)
			chat.Static("/uploads", "./data/uploads")
		}

		// 短链 API
		shorturl := api.Group("/shorturl")
		{
			shorturl.POST("", createRateLimiter.Middleware(), shortURLHandler.Create)
			shorturl.GET("/list", shortURLHandler.List)
			shorturl.GET("/:id/stats", shortURLHandler.GetStats)
		}

		// Mock API
		mockapi := api.Group("/mockapi")
		{
			mockapi.POST("", createRateLimiter.Middleware(), mockAPIHandler.Create)
			mockapi.GET("/:id", mockAPIHandler.Get)
			mockapi.GET("/:id/logs", mockAPIHandler.GetLogs)
			mockapi.PUT("/:id", mockAPIHandler.Update)
			mockapi.DELETE("/:id", mockAPIHandler.Delete)
		}

		// 健康检查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	// 短链重定向（非 API 路径）
	r.GET("/s/:id", shortURLHandler.Redirect)

	// Mock API 执行（非 API 路径，支持所有 HTTP 方法）
	r.Any("/mock/:id", mockAPIHandler.Execute)

	// 静态文件（生产环境）
	r.Static("/assets", "./dist/assets")
	r.StaticFile("/", "./dist/index.html")
	r.StaticFile("/alipay.jpeg", "./dist/alipay.jpeg")
	r.StaticFile("/wxpay.jpeg", "./dist/wxpay.jpeg")
	r.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})

	log.Printf("服务器启动在端口 %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
