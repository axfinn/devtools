package main

import (
	"log"
	"os"
	"strings"
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

	// 初始化生活记账数据库表
	if err := db.InitExpense(); err != nil {
		log.Fatalf("生活记账数据库初始化失败: %v", err)
	}

	// 初始化血糖监测数据库表
	if err := db.InitGlucose(); err != nil {
		log.Fatalf("血糖监测数据库初始化失败: %v", err)
	}

	// 初始化菜谱数据库表
	if err := db.InitRecipe(); err != nil {
		log.Fatalf("菜谱数据库初始化失败: %v", err)
	}

	// 初始化家庭物品数据库表
	if err := db.InitHousehold(); err != nil {
		log.Fatalf("家庭物品数据库初始化失败: %v", err)
	}

	// 初始化 SSH 数据库表
	if err := db.InitSSH(); err != nil {
		log.Fatalf("SSH 数据库初始化失败: %v", err)
	}

	// 初始化百炼图片任务数据库表
	if err := db.InitBailian(); err != nil {
		log.Fatalf("百炼图片任务数据库初始化失败: %v", err)
	}

	// 初始化 AI Gateway 数据库表
	if err := db.InitAIGateway(); err != nil {
		log.Fatalf("AI Gateway 数据库初始化失败: %v", err)
	}

	// 初始化 LLM 异步任务表
	if err := db.InitLLMTasks(); err != nil {
		log.Fatalf("LLM Tasks 数据库初始化失败: %v", err)
	}

	// 初始化 NFS 分享数据库表
	if err := db.InitNFSShare(); err != nil {
		log.Fatalf("NFS 分享数据库初始化失败: %v", err)
	}

	// 初始化 AutoDev 任务数据库表
	if err := db.InitAutoDevTasks(); err != nil {
		log.Fatalf("AutoDev 任务数据库初始化失败: %v", err)
	}

	// 后台预加载背景图（如果缓存目录为空）
	go func() {
		// 等待服务器启动完成
		time.Sleep(3 * time.Second)
		// 初始化背景图（触发预加载）
		handlers.InitBackgroundImages()
		log.Println("背景图预加载完成")
	}()

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
			// 清理过期生活记账档案
			expenseCount, err := db.CleanExpiredExpenseProfiles()
			if err == nil && expenseCount > 0 {
				log.Printf("已清理 %d 个过期生活记账档案", expenseCount)
			}
			// 清理过期血糖档案
			glucoseCount, err := db.CleanExpiredGlucoseProfiles()
			if err == nil && glucoseCount > 0 {
				log.Printf("已清理 %d 个过期血糖监测档案", glucoseCount)
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
			// 清理旧的百炼任务
			bailianCount, err := db.CleanOldBailianTasks(cfg.Bailian.TaskRetentionDays)
			if err == nil && bailianCount > 0 {
				log.Printf("已清理 %d 条旧百炼任务", bailianCount)
			}
			// 清理旧的 AI Gateway 请求明细
			aiLogCount, err := db.CleanOldAIAPIRequestLogs(cfg.AIGateway.RequestRetentionDays)
			if err == nil && aiLogCount > 0 {
				log.Printf("已清理 %d 条旧 AI Gateway 请求明细", aiLogCount)
			}
			// 清理过期/耗尽的 NFS 分享
			nfsCount, err := db.CleanExpiredNFSShares()
			if err == nil && nfsCount > 0 {
				log.Printf("已清理 %d 个过期 NFS 分享", nfsCount)
			}
			// 清理孤立的 HLS 转码缓存（分享已删除但目录残留）
			if entries, err := os.ReadDir("./data/transcode"); err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					if _, err := db.GetNFSShare(e.Name()); err != nil {
						handlers.CleanHLSCache(e.Name())
					}
				}
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
	chatHandler := handlers.NewChatHandler(db, cfg.Chat.AdminPassword, cfg.MiniMax, cfg.Chat.TTSServiceURL)
	shortURLHandler := handlers.NewShortURLHandler(db, cfg.ShortURL.Password)
	mockAPIHandler := handlers.NewMockAPIHandler(db)
	mdShareHandler := handlers.NewMDShareHandler(db, cfg.MDShare.AdminPassword, cfg.MDShare.DefaultMaxViews, cfg.MDShare.DefaultExpiresDays)
	excalidrawHandler := handlers.NewExcalidrawHandler(db, cfg.Excalidraw.AdminPassword, cfg.Excalidraw.DefaultExpiresDays, cfg.Excalidraw.MaxContentSize)
	pregnancyHandler := handlers.NewPregnancyHandler(db, cfg.Pregnancy.DefaultExpiresDays, cfg.Pregnancy.MaxDataSize)
	expenseHandler := handlers.NewExpenseHandler(db, cfg)
	glucoseHandler := handlers.NewGlucoseHandler(db, cfg)
	recipeHandler := handlers.NewRecipeHandler(db, 365, 1024*1024)
	householdHandler := handlers.NewHouseholdHandler(db, cfg)

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
	nfsShareHandler := handlers.NewNFSShareHandler(db, cfg.NFSShare)
	ocrHandler := handlers.NewOCRHandler()
	bailianHandler := handlers.NewBailianHandler(db, cfg)
	aiGatewayHandler := handlers.NewAIGatewayHandler(db, cfg, bailianHandler)
	imageUnderstandingHandler := handlers.NewImageUnderstandingHandler(cfg)
	apiGatewayHandler := handlers.NewAPIGatewayHandler(aiGatewayHandler, imageUnderstandingHandler)
	autoDevHandler := handlers.NewAutoDevHandler(db, cfg.AutoDev.AdminPassword, cfg.AutoDev.AutodevPath, cfg.AutoDev.DataDir)

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
			paste.POST("/upload", pasteHandler.UploadFile)        // 文件上传
			paste.GET("/files/:filename", pasteHandler.ServeFile) // 文件访问
			paste.GET("/:id", pasteHandler.Get)
			paste.GET("/:id/info", pasteHandler.GetInfo)

			// 分片上传 API
			paste.POST("/chunk/init", pasteHandler.InitChunkUpload)            // 初始化分片上传
			paste.POST("/chunk/:file_id", pasteHandler.UploadChunk)            // 上传分片
			paste.POST("/chunk/:file_id/merge", pasteHandler.MergeChunks)      // 合并分片
			paste.GET("/chunk/:file_id/status", pasteHandler.CheckChunkStatus) // 检查上传状态

			// 代码分析 API
			paste.POST("/analyze", pasteHandler.AnalyzeCode)             // 分析文本代码
			paste.GET("/analyze/:file_id", pasteHandler.AnalyzeFile)    // 分析上传的文件

			// 安全扫描 API
			paste.POST("/scan", pasteHandler.ScanContent)                   // 扫描内容安全
			paste.GET("/validate/:file_id", pasteHandler.ValidateFile)    // 验证文件安全

			// 信息 API
			paste.GET("/languages", pasteHandler.GetSupportedLanguages)    // 支持的语言
			paste.GET("/content-types", pasteHandler.GetSupportedContentTypes) // 支持的内容类型
			paste.GET("/stats", pasteHandler.GetStats)                    // 统计信息

			// 搜索 API
			paste.GET("/search", pasteHandler.SearchPastes)               // 搜索粘贴板

			// 管理员 API
			paste.GET("/admin/list", pasteHandler.AdminListPastes)    // 管理员列表
			paste.GET("/admin/:id", pasteHandler.AdminGetPaste)       // 管理员查看
			paste.PUT("/admin/:id", pasteHandler.AdminUpdatePaste)    // 管理员编辑
			paste.DELETE("/admin/:id", pasteHandler.AdminDeletePaste) // 管理员删除
		}

		// 代码分析 API
		analysisHandler := handlers.NewAnalysisHandler()
		analysis := api.Group("/analysis")
		{
			analysis.POST("/code", analysisHandler.AnalyzeCode)     // 代码分析
			analysis.POST("/scan", analysisHandler.ScanContent)    // 内容安全扫描
			analysis.POST("/validate", analysisHandler.ValidateFile) // 文件验证
			analysis.GET("/languages", analysisHandler.GetSupportedLanguages) // 支持的语言列表
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
			// 机器人 API
			chat.GET("/room/:id/bot", chatHandler.GetBotConfig)
			chat.POST("/room/:id/bot", chatHandler.AddBot)
			chat.DELETE("/room/:id/bot", chatHandler.RemoveBot)
			chat.POST("/room/:id/bot/stop", chatHandler.StopBot)
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

		// 生活记账 API
		expense := api.Group("/expense")
		{
			expense.POST("", createRateLimiter.Middleware(), expenseHandler.Create)
			expense.POST("/login", expenseHandler.Login)
			expense.GET("/:id", expenseHandler.Get)
			expense.DELETE("/:id", expenseHandler.Delete)
			expense.PUT("/:id/extend", expenseHandler.Extend)
			// Accounts
			expense.GET("/:id/accounts", expenseHandler.GetAccounts)
			expense.POST("/:id/accounts", expenseHandler.CreateAccount)
			expense.PUT("/:id/accounts/:accountId", expenseHandler.UpdateAccount)
			expense.DELETE("/:id/accounts/:accountId", expenseHandler.DeleteAccount)
			// Categories
			expense.GET("/:id/categories", expenseHandler.GetCategories)
			expense.POST("/:id/categories", expenseHandler.CreateCategory)
			expense.PUT("/:id/categories/:categoryId", expenseHandler.UpdateCategory)
			expense.DELETE("/:id/categories/:categoryId", expenseHandler.DeleteCategory)
			// Transactions
			expense.GET("/:id/transactions", expenseHandler.GetTransactions)
			expense.POST("/:id/transactions", expenseHandler.CreateTransaction)
			expense.PUT("/:id/transactions/:txId", expenseHandler.UpdateTransaction)
			expense.DELETE("/:id/transactions/:txId", expenseHandler.DeleteTransaction)
			// Stats
			expense.GET("/:id/stats", expenseHandler.GetStats)
			// AI Analyze
			expense.POST("/:id/analyze", expenseHandler.Analyze)
			// AI Voice Parse
			expense.POST("/:id/voice-parse", expenseHandler.VoiceParse)
		}

		// 血糖监测 API
		glucose := api.Group("/glucose")
		{
			glucose.POST("", createRateLimiter.Middleware(), glucoseHandler.Create)
			glucose.POST("/login", glucoseHandler.Login)
			glucose.GET("/:id", glucoseHandler.Get)
			glucose.DELETE("/:id", glucoseHandler.Delete)
			glucose.PUT("/:id/extend", glucoseHandler.Extend)
			// Records
			glucose.GET("/:id/records", glucoseHandler.GetRecords)
			glucose.POST("/:id/records", glucoseHandler.CreateRecord)
			glucose.PUT("/:id/records/:recordId", glucoseHandler.UpdateRecord)
			glucose.DELETE("/:id/records/:recordId", glucoseHandler.DeleteRecord)
			// History
			glucose.GET("/:id/records/:recordId/history", glucoseHandler.GetRecordHistory)
			glucose.GET("/:id/history", glucoseHandler.GetProfileHistory)
			// Import
			glucose.POST("/:id/import", glucoseHandler.ImportRecords)
			// Stats
			glucose.GET("/:id/stats", glucoseHandler.GetStats)
			// AI Voice Parse
			glucose.POST("/:id/voice-parse", glucoseHandler.VoiceParse)
		}

		// 菜谱 API
		recipe := api.Group("/recipe")
		{
			recipe.GET("/default", recipeHandler.GetDefault)   // 获取默认菜谱（无需登录）
			recipe.GET("/detailed", recipeHandler.GetDetailed) // 获取详细步骤菜谱（无需登录）
			recipe.POST("", createRateLimiter.Middleware(), recipeHandler.Create)
			recipe.POST("/login", recipeHandler.Login)
			recipe.GET("/:id", recipeHandler.Get)
			recipe.GET("/:id/creator", recipeHandler.GetByCreator)
			recipe.PUT("/:id", recipeHandler.Update)
			recipe.DELETE("/:id", recipeHandler.Delete)
		}

		// 家庭物品整理 API
		household := api.Group("/household")
		{
			household.POST("/init", householdHandler.Init)                     // 初始化
			household.GET("/items", householdHandler.GetItems)                 // 获取物品列表
			household.POST("/items", householdHandler.CreateItem)              // 创建物品
			household.GET("/items/:id", householdHandler.GetItem)              // 获取物品
			household.PUT("/items/:id", householdHandler.UpdateItem)           // 更新物品
			household.DELETE("/items/:id", householdHandler.DeleteItem)        // 删除物品
			household.POST("/items/:id/use", householdHandler.UseItem)         // 使用物品
			household.POST("/items/:id/restock", householdHandler.RestockItem) // 补充物品
			household.POST("/items/:id/open", householdHandler.OpenItem)       // 重新开封

			household.GET("/templates", householdHandler.GetTemplates)          // 获取模板
			household.POST("/templates", householdHandler.CreateTemplate)       // 创建模板
			household.DELETE("/templates/:id", householdHandler.DeleteTemplate) // 删除模板

			household.GET("/notifications", householdHandler.GetNotifications)                     // 获取通知
			household.POST("/notifications/:id/read", householdHandler.MarkNotificationAsRead)     // 标记已读
			household.POST("/notifications/read-all", householdHandler.MarkAllNotificationsAsRead) // 全部已读

			// 待购买/提醒任务
			household.GET("/todos", householdHandler.GetTodos)
			household.POST("/todos", householdHandler.CreateTodo)
			household.PUT("/todos/:id", householdHandler.UpdateTodo)
			household.DELETE("/todos/:id", householdHandler.DeleteTodo)

			household.GET("/stats", householdHandler.GetStats) // 统计信息

			// AI 智能功能
			household.GET("/ai/check", householdHandler.AIFeatureCheck)      // 检查 AI 功能
			household.POST("/ai/analyze", householdHandler.AIAnalyze)        // AI 分析库存
			household.POST("/ai/add", householdHandler.AIAddItem)            // AI 智能添加物品
			household.POST("/ai/parse", householdHandler.AIParseItems)       // AI 解析物品清单
			household.POST("/ai/todos/merge", householdHandler.AIMergeTodos) // AI 合并待办
			household.GET("/ai/restock", householdHandler.AISuggestRestock)  // AI 推荐补充

			// 对话功能
			household.POST("/chat", householdHandler.Chat)                       // 对话
			household.GET("/chat/history", householdHandler.GetChatHistory)      // 获取对话历史
			household.DELETE("/chat/history", householdHandler.ClearChatHistory) // 清除对话历史

			// 条码查询
			household.POST("/barcode/lookup", householdHandler.BarcodeLookup) // 条码查询

			// 小票 OCR 识别
			household.POST("/ocr/receipt", householdHandler.ReceiptOCR) // 小票识别

			// 档案管理 API
			household.POST("/profile", householdHandler.CreateProfile)           // 创建档案
			household.POST("/profile/login", householdHandler.LoginProfile)      // 登录档案
			household.GET("/profile/:id", householdHandler.GetProfile)           // 获取档案
			household.PUT("/profile/:id/extend", householdHandler.ExtendProfile) // 延长档案
			household.DELETE("/profile/:id", householdHandler.DeleteProfile)     // 删除档案

			// 档案物品管理 API
			household.POST("/profile/:id/items", householdHandler.CreateProfileItem)      // 创建档案物品
			household.GET("/profile/:id/items", householdHandler.GetProfileItems)         // 获取档案物品列表
			household.GET("/profile/:id/locations", householdHandler.GetProfileLocations) // 获取档案位置列表
			household.GET("/profile/:id/locations/library", householdHandler.GetLocationLibrary)
			household.POST("/profile/:id/locations/library", householdHandler.CreateLocation)
			household.PUT("/profile/:id/locations/library/:locId", householdHandler.UpdateLocation)
			household.DELETE("/profile/:id/locations/library/:locId", householdHandler.DeleteLocation)
			household.GET("/profile/:id/space", householdHandler.GetSpaceLayout)
			household.POST("/profile/:id/space", householdHandler.SaveSpaceLayout)
			household.POST("/profile/:id/space/share", householdHandler.CreateSpaceShare)
			household.GET("/space/share/:shareId", householdHandler.GetSpaceShare)
			household.PUT("/profile/:id/items/:itemId", householdHandler.UpdateProfileItem)           // 更新档案物品
			household.DELETE("/profile/:id/items/:itemId", householdHandler.DeleteProfileItem)        // 删除档案物品
			household.POST("/profile/:id/items/:itemId/use", householdHandler.UseProfileItem)         // 使用档案物品
			household.POST("/profile/:id/items/:itemId/restock", householdHandler.RestockProfileItem) // 补充档案物品
			household.POST("/profile/:id/items/:itemId/open", householdHandler.OpenProfileItem)       // 重新开封
		}

		// 终端 API
		terminal := api.Group("/terminal")
		{
			terminal.POST("", createRateLimiter.Middleware(), terminalHandler.Create)
			terminal.POST("/login", terminalHandler.Login)
			terminal.GET("/list", terminalHandler.List)                // 列出用户所有会话
			terminal.GET("/admin/list", terminalHandler.AdminList)     // 管理员查看所有
			terminal.DELETE("/admin/:id", terminalHandler.AdminDelete) // 管理员删除
			terminal.GET("/:id", terminalHandler.Get)
			terminal.GET("/:id/creator", terminalHandler.GetByCreator)
			terminal.GET("/:id/history", terminalHandler.GetHistory)     // 获取命令历史
			terminal.POST("/:id/resume", terminalHandler.Resume)         // 恢复会话
			terminal.POST("/:id/disconnect", terminalHandler.Disconnect) // 断开连接
			terminal.PUT("/:id", terminalHandler.Update)
			terminal.DELETE("/:id", terminalHandler.Delete)
			terminal.GET("/:id/ws", terminalHandler.HandleWebSocket) // WebSocket 连接
		}

		// NFS/SMB 文件分享 API
		nfsshare := api.Group("/nfsshare")
		{
			nfsshare.GET("/status", nfsShareHandler.Status)                      // 功能状态（公开）
			nfsshare.GET("/turn-credentials", nfsShareHandler.GetTurnCredentials) // TURN 临时凭证（公开）
			nfsshare.GET("/:id/info", nfsShareHandler.Info)                      // 分享信息（公开，不消耗次数）
			nfsshare.GET("/:id/stream", nfsShareHandler.Stream)                   // 原生视频流（公开，Range 支持）
			nfsshare.GET("/:id/qualities", nfsShareHandler.HLSQualities)           // 可用清晰度列表（公开）
			nfsshare.GET("/:id/hls/:quality/:segment", func(c *gin.Context) {     // HLS 转码播放（公开）
				if c.Param("segment") == "index.m3u8" {
					nfsShareHandler.HLSPlaylist(c)
				} else {
					nfsShareHandler.HLSSegment(c)
				}
			})
			nfsshare.GET("/:id", nfsShareHandler.Access)                         // 访问/下载文件（公开，消耗次数）
			nfsshare.GET("/:id/watch/ws", nfsShareHandler.WatchWS)              // 一起看 WebSocket（公开）
			nfsshare.POST("", nfsShareHandler.Create)                            // 创建分享（超管）
			nfsshare.GET("/admin/browse", nfsShareHandler.Browse)                // 浏览目录（超管）
			nfsshare.GET("/admin/list", nfsShareHandler.AdminList)               // 分享列表（超管）
			nfsshare.GET("/admin/mounts", nfsShareHandler.MountsList)            // 挂载点列表及状态（超管）
			nfsshare.POST("/admin/mounts/:name/remount", nfsShareHandler.MountsRemount) // 重新挂载（超管）
			nfsshare.POST("/admin/mounts/:name/umount", nfsShareHandler.MountsUmount)   // 卸载（超管）
			nfsshare.GET("/admin/:id/logs", nfsShareHandler.AdminGetLogs)        // 访问日志（超管）
			nfsshare.PUT("/admin/:id", nfsShareHandler.AdminUpdate)              // 修改配置（超管）
			nfsshare.DELETE("/admin/:id", nfsShareHandler.AdminDelete)           // 删除分享（超管）
		}

		// OCR 文字识别
		api.POST("/ocr", createRateLimiter.Middleware(), ocrHandler.Extract)

		// MiniMax MCP 图像理解
		imageUnderstanding := api.Group("/image-understanding")
		{
			imageUnderstanding.GET("/tools", imageUnderstandingHandler.ListTools)
			imageUnderstanding.POST("/describe", createRateLimiter.Middleware(), imageUnderstandingHandler.Describe)
			imageUnderstanding.POST("/describe-file", createRateLimiter.Middleware(), imageUnderstandingHandler.DescribeFromUpload)
			// SSE 模式（任务队列）
			imageUnderstanding.POST("/sse/create", createRateLimiter.Middleware(), imageUnderstandingHandler.CreateSseTask)
			imageUnderstanding.POST("/sse/create-file", createRateLimiter.Middleware(), imageUnderstandingHandler.CreateSseTaskFromFile)
			imageUnderstanding.GET("/sse/task/:id", imageUnderstandingHandler.GetSseTask)
			imageUnderstanding.GET("/sse/stream/:id", imageUnderstandingHandler.StreamSseTask)
			// Qwen 视觉理解（内部免 API Key，使用服务端 DashScope 配置）
			imageUnderstanding.POST("/qwen-vision", createRateLimiter.Middleware(), aiGatewayHandler.InternalQwenVision)
			imageUnderstanding.GET("/qwen-vision/logs", aiGatewayHandler.AdminListQwenVisionLogs)
		}

		// 百炼图片模型
		bailian := api.Group("/bailian")
		{
			bailian.GET("/docs", bailianHandler.GetDocs)
			bailian.GET("/models", bailianHandler.GetModels)
			bailian.POST("/tasks", createRateLimiter.Middleware(), bailianHandler.CreateTask)
			bailian.GET("/tasks", bailianHandler.ListTasks)
			bailian.GET("/tasks/:id/events", bailianHandler.GetTaskEvents)
			bailian.POST("/tasks/:id/poll", bailianHandler.PollTask)
			bailian.GET("/tasks/:id", bailianHandler.GetTask)
			bailian.POST("/generate", createRateLimiter.Middleware(), bailianHandler.OpenAPICreateTask)
		}

		// AI Gateway
		aigw := api.Group("/ai-gateway")
		{
			aigw.GET("/docs", aiGatewayHandler.GetDocs)
			aigw.GET("/docs/anthropic", aiGatewayHandler.GetAnthropicDocs)
			aigw.GET("/catalog", aiGatewayHandler.GetCatalog)
			aigw.POST("/admin/keys", aiGatewayHandler.AdminCreateKey)
			aigw.GET("/admin/keys", aiGatewayHandler.AdminListKeys)
			aigw.GET("/admin/keys/:id", aiGatewayHandler.AdminGetKey)
			aigw.POST("/admin/keys/:id/revoke", aiGatewayHandler.AdminRevokeKey)
			aigw.GET("/admin/logs", aiGatewayHandler.AdminListLogs)
			aigw.GET("/admin/reports", aiGatewayHandler.AdminReports)
			aigw.GET("/admin/alerts", aiGatewayHandler.AdminAlerts)
			aigw.POST("/admin/test-model", aiGatewayHandler.AdminTestModel)
			aigw.POST("/v1/chat/completions", aiGatewayHandler.ChatCompletions)
			aigw.POST("/v1/chat/tasks", aiGatewayHandler.AsyncChatCompletions)
			aigw.GET("/v1/chat/tasks/:id", aiGatewayHandler.GetChatTask)
			aigw.POST("/v1/media/generations", aiGatewayHandler.MediaGenerations)
			aigw.GET("/v1/media/tasks", aiGatewayHandler.ListMediaTasks)
			aigw.GET("/v1/media/tasks/:id", aiGatewayHandler.GetMediaTask)
		}

		// MiniMax Anthropic 协议代理
		api.POST("/minimax/anthropic/v1/messages", aiGatewayHandler.ProxyMinimaxAnthropic)

		// MiniMax TTS 语音合成代理
		api.POST("/minimax/tts/v1/generations", aiGatewayHandler.ProxyMinimaxTTS)
		api.GET("/minimax/tts/docs", aiGatewayHandler.GetTTSDocs)

		// MiniMax Token Plan 媒体生成代理（TTS HD / Hailuo 视频 / Music / image-01）
		api.POST("/minimax/token-plan/v1/generations", aiGatewayHandler.ProxyMinimaxTokenPlan)
		api.GET("/minimax/token-plan/docs", aiGatewayHandler.GetTokenPlanDocs)

		// DashScope Anthropic 协议代理
		api.POST("/dashscope/anthropic/v1/messages", aiGatewayHandler.ProxyDashScopeAnthropic)

		// 通用 API Gateway 代理
		apigw := api.Group("/api-gateway")
		{
			apigw.Any("/cpa/v1", apiGatewayHandler.ProxyCPA)
			apigw.Any("/cpa/v1/*proxyPath", apiGatewayHandler.ProxyCPA)
			apigw.POST("/v1/image/understanding", apiGatewayHandler.ImageUnderstanding)
			apigw.POST("/v1/image/understanding/file", apiGatewayHandler.ImageUnderstandingFile)
			// SSE 模式
			apigw.POST("/v1/image/understanding/sse", apiGatewayHandler.ImageUnderstandingSSE)
			apigw.POST("/v1/image/understanding/sse/file", apiGatewayHandler.ImageUnderstandingSSEFile)
			apigw.GET("/v1/image/understanding/stream/:id", apiGatewayHandler.ImageUnderstandingStream)
		}

		// AutoDev AI 任务
		autodev := api.Group("/autodev")
		{
			autodev.POST("/verify", autoDevHandler.VerifyPassword)
			autodev.POST("/tasks", autoDevHandler.Submit)
			autodev.GET("/tasks", autoDevHandler.List)
			autodev.GET("/projects", autoDevHandler.ListProjects)
			autodev.GET("/tasks/:id", autoDevHandler.GetTask)
			autodev.GET("/tasks/:id/state", autoDevHandler.GetState)
			autodev.GET("/tasks/:id/files", autoDevHandler.GetFiles)
			autodev.GET("/tasks/:id/file", autoDevHandler.GetFile)
			autodev.GET("/tasks/:id/logs", autoDevHandler.GetLogs)
			autodev.GET("/tasks/:id/download", autoDevHandler.Download)
			autodev.GET("/tasks/:id/site/*filepath", autoDevHandler.GetSite)
			autodev.POST("/tasks/:id/stop", autoDevHandler.StopTask)
			autodev.DELETE("/tasks/:id", autoDevHandler.DeleteTask)
			// Ask API - quick Q&A
			autodev.POST("/ask", autoDevHandler.Ask)
			autodev.GET("/ask/:id", autoDevHandler.GetAskResult)
			// Extend API - extend existing project with new requirements
			autodev.POST("/extend", autoDevHandler.Extend)
			// Init API - stream CLAUDE.md generation via SSE (GET with query params)
			autodev.GET("/init/stream", autoDevHandler.InitProject)
			// SSH key for GitHub access
			autodev.GET("/sshkey", autoDevHandler.GetSSHKey)
			autodev.POST("/sshkey/regenerate", autoDevHandler.RegenerateSSHKey)
			// Claude CLI 版本管理
			autodev.GET("/claude/version", autoDevHandler.GetClaudeVersion)
			autodev.GET("/claude/test", autoDevHandler.TestModel)
			autodev.GET("/claude/update/stream", autoDevHandler.UpdateClaude)
			// clawtest 版本管理
			autodev.GET("/clawtest/version", autoDevHandler.GetClawtestVersion)
			autodev.GET("/clawtest/update/stream", autoDevHandler.UpdateClawtest)
		}

		// 背景图 API
		api.GET("/bg", handlers.GetBackgroundImages)
		api.POST("/bg/cache", handlers.CacheBackgroundImages) // 缓存图片
		api.POST("/bg/replace", handlers.ReplaceRandomImages) // 随机替换图片
		api.GET("/bg/random", handlers.GetRandomBackground)   // 随机图片
		api.GET("/bg/cached/:filename", handlers.ServeCachedBackground)

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
		path := c.Request.URL.Path
		if path == "/api" || strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"error": "接口不存在", "path": path})
			return
		}
		c.File("./dist/index.html")
	})

	log.Printf("服务器启动在端口 %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
