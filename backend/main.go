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

	// 初始化 Markdown 分享数据库表
	if err := db.InitMDShare(); err != nil {
		log.Fatalf("Markdown 分享数据库初始化失败: %v", err)
	}

	// 初始化 Excalidraw 数据库表
	if err := db.InitExcalidraw(); err != nil {
		log.Fatalf("Excalidraw 数据库初始化失败: %v", err)
	}

	// 初始化孕期管理数据库表
	if err := db.InitPregnancy(); err != nil {
		log.Fatalf("孕期管理数据库初始化失败: %v", err)
	}

	// 初始化菜谱数据库表
	if err := db.InitRecipe(); err != nil {
		log.Fatalf("菜谱数据库初始化失败: %v", err)
	}

	// 初始化 SSH 数据库表
	if err := db.InitSSH(); err != nil {
		log.Fatalf("SSH 数据库初始化失败: %v", err)
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
			// 清理过期 Markdown 分享
			mdCount, err := db.CleanExpiredMDShares()
			if err == nil && mdCount > 0 {
				log.Printf("已清理 %d 个过期 Markdown 分享", mdCount)
			}
			// 清理过期 Excalidraw 画图
			excalidrawCount, err := db.CleanExpiredExcalidrawShares()
			if err == nil && excalidrawCount > 0 {
				log.Printf("已清理 %d 个过期 Excalidraw 画图", excalidrawCount)
			}
			// 清理过期孕期档案
			pregnancyCount, err := db.CleanExpiredPregnancyProfiles()
			if err == nil && pregnancyCount > 0 {
				log.Printf("已清理 %d 个过期孕期档案", pregnancyCount)
			}
			// 清理过期菜谱
			recipeCount, err := db.CleanExpiredRecipes()
			if err == nil && recipeCount > 0 {
				log.Printf("已清理 %d 个过期菜谱", recipeCount)
			}
			// 清理过期 SSH 会话
			sshExpiredCount, err := db.CleanExpiredSSHSessions()
			if err == nil && sshExpiredCount > 0 {
				log.Printf("已清理 %d 个过期 SSH 会话", sshExpiredCount)
			}
			// 清理不活跃的 SSH 会话
			sshInactiveCount, err := db.CleanInactiveSSHSessions(cfg.SSH.SessionMaxAgeDays)
			if err == nil && sshInactiveCount > 0 {
				log.Printf("已清理 %d 个不活跃 SSH 会话（超过%d天）", sshInactiveCount, cfg.SSH.SessionMaxAgeDays)
			}
			// 清理旧的 SSH 历史记录
			historyCount, err := db.CleanOldSSHHistory(cfg.SSH.HistoryMaxAgeDays)
			if err == nil && historyCount > 0 {
				log.Printf("已清理 %d 条旧 SSH 历史记录（超过%d天）", historyCount, cfg.SSH.HistoryMaxAgeDays)
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
	chatHandler := handlers.NewChatHandler(db, cfg.Chat.AdminPassword)
	shortURLHandler := handlers.NewShortURLHandler(db, cfg.ShortURL.Password)
	mockAPIHandler := handlers.NewMockAPIHandler(db)
	mdShareHandler := handlers.NewMDShareHandler(db, cfg.MDShare.AdminPassword, cfg.MDShare.DefaultMaxViews, cfg.MDShare.DefaultExpiresDays)
	excalidrawHandler := handlers.NewExcalidrawHandler(db, cfg.Excalidraw.AdminPassword, cfg.Excalidraw.DefaultExpiresDays, cfg.Excalidraw.MaxContentSize)
	pregnancyHandler := handlers.NewPregnancyHandler(db, cfg.Pregnancy.DefaultExpiresDays, cfg.Pregnancy.MaxDataSize)
	recipeHandler := handlers.NewRecipeHandler(db, 365, 1024*1024)

	// 创建加密服务（用于 SSH 密码加密）
	// 优先使用配置文件中设置的密钥，如果没有则使用环境变量
	encryptionKey := cfg.SSH.EncryptionKey
	if encryptionKey == "" {
		encryptionKey = os.Getenv("TERMINAL_ENCRYPTION_KEY")
	}
	if encryptionKey == "" {
		// 生成随机密钥并提示用户
		randomKey, _ := utils.GenerateRandomKey()
		log.Printf("WARNING: 未设置加密密钥，使用临时随机密钥，重启后之前保存的 SSH 密码将无法解密")
		log.Printf("WARNING: 请在配置文件中设置 ssh.encryption_key 或设置环境变量 TERMINAL_ENCRYPTION_KEY")
		encryptionKey = randomKey
	}
	encryptionService, err := utils.NewEncryptionService(encryptionKey)
	if err != nil {
		log.Fatalf("创建加密服务失败: %v", err)
	}

	// 创建 SSH Handler 配置
	sshConfig := &handlers.SSHHandlerConfig{
		AdminPassword:       cfg.SSH.AdminPassword,
		HostKeyVerification: cfg.SSH.HostKeyVerification,
		MaxSessionsPerUser:  cfg.SSH.MaxSessionsPerUser,
		SessionIdleTimeout:  time.Duration(cfg.SSH.SessionIdleTimeout) * time.Minute,
	}
	terminalHandler := handlers.NewSSHHandler(db, encryptionService, sshConfig)

	// 启动 SSH 会话清理协程
	terminalHandler.StartCleanupRoutine()

	// API 路由
	api := r.Group("/api")
	{
		// 粘贴板 API
		paste := api.Group("/paste")
		{
			// 只有创建操作需要限流
			paste.POST("", createRateLimiter.Middleware(), pasteHandler.Create)
			paste.POST("/upload", pasteHandler.UploadFile)           // 文件上传
			paste.GET("/files/:filename", pasteHandler.ServeFile)    // 文件访问
			paste.GET("/:id", pasteHandler.Get)
			paste.GET("/:id/info", pasteHandler.GetInfo)

			// 分片上传 API
			paste.POST("/chunk/init", pasteHandler.InitChunkUpload)       // 初始化分片上传
			paste.POST("/chunk/:file_id", pasteHandler.UploadChunk)       // 上传分片
			paste.POST("/chunk/:file_id/merge", pasteHandler.MergeChunks) // 合并分片
			paste.GET("/chunk/:file_id/status", pasteHandler.CheckChunkStatus) // 检查上传状态

			// 管理员 API
			paste.GET("/admin/list", pasteHandler.AdminListPastes)   // 管理员列表
			paste.GET("/admin/:id", pasteHandler.AdminGetPaste)      // 管理员查看
			paste.PUT("/admin/:id", pasteHandler.AdminUpdatePaste)   // 管理员编辑
			paste.DELETE("/admin/:id", pasteHandler.AdminDeletePaste) // 管理员删除
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
			chat.GET("/room/:id/messages", chatHandler.GetRoomMessages) // 获取历史消息
			chat.POST("/room/:id/join", chatHandler.JoinRoom)
			chat.GET("/room/:id/ws", chatHandler.HandleWebSocket)
			// 图片上传
			chat.POST("/upload", createRateLimiter.Middleware(), chatHandler.UploadImage)
			chat.Static("/uploads", "./data/uploads")
			// 管理员 API
			chat.GET("/admin/rooms", chatHandler.AdminListRooms)
			chat.DELETE("/admin/room/:id", chatHandler.AdminDeleteRoom)
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

		// Markdown 分享 API
		mdshare := api.Group("/mdshare")
		{
			mdshare.POST("", createRateLimiter.Middleware(), mdShareHandler.Create)
			mdshare.GET("/:id", mdShareHandler.Get)
			mdshare.GET("/:id/creator", mdShareHandler.GetByCreator)
			mdshare.PUT("/:id", mdShareHandler.Update)
			mdshare.DELETE("/:id", mdShareHandler.Delete)
			// Admin routes
			mdshare.GET("/admin/list", mdShareHandler.List)
			mdshare.GET("/admin/:id", mdShareHandler.AdminGet)
			mdshare.DELETE("/admin/:id", mdShareHandler.AdminDelete)
		}

		// Excalidraw 画图 API
		excalidraw := api.Group("/excalidraw")
		{
			excalidraw.POST("", createRateLimiter.Middleware(), excalidrawHandler.Create)
			excalidraw.GET("/:id", excalidrawHandler.Get)
			excalidraw.GET("/:id/creator", excalidrawHandler.GetByCreator)
			excalidraw.PUT("/:id", excalidrawHandler.Update)
			excalidraw.DELETE("/:id", excalidrawHandler.Delete)
			// Admin routes
			excalidraw.GET("/admin/list", excalidrawHandler.List)
			excalidraw.GET("/admin/:id", excalidrawHandler.AdminGet)
			excalidraw.DELETE("/admin/:id", excalidrawHandler.AdminDelete)
		}

		// 孕期管理 API
		pregnancy := api.Group("/pregnancy")
		{
			pregnancy.POST("", createRateLimiter.Middleware(), pregnancyHandler.Create)
			pregnancy.POST("/login", pregnancyHandler.Login)
			pregnancy.GET("/:id", pregnancyHandler.Get)
			pregnancy.GET("/:id/creator", pregnancyHandler.GetByCreator)
			pregnancy.PUT("/:id", pregnancyHandler.Update)
			pregnancy.DELETE("/:id", pregnancyHandler.Delete)
		}

		// 菜谱 API
		recipe := api.Group("/recipe")
		{
			recipe.GET("/default", recipeHandler.GetDefault)                    // 获取默认菜谱（无需登录）
			recipe.GET("/detailed", recipeHandler.GetDetailed)                  // 获取详细步骤菜谱（无需登录）
			recipe.POST("", createRateLimiter.Middleware(), recipeHandler.Create)
			recipe.POST("/login", recipeHandler.Login)
			recipe.GET("/:id", recipeHandler.Get)
			recipe.GET("/:id/creator", recipeHandler.GetByCreator)
			recipe.PUT("/:id", recipeHandler.Update)
			recipe.DELETE("/:id", recipeHandler.Delete)
		}

		// 终端 API
		terminal := api.Group("/terminal")
		{
			terminal.POST("", createRateLimiter.Middleware(), terminalHandler.Create)
			terminal.POST("/login", terminalHandler.Login)
			terminal.GET("/list", terminalHandler.List) // 列出用户所有会话
			terminal.GET("/admin/list", terminalHandler.AdminList)   // 管理员查看所有
			terminal.DELETE("/admin/:id", terminalHandler.AdminDelete) // 管理员删除
			terminal.GET("/:id", terminalHandler.Get)
			terminal.GET("/:id/creator", terminalHandler.GetByCreator)
			terminal.GET("/:id/history", terminalHandler.GetHistory)      // 获取命令历史
			terminal.POST("/:id/resume", terminalHandler.Resume)          // 恢复会话
			terminal.POST("/:id/disconnect", terminalHandler.Disconnect)  // 断开连接
			terminal.PUT("/:id", terminalHandler.Update)
			terminal.DELETE("/:id", terminalHandler.Delete)
			terminal.GET("/:id/ws", terminalHandler.HandleWebSocket) // WebSocket 连接
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
