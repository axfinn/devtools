package main

import (
	"devtools/handlers"
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// routeHandlers 包含所有处理器实例，用于路由注册
type routeHandlers struct {
	pasteHandler              *handlers.PasteHandler
	dnsHandler                *handlers.DNSHandler
	chatHandler               *handlers.ChatHandler
	shortURLHandler           *handlers.ShortURLHandler
	mockAPIHandler            *handlers.MockAPIHandler
	mdShareHandler            *handlers.MDShareHandler
	excalidrawHandler         *handlers.ExcalidrawHandler
	pregnancyHandler          *handlers.PregnancyHandler
	expenseHandler            *handlers.ExpenseHandler
	glucoseHandler            *handlers.GlucoseHandler
	plannerHandler            *handlers.PlannerHandler
	recipeHandler             *handlers.RecipeHandler
	householdHandler          *handlers.HouseholdHandler
	photoWallHandler          *handlers.PhotoWallHandler
	consoleHandler            *handlers.ConsoleHandler
	terminalHandler           *handlers.SSHHandler
	nfsShareHandler           *handlers.NFSShareHandler
	ocrHandler                *handlers.OCRHandler
	bailianHandler            *handlers.BailianHandler
	aiGatewayHandler          *handlers.AIGatewayHandler
	imageUnderstandingHandler *handlers.ImageUnderstandingHandler
	apiGatewayHandler         *handlers.APIGatewayHandler
	autoDevHandler            *handlers.AutoDevHandler
	mermaidHandler            *handlers.MermaidHandler
	npsHandler                *handlers.NPSHandler
	proxyHandler              *handlers.ProxyHandler
	hermesHandler             *handlers.HermesHandler
	edgeTTSHandler            *handlers.EdgeTTSHandler
	gameHandler               *handlers.GameHandler
}

func setupRoutes(api *gin.RouterGroup, createRateLimiter *middleware.RateLimiter, h *routeHandlers) {
	// code analysis handler is created inline since it has no dependencies
	// 粘贴板 API
	paste := api.Group("/paste")
	{
		// 只有创建操作需要限流
		paste.POST("", createRateLimiter.Middleware(), h.pasteHandler.Create)
		paste.POST("/upload", h.pasteHandler.UploadFile)        // 文件上传
		paste.GET("/files/:filename", h.pasteHandler.ServeFile) // 文件访问
		paste.GET("/:id", h.pasteHandler.Get)
		paste.GET("/:id/info", h.pasteHandler.GetInfo)

		// 分片上传 API
		paste.POST("/chunk/init", h.pasteHandler.InitChunkUpload)            // 初始化分片上传
		paste.POST("/chunk/:file_id", h.pasteHandler.UploadChunk)            // 上传分片
		paste.POST("/chunk/:file_id/merge", h.pasteHandler.MergeChunks)      // 合并分片
		paste.GET("/chunk/:file_id/status", h.pasteHandler.CheckChunkStatus) // 检查上传状态

		// 代码分析 API
		paste.POST("/analyze", h.pasteHandler.AnalyzeCode)         // 分析文本代码
		paste.GET("/analyze/:file_id", h.pasteHandler.AnalyzeFile) // 分析上传的文件

		// 安全扫描 API
		paste.POST("/scan", h.pasteHandler.ScanContent)              // 扫描内容安全
		paste.GET("/validate/:file_id", h.pasteHandler.ValidateFile) // 验证文件安全

		// 信息 API
		paste.GET("/languages", h.pasteHandler.GetSupportedLanguages)        // 支持的语言
		paste.GET("/content-types", h.pasteHandler.GetSupportedContentTypes) // 支持的内容类型
		paste.GET("/stats", h.pasteHandler.GetStats)                         // 统计信息

		// 搜索 API
		paste.GET("/search", h.pasteHandler.SearchPastes) // 搜索粘贴板

		// 管理员 API
		paste.GET("/admin/list", h.pasteHandler.AdminListPastes)    // 管理员列表
		paste.GET("/admin/:id", h.pasteHandler.AdminGetPaste)       // 管理员查看
		paste.PUT("/admin/:id", h.pasteHandler.AdminUpdatePaste)    // 管理员编辑
		paste.DELETE("/admin/:id", h.pasteHandler.AdminDeletePaste) // 管理员删除
	}

	// 代码分析 API
	analysisHandler := handlers.NewAnalysisHandler()
	analysis := api.Group("/analysis")
	{
		analysis.POST("/code", analysisHandler.AnalyzeCode)               // 代码分析
		analysis.POST("/scan", analysisHandler.ScanContent)               // 内容安全扫描
		analysis.POST("/validate", analysisHandler.ValidateFile)          // 文件验证
		analysis.GET("/languages", analysisHandler.GetSupportedLanguages) // 支持的语言列表
	}

	// IP/DNS API
	api.GET("/ip", h.dnsHandler.GetIP)
	api.GET("/dns", h.dnsHandler.Lookup)

	// 图片代理（解决跨域）
	api.GET("/proxy-image", handlers.ProxyImage)

	// 聊天室 API
	chat := api.Group("/chat")
	{
		chat.POST("/room", createRateLimiter.Middleware(), h.chatHandler.CreateRoom)
		chat.GET("/rooms", h.chatHandler.GetRooms)
		chat.GET("/room/:id", h.chatHandler.GetRoom)
		chat.GET("/room/:id/messages", h.chatHandler.GetRoomMessages) // 获取历史消息
		chat.POST("/room/:id/join", h.chatHandler.JoinRoom)
		chat.GET("/room/:id/ws", h.chatHandler.HandleWebSocket)
		// 图片上传
		chat.POST("/upload", createRateLimiter.Middleware(), h.chatHandler.UploadImage)
		chat.Static("/uploads", "./data/uploads")
		// 管理员 API
		chat.GET("/admin/rooms", h.chatHandler.AdminListRooms)
		chat.DELETE("/admin/room/:id", h.chatHandler.AdminDeleteRoom)
		// 机器人 API
		chat.GET("/room/:id/bot", h.chatHandler.GetBotConfig)
		chat.POST("/room/:id/bot", h.chatHandler.AddBot)
		chat.DELETE("/room/:id/bot", h.chatHandler.RemoveBot)
		chat.POST("/room/:id/bot/stop", h.chatHandler.StopBot)
	}

	// Edge TTS API
	edgeTTS := api.Group("/edge-tts")
	{
		edgeTTS.GET("/voices", h.edgeTTSHandler.ListVoices)
		edgeTTS.POST("/tts", h.edgeTTSHandler.Synthesize)
		edgeTTS.GET("/audio/:filename", h.edgeTTSHandler.ServeAudioFile)
		edgeTTS.GET("/health", h.edgeTTSHandler.Health)
		edgeTTS.POST("/convert", h.edgeTTSHandler.ConvertAudioFormat)
	}

	// 游戏 API
	game := api.Group("/game")
	{
		game.GET("/info", h.gameHandler.GetGameInfo)
		// 井字棋
		game.POST("/tictactoe/init", h.gameHandler.InitTicTacToe)
		game.POST("/tictactoe/move", h.gameHandler.MoveTicTacToe)
		// 五子棋
		game.POST("/gomoku/init", h.gameHandler.InitGomoku)
		game.POST("/gomoku/move", h.gameHandler.MoveGomoku)
		// 猜数字
		game.POST("/guessnumber/init", h.gameHandler.InitGuessNumber)
		game.POST("/guessnumber/guess", h.gameHandler.Guess)
		// 石头剪刀布
		game.POST("/rps/init", h.gameHandler.InitRPS)
		game.POST("/rps/play", h.gameHandler.PlayRPS)
		// 色子比大小
		game.POST("/dice/init", h.gameHandler.InitDice)
		game.POST("/dice/roll", h.gameHandler.RollDice)
		// AI 对话
		game.POST("/ai-chat", h.gameHandler.AIChat)
	}

	// 短链 API
	shorturl := api.Group("/shorturl")
	{
		shorturl.POST("", createRateLimiter.Middleware(), h.shortURLHandler.Create)
		shorturl.GET("/list", h.shortURLHandler.List)
		shorturl.GET("/:id/stats", h.shortURLHandler.GetStats)
		shorturl.PUT("/:id", h.shortURLHandler.Update)
		shorturl.DELETE("/:id", h.shortURLHandler.Delete)
	}

	// Mock API
	mockapi := api.Group("/mockapi")
	{
		mockapi.POST("", createRateLimiter.Middleware(), h.mockAPIHandler.Create)
		mockapi.GET("/:id", h.mockAPIHandler.Get)
		mockapi.GET("/:id/logs", h.mockAPIHandler.GetLogs)
		mockapi.PUT("/:id", h.mockAPIHandler.Update)
		mockapi.DELETE("/:id", h.mockAPIHandler.Delete)
	}

	// Markdown 分享 API
	mdshare := api.Group("/mdshare")
	{
		mdshare.POST("", createRateLimiter.Middleware(), h.mdShareHandler.Create)
		mdshare.GET("/:id", h.mdShareHandler.Get)
		mdshare.GET("/:id/creator", h.mdShareHandler.GetByCreator)
		mdshare.PUT("/:id", h.mdShareHandler.Update)
		mdshare.DELETE("/:id", h.mdShareHandler.Delete)
		// Admin routes
		mdshare.GET("/admin/list", h.mdShareHandler.List)
		mdshare.GET("/admin/:id", h.mdShareHandler.AdminGet)
		mdshare.DELETE("/admin/:id", h.mdShareHandler.AdminDelete)
	}

	// Excalidraw 画图 API
	excalidraw := api.Group("/excalidraw")
	{
		excalidraw.POST("", createRateLimiter.Middleware(), h.excalidrawHandler.Create)
		excalidraw.GET("/:id", h.excalidrawHandler.Get)
		excalidraw.GET("/:id/creator", h.excalidrawHandler.GetByCreator)
		excalidraw.PUT("/:id", h.excalidrawHandler.Update)
		excalidraw.DELETE("/:id", h.excalidrawHandler.Delete)
		// Admin routes
		excalidraw.GET("/admin/list", h.excalidrawHandler.List)
		excalidraw.GET("/admin/:id", h.excalidrawHandler.AdminGet)
		excalidraw.DELETE("/admin/:id", h.excalidrawHandler.AdminDelete)
	}

	// 孕期管理 API
	pregnancy := api.Group("/pregnancy")
	{
		pregnancy.POST("", createRateLimiter.Middleware(), h.pregnancyHandler.Create)
		pregnancy.POST("/login", h.pregnancyHandler.Login)
		pregnancy.GET("/:id", h.pregnancyHandler.Get)
		pregnancy.GET("/:id/creator", h.pregnancyHandler.GetByCreator)
		pregnancy.GET("/:id/device", h.pregnancyHandler.GetByDevice)
		pregnancy.PUT("/:id", h.pregnancyHandler.Update)
		pregnancy.DELETE("/:id", h.pregnancyHandler.Delete)
	}

	// 生活记账 API
	expense := api.Group("/expense")
	{
		expense.POST("", createRateLimiter.Middleware(), h.expenseHandler.Create)
		expense.POST("/login", h.expenseHandler.Login)
		expense.GET("/:id", h.expenseHandler.Get)
		expense.DELETE("/:id", h.expenseHandler.Delete)
		expense.PUT("/:id/extend", h.expenseHandler.Extend)
		// Accounts
		expense.GET("/:id/accounts", h.expenseHandler.GetAccounts)
		expense.POST("/:id/accounts", h.expenseHandler.CreateAccount)
		expense.PUT("/:id/accounts/:accountId", h.expenseHandler.UpdateAccount)
		expense.DELETE("/:id/accounts/:accountId", h.expenseHandler.DeleteAccount)
		// Categories
		expense.GET("/:id/categories", h.expenseHandler.GetCategories)
		expense.POST("/:id/categories", h.expenseHandler.CreateCategory)
		expense.PUT("/:id/categories/:categoryId", h.expenseHandler.UpdateCategory)
		expense.DELETE("/:id/categories/:categoryId", h.expenseHandler.DeleteCategory)
		// Transactions
		expense.GET("/:id/transactions", h.expenseHandler.GetTransactions)
		expense.POST("/:id/transactions", h.expenseHandler.CreateTransaction)
		expense.PUT("/:id/transactions/:txId", h.expenseHandler.UpdateTransaction)
		expense.DELETE("/:id/transactions/:txId", h.expenseHandler.DeleteTransaction)
		// Stats
		expense.GET("/:id/stats", h.expenseHandler.GetStats)
		// AI Analyze
		expense.POST("/:id/analyze", h.expenseHandler.Analyze)
		// AI Voice Parse
		expense.POST("/:id/voice-parse", h.expenseHandler.VoiceParse)
	}

	// 血糖监测 API
	glucose := api.Group("/glucose")
	{
		glucose.POST("", createRateLimiter.Middleware(), h.glucoseHandler.Create)
		glucose.POST("/login", h.glucoseHandler.Login)
		glucose.GET("/:id", h.glucoseHandler.Get)
		glucose.DELETE("/:id", h.glucoseHandler.Delete)
		glucose.PUT("/:id/extend", h.glucoseHandler.Extend)
		// Records
		glucose.GET("/:id/records", h.glucoseHandler.GetRecords)
		glucose.POST("/:id/records", h.glucoseHandler.CreateRecord)
		glucose.PUT("/:id/records/:recordId", h.glucoseHandler.UpdateRecord)
		glucose.DELETE("/:id/records/:recordId", h.glucoseHandler.DeleteRecord)
		// History
		glucose.GET("/:id/records/:recordId/history", h.glucoseHandler.GetRecordHistory)
		glucose.GET("/:id/history", h.glucoseHandler.GetProfileHistory)
		// Import
		glucose.POST("/:id/import", h.glucoseHandler.ImportRecords)
		// Stats
		glucose.GET("/:id/stats", h.glucoseHandler.GetStats)
		// AI Voice Parse
		glucose.POST("/:id/voice-parse", h.glucoseHandler.VoiceParse)
	}

	// 事项管理 API
	planner := api.Group("/planner")
	{
		planner.POST("/profile", createRateLimiter.Middleware(), h.plannerHandler.CreateProfile)
		planner.POST("/profile/login", h.plannerHandler.LoginProfile)
		planner.GET("/profile/:id", h.plannerHandler.GetProfile)
		planner.PUT("/profile/:id", h.plannerHandler.UpdateProfile)
		planner.DELETE("/profile/:id", h.plannerHandler.DeleteProfile)
		planner.GET("/profile/:id/timeline", h.plannerHandler.ListTimeline)
		planner.GET("/profile/:id/review", h.plannerHandler.Review)
		planner.POST("/profile/:id/tasks", h.plannerHandler.CreateTask)
		planner.POST("/profile/:id/tasks/batch", h.plannerHandler.CreateTaskBatch)
		planner.PUT("/profile/:id/tasks/:taskId", h.plannerHandler.UpdateTask)
		planner.DELETE("/profile/:id/tasks/:taskId", h.plannerHandler.DeleteTask)
		planner.GET("/profile/:id/tasks/:taskId/comments", h.plannerHandler.ListTaskComments)
		planner.POST("/profile/:id/tasks/:taskId/comments", h.plannerHandler.CreateTaskComment)
		planner.GET("/profile/:id/tasks/:taskId/activities", h.plannerHandler.ListTaskActivities)
		planner.GET("/profile/:id/tasks/:taskId/calendar", h.plannerHandler.DownloadCalendar)
		planner.POST("/profile/:id/ai/parse", h.plannerHandler.AIParse)
		planner.POST("/profile/:id/ai/advise", h.plannerHandler.AIAdvise)
		planner.GET("/admin/list", h.plannerHandler.AdminList)
		planner.GET("/admin/:id", h.plannerHandler.AdminGet)
		planner.PUT("/admin/:id", h.plannerHandler.AdminUpdate)
		planner.DELETE("/admin/:id", h.plannerHandler.AdminDelete)
	}

	// 菜谱 API
	recipe := api.Group("/recipe")
	{
		recipe.GET("/default", h.recipeHandler.GetDefault)   // 获取默认菜谱（无需登录）
		recipe.GET("/detailed", h.recipeHandler.GetDetailed) // 获取详细步骤菜谱（无需登录）
		recipe.POST("", createRateLimiter.Middleware(), h.recipeHandler.Create)
		recipe.POST("/login", h.recipeHandler.Login)
		recipe.GET("/:id", h.recipeHandler.Get)
		recipe.GET("/:id/creator", h.recipeHandler.GetByCreator)
		recipe.PUT("/:id", h.recipeHandler.Update)
		recipe.DELETE("/:id", h.recipeHandler.Delete)
	}

	// 家庭物品整理 API
	household := api.Group("/household")
	{
		household.POST("/init", h.householdHandler.Init)                     // 初始化
		household.GET("/items", h.householdHandler.GetItems)                 // 获取物品列表
		household.POST("/items", h.householdHandler.CreateItem)              // 创建物品
		household.GET("/items/:id", h.householdHandler.GetItem)              // 获取物品
		household.PUT("/items/:id", h.householdHandler.UpdateItem)           // 更新物品
		household.DELETE("/items/:id", h.householdHandler.DeleteItem)        // 删除物品
		household.POST("/items/:id/use", h.householdHandler.UseItem)         // 使用物品
		household.POST("/items/:id/restock", h.householdHandler.RestockItem) // 补充物品
		household.POST("/items/:id/open", h.householdHandler.OpenItem)       // 重新开封

		household.GET("/templates", h.householdHandler.GetTemplates)          // 获取模板
		household.POST("/templates", h.householdHandler.CreateTemplate)       // 创建模板
		household.DELETE("/templates/:id", h.householdHandler.DeleteTemplate) // 删除模板

		household.GET("/notifications", h.householdHandler.GetNotifications)                     // 获取通知
		household.POST("/notifications/:id/read", h.householdHandler.MarkNotificationAsRead)     // 标记已读
		household.POST("/notifications/read-all", h.householdHandler.MarkAllNotificationsAsRead) // 全部已读

		// 待购买/提醒任务
		household.GET("/todos", h.householdHandler.GetTodos)
		household.POST("/todos", h.householdHandler.CreateTodo)
		household.PUT("/todos/:id", h.householdHandler.UpdateTodo)
		household.DELETE("/todos/:id", h.householdHandler.DeleteTodo)

		household.GET("/stats", h.householdHandler.GetStats) // 统计信息

		// AI 智能功能
		household.GET("/ai/check", h.householdHandler.AIFeatureCheck)      // 检查 AI 功能
		household.POST("/ai/analyze", h.householdHandler.AIAnalyze)        // AI 分析库存
		household.POST("/ai/add", h.householdHandler.AIAddItem)            // AI 智能添加物品
		household.POST("/ai/parse", h.householdHandler.AIParseItems)       // AI 解析物品清单
		household.POST("/ai/todos/merge", h.householdHandler.AIMergeTodos) // AI 合并待办
		household.GET("/ai/restock", h.householdHandler.AISuggestRestock)  // AI 推荐补充

		// 对话功能
		household.POST("/chat", h.householdHandler.Chat)                       // 对话
		household.GET("/chat/history", h.householdHandler.GetChatHistory)      // 获取对话历史
		household.DELETE("/chat/history", h.householdHandler.ClearChatHistory) // 清除对话历史

		// 条码查询
		household.POST("/barcode/lookup", h.householdHandler.BarcodeLookup) // 条码查询

		// 小票 OCR 识别
		household.POST("/ocr/receipt", h.householdHandler.ReceiptOCR) // 小票识别

		// 档案管理 API
		household.POST("/profile", h.householdHandler.CreateProfile)           // 创建档案
		household.POST("/profile/login", h.householdHandler.LoginProfile)      // 登录档案
		household.GET("/profile/:id", h.householdHandler.GetProfile)           // 获取档案
		household.PUT("/profile/:id/extend", h.householdHandler.ExtendProfile) // 延长档案
		household.DELETE("/profile/:id", h.householdHandler.DeleteProfile)     // 删除档案

		// 档案物品管理 API
		household.POST("/profile/:id/items", h.householdHandler.CreateProfileItem)      // 创建档案物品
		household.GET("/profile/:id/items", h.householdHandler.GetProfileItems)         // 获取档案物品列表
		household.GET("/profile/:id/locations", h.householdHandler.GetProfileLocations) // 获取档案位置列表
		household.GET("/profile/:id/locations/library", h.householdHandler.GetLocationLibrary)
		household.POST("/profile/:id/locations/library", h.householdHandler.CreateLocation)
		household.PUT("/profile/:id/locations/library/:locId", h.householdHandler.UpdateLocation)
		household.DELETE("/profile/:id/locations/library/:locId", h.householdHandler.DeleteLocation)
		household.GET("/profile/:id/space", h.householdHandler.GetSpaceLayout)
		household.POST("/profile/:id/space", h.householdHandler.SaveSpaceLayout)
		household.POST("/profile/:id/space/share", h.householdHandler.CreateSpaceShare)
		household.GET("/space/share/:shareId", h.householdHandler.GetSpaceShare)
		household.PUT("/profile/:id/items/:itemId", h.householdHandler.UpdateProfileItem)           // 更新档案物品
		household.DELETE("/profile/:id/items/:itemId", h.householdHandler.DeleteProfileItem)        // 删除档案物品
		household.POST("/profile/:id/items/:itemId/use", h.householdHandler.UseProfileItem)         // 使用档案物品
		household.POST("/profile/:id/items/:itemId/restock", h.householdHandler.RestockProfileItem) // 补充档案物品
		household.POST("/profile/:id/items/:itemId/open", h.householdHandler.OpenProfileItem)       // 重新开封
	}

	// 档案照片墙 API
	photowall := api.Group("/photowall")
	{
		photowall.POST("/profile", createRateLimiter.Middleware(), h.photoWallHandler.CreateProfile)
		photowall.POST("/profile/login", h.photoWallHandler.LoginProfile)
		photowall.GET("/profile/:id", h.photoWallHandler.GetProfile)
		photowall.PUT("/profile/:id", h.photoWallHandler.UpdateProfile)
		photowall.DELETE("/profile/:id", h.photoWallHandler.DeleteProfile)
		photowall.POST("/profile/:id/items", createRateLimiter.Middleware(), h.photoWallHandler.UploadItem)
		photowall.PUT("/profile/:id/items/:itemId", h.photoWallHandler.UpdateItem)
		photowall.DELETE("/profile/:id/items/:itemId", h.photoWallHandler.DeleteItem)
		photowall.GET("/profile/:id/download", h.photoWallHandler.DownloadSelection)
		photowall.GET("/share/:id", h.photoWallHandler.GetShare)
		photowall.GET("/files/:filename", h.photoWallHandler.ServeFile)
		photowall.GET("/admin/list", h.photoWallHandler.AdminList)
		photowall.GET("/admin/:id", h.photoWallHandler.AdminGet)
		photowall.PUT("/admin/:id", h.photoWallHandler.AdminUpdate)
		photowall.DELETE("/admin/:id", h.photoWallHandler.AdminDelete)
	}

	// 终端 API
	terminal := api.Group("/terminal")
	{
		terminal.POST("", createRateLimiter.Middleware(), h.terminalHandler.Create)
		terminal.POST("/login", h.terminalHandler.Login)
		terminal.GET("/list", h.terminalHandler.List)                // 列出用户所有会话
		terminal.GET("/admin/list", h.terminalHandler.AdminList)     // 管理员查看所有
		terminal.DELETE("/admin/:id", h.terminalHandler.AdminDelete) // 管理员删除
		terminal.GET("/:id", h.terminalHandler.Get)
		terminal.GET("/:id/creator", h.terminalHandler.GetByCreator)
		terminal.GET("/:id/history", h.terminalHandler.GetHistory)     // 获取命令历史
		terminal.POST("/:id/resume", h.terminalHandler.Resume)         // 恢复会话
		terminal.POST("/:id/disconnect", h.terminalHandler.Disconnect) // 断开连接
		terminal.PUT("/:id", h.terminalHandler.Update)
		terminal.DELETE("/:id", h.terminalHandler.Delete)
		terminal.GET("/:id/ws", h.terminalHandler.HandleWebSocket) // WebSocket 连接
	}

	// NFS/SMB 文件分享 API
	nfsshare := api.Group("/nfsshare")
	{
		nfsshare.GET("/status", h.nfsShareHandler.Status)                       // 功能状态（公开）
		nfsshare.GET("/turn-credentials", h.nfsShareHandler.GetTurnCredentials) // TURN 临时凭证（公开）
		nfsshare.GET("/:id/info", h.nfsShareHandler.Info)                       // 分享信息（公开，不消耗次数）
		nfsshare.GET("/:id/stream", h.nfsShareHandler.Stream)                   // 原生视频流（公开，Range 支持）
		nfsshare.GET("/:id/qualities", h.nfsShareHandler.HLSQualities)          // 可用清晰度列表（公开）
		nfsshare.GET("/:id/hls/:quality/:segment", func(c *gin.Context) {     // HLS 转码播放（公开）
			if c.Param("segment") == "index.m3u8" {
				h.nfsShareHandler.HLSPlaylist(c)
			} else {
				h.nfsShareHandler.HLSSegment(c)
			}
		})
		nfsshare.GET("/:id", h.nfsShareHandler.Access)                                   // 访问/下载文件（公开，消耗次数）
		nfsshare.GET("/:id/watch/ws", h.nfsShareHandler.WatchWS)                         // 一起看 WebSocket（公开）
		nfsshare.POST("/:id/record", h.nfsShareHandler.UploadRecord)                     // 上传录音（公开，需分享密码）
		nfsshare.GET("/:id/record/:filename", h.nfsShareHandler.ServeRecord)             // 播放录音（超管）
		nfsshare.POST("", h.nfsShareHandler.Create)                                      // 创建分享（超管）
		nfsshare.GET("/admin/browse", h.nfsShareHandler.Browse)                          // 浏览目录（超管）
		nfsshare.GET("/admin/list", h.nfsShareHandler.AdminList)                         // 分享列表（超管）
		nfsshare.GET("/admin/mounts", h.nfsShareHandler.MountsList)                      // 挂载点列表及状态（超管）
		nfsshare.POST("/admin/mounts/:name/remount", h.nfsShareHandler.MountsRemount)    // 重新挂载（超管）
		nfsshare.POST("/admin/mounts/:name/umount", h.nfsShareHandler.MountsUmount)      // 卸载（超管）
		nfsshare.GET("/admin/:id/logs", h.nfsShareHandler.AdminGetLogs)                  // 访问日志（超管）
		nfsshare.PUT("/admin/:id", h.nfsShareHandler.AdminUpdate)                        // 修改配置（超管）
		nfsshare.DELETE("/admin/:id", h.nfsShareHandler.AdminDelete)                     // 删除分享（超管）
		nfsshare.POST("/admin/upload/init", h.nfsShareHandler.UploadInit)                // 初始化上传（超管）
		nfsshare.POST("/admin/upload/:token/chunk", h.nfsShareHandler.UploadChunk)       // 上传分片（超管）
		nfsshare.POST("/admin/upload/:token/complete", h.nfsShareHandler.UploadComplete) // 合并完成（超管）
	}

	// OCR 文字识别
	api.POST("/ocr", createRateLimiter.Middleware(), h.ocrHandler.Extract)

	// MiniMax MCP 图像理解
	imageUnderstanding := api.Group("/image-understanding")
	{
		imageUnderstanding.GET("/tools", h.imageUnderstandingHandler.ListTools)
		imageUnderstanding.POST("/describe", createRateLimiter.Middleware(), h.imageUnderstandingHandler.Describe)
		imageUnderstanding.POST("/describe-file", createRateLimiter.Middleware(), h.imageUnderstandingHandler.DescribeFromUpload)
		// SSE 模式（任务队列）
		imageUnderstanding.POST("/sse/create", createRateLimiter.Middleware(), h.imageUnderstandingHandler.CreateSseTask)
		imageUnderstanding.POST("/sse/create-file", createRateLimiter.Middleware(), h.imageUnderstandingHandler.CreateSseTaskFromFile)
		imageUnderstanding.GET("/sse/task/:id", h.imageUnderstandingHandler.GetSseTask)
		imageUnderstanding.GET("/sse/stream/:id", h.imageUnderstandingHandler.StreamSseTask)
		// Qwen 视觉理解（内部免 API Key，使用服务端 DashScope 配置）
		imageUnderstanding.POST("/qwen-vision", createRateLimiter.Middleware(), h.aiGatewayHandler.InternalQwenVision)
		imageUnderstanding.GET("/qwen-vision/logs", h.aiGatewayHandler.AdminListQwenVisionLogs)
	}

	// 百炼图片模型
	bailian := api.Group("/bailian")
	{
		bailian.GET("/docs", h.bailianHandler.GetDocs)
		bailian.GET("/models", h.bailianHandler.GetModels)
		bailian.POST("/tasks", createRateLimiter.Middleware(), h.bailianHandler.CreateTask)
		bailian.GET("/tasks", h.bailianHandler.ListTasks)
		bailian.GET("/tasks/:id/events", h.bailianHandler.GetTaskEvents)
		bailian.POST("/tasks/:id/poll", h.bailianHandler.PollTask)
		bailian.GET("/tasks/:id", h.bailianHandler.GetTask)
		bailian.POST("/generate", createRateLimiter.Middleware(), h.bailianHandler.OpenAPICreateTask)
	}

	// AI Gateway
	aigw := api.Group("/ai-gateway")
	{
		aigw.GET("/docs", h.aiGatewayHandler.GetDocs)
		aigw.GET("/docs/anthropic", h.aiGatewayHandler.GetAnthropicDocs)
		aigw.GET("/catalog", h.aiGatewayHandler.GetCatalog)
		aigw.POST("/admin/keys", h.aiGatewayHandler.AdminCreateKey)
		aigw.GET("/admin/keys", h.aiGatewayHandler.AdminListKeys)
		aigw.GET("/admin/keys/:id", h.aiGatewayHandler.AdminGetKey)
		aigw.POST("/admin/keys/:id/revoke", h.aiGatewayHandler.AdminRevokeKey)
		aigw.GET("/admin/logs", h.aiGatewayHandler.AdminListLogs)
		aigw.GET("/admin/reports", h.aiGatewayHandler.AdminReports)
		aigw.GET("/admin/alerts", h.aiGatewayHandler.AdminAlerts)
		aigw.POST("/admin/test-model", h.aiGatewayHandler.AdminTestModel)
		aigw.POST("/v1/chat/completions", h.aiGatewayHandler.ChatCompletions)
		aigw.POST("/v1/chat/tasks", h.aiGatewayHandler.AsyncChatCompletions)
		aigw.GET("/v1/chat/tasks/:id", h.aiGatewayHandler.GetChatTask)
		aigw.POST("/v1/media/generations", h.aiGatewayHandler.MediaGenerations)
		aigw.GET("/v1/media/tasks", h.aiGatewayHandler.ListMediaTasks)
		aigw.GET("/v1/media/tasks/:id", h.aiGatewayHandler.GetMediaTask)
	}

	// 内部免认证聊天接口（同域浏览器调用）
	api.POST("/internal/chat", createRateLimiter.Middleware(), h.aiGatewayHandler.InternalChat)

	// MiniMax Anthropic 协议代理
	api.POST("/minimax/anthropic/v1/messages", h.aiGatewayHandler.ProxyMinimaxAnthropic)

	// MiniMax TTS 语音合成代理
	api.POST("/minimax/tts/v1/generations", h.aiGatewayHandler.ProxyMinimaxTTS)
	api.GET("/minimax/tts/docs", h.aiGatewayHandler.GetTTSDocs)

	// MiniMax Token Plan 媒体生成代理（TTS HD / Hailuo 视频 / Music / image-01）
	api.POST("/minimax/token-plan/v1/generations", h.aiGatewayHandler.ProxyMinimaxTokenPlan)
	api.GET("/minimax/token-plan/docs", h.aiGatewayHandler.GetTokenPlanDocs)
	api.GET("/minimax/token-plan/tasks", h.aiGatewayHandler.ListMinimaxTokenPlanTasks)
	api.GET("/minimax/token-plan/tasks/:id", h.aiGatewayHandler.GetMinimaxTokenPlanTask)
	api.GET("/minimax/token-plan/tasks/:id/download", h.aiGatewayHandler.DownloadMinimaxTokenPlanTask)

	// MiniMax Music 工作流代理
	api.GET("/minimax/music/docs", h.aiGatewayHandler.GetMiniMaxMusicDocs)
	api.POST("/minimax/music/v1/lyrics_generation", h.aiGatewayHandler.MiniMaxLyricsGeneration)
	api.POST("/minimax/music/v1/cover_preprocess", h.aiGatewayHandler.MiniMaxMusicCoverPreprocess)

	// MiniMax 结果分享
	api.GET("/minimax/result-shares/docs", h.aiGatewayHandler.GetMiniMaxResultShareDocs)
	api.POST("/minimax/result-shares", h.aiGatewayHandler.CreateMiniMaxResultShare)
	api.GET("/minimax/result-shares/:id", h.aiGatewayHandler.GetMiniMaxResultShare)
	api.GET("/minimax/result-shares/:id/assets/:assetId", h.aiGatewayHandler.GetMiniMaxResultShareAsset)
	api.GET("/minimax/result-shares/admin/list", h.aiGatewayHandler.AdminListMiniMaxResultShares)
	api.GET("/minimax/result-shares/admin/:id", h.aiGatewayHandler.AdminGetMiniMaxResultShare)
	api.PUT("/minimax/result-shares/admin/:id", h.aiGatewayHandler.AdminUpdateMiniMaxResultShare)
	api.DELETE("/minimax/result-shares/admin/:id", h.aiGatewayHandler.AdminDeleteMiniMaxResultShare)

	// MiniMax Voice Cloning 音色克隆代理
	api.POST("/minimax/voice-cloning/upload", h.aiGatewayHandler.UploadVoiceClone)
	api.GET("/minimax/voice-cloning/voices", h.aiGatewayHandler.ListVoiceClones)
	api.DELETE("/minimax/voice-cloning/voices/:id", h.aiGatewayHandler.DeleteVoiceClone)
	api.POST("/minimax/voice-cloning/tts", h.aiGatewayHandler.TTSWithVoiceClone)
	api.GET("/minimax/voice-cloning/docs", h.aiGatewayHandler.GetVoiceCloningDocs)

	// MiniMax Speech 官方语音 HTTP 网关
	api.GET("/minimax/speech/docs", h.aiGatewayHandler.GetMiniMaxSpeechDocs)
	api.POST("/minimax/speech/v1/t2a_v2", h.aiGatewayHandler.MiniMaxSpeechSyncT2A)
	api.POST("/minimax/speech/v1/t2a_async_v2", h.aiGatewayHandler.MiniMaxSpeechAsyncCreate)
	api.GET("/minimax/speech/v1/query/t2a_async_query_v2", h.aiGatewayHandler.MiniMaxSpeechAsyncQuery)
	api.POST("/minimax/speech/v1/get_voice", h.aiGatewayHandler.MiniMaxSpeechGetVoice)
	api.POST("/minimax/speech/v1/voice_design", h.aiGatewayHandler.MiniMaxSpeechVoiceDesign)
	api.POST("/minimax/speech/v1/voice_clone", h.aiGatewayHandler.MiniMaxSpeechVoiceClone)
	api.POST("/minimax/speech/v1/delete_voice", h.aiGatewayHandler.MiniMaxSpeechDeleteVoice)
	api.POST("/minimax/speech/v1/files/upload", h.aiGatewayHandler.MiniMaxSpeechUploadFile)
	api.GET("/minimax/speech/v1/files/list", h.aiGatewayHandler.MiniMaxSpeechListFiles)
	api.GET("/minimax/speech/v1/files/retrieve", h.aiGatewayHandler.MiniMaxSpeechRetrieveFile)
	api.GET("/minimax/speech/v1/files/retrieve_content", h.aiGatewayHandler.MiniMaxSpeechRetrieveFileContent)
	api.POST("/minimax/speech/v1/files/delete", h.aiGatewayHandler.MiniMaxSpeechDeleteFile)
	api.GET("/minimax/speech/tasks", h.aiGatewayHandler.MiniMaxSpeechListTasks)
	api.GET("/minimax/speech/tasks/:id", h.aiGatewayHandler.MiniMaxSpeechGetTask)

	// DashScope Anthropic 协议代理
	api.POST("/dashscope/anthropic/v1/messages", h.aiGatewayHandler.ProxyDashScopeAnthropic)

	// DeepSeek Anthropic 协议代理
	api.POST("/deepseek/anthropic/v1/messages", h.aiGatewayHandler.ProxyDeepSeekAnthropic)

	// 通用 API Gateway 代理
	apigw := api.Group("/api-gateway")
	{
		apigw.Any("/cpa/v1", h.apiGatewayHandler.ProxyCPA)
		apigw.Any("/cpa/v1/*proxyPath", h.apiGatewayHandler.ProxyCPA)
		apigw.POST("/v1/image/understanding", h.apiGatewayHandler.ImageUnderstanding)
		apigw.POST("/v1/image/understanding/file", h.apiGatewayHandler.ImageUnderstandingFile)
		// SSE 模式
		apigw.POST("/v1/image/understanding/sse", h.apiGatewayHandler.ImageUnderstandingSSE)
		apigw.POST("/v1/image/understanding/sse/file", h.apiGatewayHandler.ImageUnderstandingSSEFile)
		apigw.GET("/v1/image/understanding/stream/:id", h.apiGatewayHandler.ImageUnderstandingStream)
	}

	// AutoDev AI 任务
	autodev := api.Group("/autodev")
	{
		autodev.POST("/verify", h.autoDevHandler.VerifyPassword)
		autodev.GET("/capabilities", h.autoDevHandler.GetCapabilities)
		autodev.POST("/tasks", h.autoDevHandler.Submit)
		autodev.GET("/tasks", h.autoDevHandler.List)
		autodev.GET("/projects", h.autoDevHandler.ListProjects)
		autodev.GET("/tasks/:id", h.autoDevHandler.GetTask)
		autodev.GET("/tasks/:id/state", h.autoDevHandler.GetState)
		autodev.GET("/tasks/:id/files", h.autoDevHandler.GetFiles)
		autodev.GET("/tasks/:id/file", h.autoDevHandler.GetFile)
		autodev.GET("/tasks/:id/raw", h.autoDevHandler.GetRawFile)
		autodev.GET("/tasks/:id/logs", h.autoDevHandler.GetLogs)
		autodev.GET("/tasks/:id/download", h.autoDevHandler.Download)
		autodev.GET("/tasks/:id/site/*filepath", h.autoDevHandler.GetSite)
		autodev.POST("/tasks/:id/stop", h.autoDevHandler.StopTask)
		autodev.POST("/tasks/:id/terminate", h.autoDevHandler.TerminateTask)
		autodev.DELETE("/tasks/:id", h.autoDevHandler.DeleteTask)
		// Ask API - quick Q&A
		autodev.POST("/ask", h.autoDevHandler.Ask)
		autodev.GET("/ask/:id", h.autoDevHandler.GetAskResult)
		// Extend API - extend existing project with new requirements
		autodev.POST("/extend", h.autoDevHandler.Extend)
		// Init API - stream CLAUDE.md generation via SSE (GET with query params)
		autodev.GET("/init/stream", h.autoDevHandler.InitProject)
		// SSH key for GitHub access
		autodev.GET("/sshkey", h.autoDevHandler.GetSSHKey)
		autodev.POST("/sshkey/regenerate", h.autoDevHandler.RegenerateSSHKey)
		// Claude / Codex CLI 版本管理
		autodev.GET("/claude/version", h.autoDevHandler.GetClaudeVersion)
		autodev.GET("/claude/cli/test", h.autoDevHandler.TestClaudeCLI)
		autodev.GET("/claude/test", h.autoDevHandler.TestModel)
		autodev.GET("/claude/update/stream", h.autoDevHandler.UpdateClaude)
		autodev.GET("/codex/version", h.autoDevHandler.GetCodexVersion)
		autodev.GET("/codex/cli/test", h.autoDevHandler.TestCodexCLI)
		autodev.GET("/codex/update/stream", h.autoDevHandler.UpdateCodex)
		// clawtest 版本管理
		autodev.GET("/clawtest/version", h.autoDevHandler.GetClawtestVersion)
		autodev.GET("/clawtest/update/stream", h.autoDevHandler.UpdateClawtest)
	}

	// 背景图 API
	api.GET("/bg", handlers.GetBackgroundImages)

	// Mermaid 档案 + AI 生图
	mermaid := api.Group("/mermaid")
	{
		mermaid.POST("/projects", h.mermaidHandler.CreateProject)
		mermaid.GET("/projects", h.mermaidHandler.ListProjects)
		mermaid.DELETE("/projects/:id", h.mermaidHandler.DeleteProject)
		mermaid.POST("/projects/:id/versions", h.mermaidHandler.SaveVersion)
		mermaid.GET("/projects/:id/versions", h.mermaidHandler.ListVersions)
		mermaid.POST("/projects/:id/generate", h.mermaidHandler.AIGenerate)
		mermaid.GET("/projects/:id/messages", h.mermaidHandler.ListMessages)
		mermaid.DELETE("/projects/:id/messages", h.mermaidHandler.ClearMessages)
	}

	// 科学上网代理
	proxyGroup := api.Group("/proxy")
	{
		proxyGroup.POST("/config", h.proxyHandler.LoadConfig)
		proxyGroup.POST("/speedtest", h.proxyHandler.SpeedTest)
		proxyGroup.POST("/start", h.proxyHandler.Start)
		proxyGroup.POST("/auto-start", h.proxyHandler.AutoStart)
		proxyGroup.POST("/stop", h.proxyHandler.Stop)
		proxyGroup.POST("/check", h.proxyHandler.CheckNodes)
		proxyGroup.POST("/subscription-refresh", h.proxyHandler.TriggerSubscriptionRefresh)
		proxyGroup.POST("/nps-tunnel", h.proxyHandler.CreateNPSTunnel)
		proxyGroup.GET("/status", h.proxyHandler.Status)
		proxyGroup.GET("/fetch", h.proxyHandler.Fetch)
		proxyGroup.GET("/resource", h.proxyHandler.Resource)
		proxyGroup.GET("/extension", h.proxyHandler.DownloadExtension)
		proxyGroup.GET("/ws-tunnel", h.proxyHandler.WsTunnel)
		proxyGroup.GET("/client/download", h.proxyHandler.DownloadClient)
		proxyGroup.GET("/custom-domains", h.proxyHandler.ListCustomDomains)
		proxyGroup.POST("/custom-domains", h.proxyHandler.AddCustomDomain)
		proxyGroup.DELETE("/custom-domains", h.proxyHandler.RemoveCustomDomain)
	}

	// NPS 端口映射管理
	nps := api.Group("/nps")
	{
		nps.GET("/status", h.npsHandler.Status)
		nps.GET("/tunnels", h.npsHandler.ListTunnels)
		nps.POST("/tunnels", h.npsHandler.AddTunnel)
		nps.DELETE("/tunnels/:id", h.npsHandler.DeleteTunnel)
	}

	// Hermes Agent 接入
	hermes := api.Group("/hermes")
	{
		hermes.POST("/verify", h.hermesHandler.VerifyPassword)
		hermes.GET("/status", h.hermesHandler.Status)
		hermes.GET("/models", h.hermesHandler.Models)
		hermes.POST("/chat", h.hermesHandler.Chat)
	}
	api.POST("/bg/cache", handlers.CacheBackgroundImages) // 缓存图片
	api.POST("/bg/replace", handlers.ReplaceRandomImages) // 随机替换图片
	api.GET("/bg/random", handlers.GetRandomBackground)   // 随机图片
	api.GET("/bg/cached/:filename", handlers.ServeCachedBackground)

	// 健康检查
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 控制台设置
	api.GET("/console/settings", h.consoleHandler.GetSettings)
	api.POST("/console/settings", h.consoleHandler.SaveSettings)
	api.POST("/console/verify", h.consoleHandler.VerifyPassword)
}
