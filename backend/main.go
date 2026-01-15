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

	// 初始化聊天室数据库表
	if err := db.InitChat(); err != nil {
		log.Fatalf("聊天室数据库初始化失败: %v", err)
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

	// 内容大小限制（35MB，留余量给 base64 编码）
	r.Use(middleware.ContentSizeLimiter(35 * 1024 * 1024))

	// 创建限流（仅用于上传，每 IP 每分钟 10 次）
	createRateLimiter := middleware.NewRateLimiter(10, time.Minute)

	// 处理器
	pasteHandler := handlers.NewPasteHandler(db)
	dnsHandler := handlers.NewDNSHandler()
	chatHandler := handlers.NewChatHandler(db)

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
		}

		// 健康检查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

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
