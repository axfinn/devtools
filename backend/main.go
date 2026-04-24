package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devtools/config"
	"devtools/handlers"
	"devtools/middleware"
	"devtools/models"
	"devtools/state"
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

	transientStore, err := state.New(cfg.Redis)
	if err != nil {
		log.Fatalf("Redis 初始化失败: %v", err)
	}

	imageTaskTTL := time.Duration(cfg.Redis.ImageTaskTTLHours) * time.Hour
	if imageTaskTTL <= 0 {
		imageTaskTTL = 7 * 24 * time.Hour
	}
	if err := transientStore.FailProcessingImageTasks(
		context.Background(),
		"服务已重启，任务中断，请重新提交",
		imageTaskTTL,
	); err != nil {
		log.Printf("恢复图像任务状态失败: %v", err)
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


	// 初始化所有数据库表（每个 model 通过 init() 自动注册）
	if err := db.InitAll(); err != nil {
		log.Fatalf("%v", err)
	}

	// 后台预加载背景图（如果缓存目录为空）
	go func() {
		// 等待服务器启动完成
		time.Sleep(3 * time.Second)
		// 初始化背景图（触发预加载）
		handlers.InitBackgroundImages()
		log.Println("背景图预加载完成")
	}()

	var plannerHandler *handlers.PlannerHandler


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

	// 内容大小限制（支持大文件上传）
	r.Use(middleware.ContentSizeLimiter(200 * 1024 * 1024))

	// 创建限流（仅用于上传，每 IP 每分钟 10 次）
	createRateLimiter := middleware.NewRateLimiter(10, time.Minute, transientStore)

	// 处理器
	pasteHandler := handlers.NewPasteHandler(db, cfg, transientStore)
	dnsHandler := handlers.NewDNSHandler()
	chatHandler := handlers.NewChatHandler(db, cfg.Chat.AdminPassword, cfg.MiniMax, cfg.Chat.TTSServiceURL)
	shortURLHandler := handlers.NewShortURLHandler(db, cfg.ShortURL.Password)
	mockAPIHandler := handlers.NewMockAPIHandler(db)
	mdShareHandler := handlers.NewMDShareHandler(db, cfg.MDShare.AdminPassword, cfg.MDShare.DefaultMaxViews, cfg.MDShare.DefaultExpiresDays)
	excalidrawHandler := handlers.NewExcalidrawHandler(db, cfg.Excalidraw.AdminPassword, cfg.Excalidraw.DefaultExpiresDays, cfg.Excalidraw.MaxContentSize)
	pregnancyHandler := handlers.NewPregnancyHandler(db, cfg.Pregnancy.DefaultExpiresDays, cfg.Pregnancy.MaxDataSize)
	expenseHandler := handlers.NewExpenseHandler(db, cfg)
	glucoseHandler := handlers.NewGlucoseHandler(db, cfg)
	plannerHandler = handlers.NewPlannerHandler(db, cfg)
	recipeHandler := handlers.NewRecipeHandler(db, 365, 1024*1024)
	householdHandler := handlers.NewHouseholdHandler(db, cfg)
	photoWallHandler := handlers.NewPhotoWallHandler(db, cfg)
	consoleHandler := handlers.NewConsoleHandler(db, cfg.Console.AdminPassword)

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
	imageUnderstandingHandler := handlers.NewImageUnderstandingHandler(cfg, transientStore)
	apiGatewayHandler := handlers.NewAPIGatewayHandler(aiGatewayHandler, imageUnderstandingHandler)
	autoDevHandler := handlers.NewAutoDevHandler(db, cfg.AutoDev.AdminPassword, cfg.AutoDev.AutodevPath, cfg.AutoDev.DataDir)
	mermaidHandler := handlers.NewMermaidHandler(db, cfg)
	npsHandler := handlers.NewNPSHandler(cfg.NPS, cfg.Proxy.TunnelPort)
	proxyHandler := handlers.NewProxyHandler(cfg, npsHandler)
	hermesHandler := handlers.NewHermesHandler(cfg.Hermes)
	handlers.InitGFWList("./data/proxy.db")
	proxyHandler.AutoSelectOnStartup()
	proxyHandler.StartAutoMaintenance()
	plannerHandler.ProcessDueReminders()

	// Edge TTS 处理器
	edgeTTSHandler := handlers.NewEdgeTTSHandler(cfg.Chat.TTSServiceURL)

	// 游戏处理器
	gameHandler := handlers.NewGameHandler(cfg.MiniMax)

	// 启动 SSH 会话清理协程
	terminalHandler.StartCleanupRoutine()

	// 启动定期清理协程（每小时清理过期数据）
	startCleanupRoutine(db, plannerHandler, cfg)

// API 路由
	api := r.Group("/api")
	h := &routeHandlers{
		pasteHandler:              pasteHandler,
		dnsHandler:                dnsHandler,
		chatHandler:               chatHandler,
		shortURLHandler:           shortURLHandler,
		mockAPIHandler:            mockAPIHandler,
		mdShareHandler:            mdShareHandler,
		excalidrawHandler:         excalidrawHandler,
		pregnancyHandler:          pregnancyHandler,
		expenseHandler:            expenseHandler,
		glucoseHandler:            glucoseHandler,
		plannerHandler:            plannerHandler,
		recipeHandler:             recipeHandler,
		householdHandler:          householdHandler,
		photoWallHandler:          photoWallHandler,
		consoleHandler:            consoleHandler,
		terminalHandler:           terminalHandler,
		nfsShareHandler:           nfsShareHandler,
		ocrHandler:                ocrHandler,
		bailianHandler:            bailianHandler,
		aiGatewayHandler:          aiGatewayHandler,
		imageUnderstandingHandler: imageUnderstandingHandler,
		apiGatewayHandler:         apiGatewayHandler,
		autoDevHandler:            autoDevHandler,
		mermaidHandler:            mermaidHandler,
		npsHandler:                npsHandler,
		proxyHandler:              proxyHandler,
		hermesHandler:             hermesHandler,
		edgeTTSHandler:            edgeTTSHandler,
		gameHandler:               gameHandler,
	}
	setupRoutes(api, createRateLimiter, h)

	// 短链重定向（非 API 路径）
	r.GET("/s/:id", shortURLHandler.Redirect)

	// Mock API 执行（非 API 路径，支持所有 HTTP 方法）
	r.Any("/mock/:id", mockAPIHandler.Execute)

	// 静态文件（生产环境）
	distDir := "./dist"
	r.Static("/assets", distDir+"/assets")
	r.Static("/neon", distDir+"/neon")
	r.StaticFile("/", distDir+"/index.html")
	r.StaticFile("/alipay.jpeg", distDir+"/alipay.jpeg")
	r.StaticFile("/wxpay.jpeg", distDir+"/wxpay.jpeg")
	r.GET("/manifest.webmanifest", func(c *gin.Context) {
		c.Header("Content-Type", "application/manifest+json; charset=utf-8")
		c.File(distDir + "/manifest.webmanifest")
	})
	r.GET("/sw.js", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File(distDir + "/sw.js")
	})
	r.StaticFile("/pregnancy-shortcut-192.png", distDir+"/pregnancy-shortcut-192.png")
	r.StaticFile("/pregnancy-shortcut-512.png", distDir+"/pregnancy-shortcut-512.png")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/api" || strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"error": "接口不存在", "path": path})
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

	log.Printf("服务器启动在端口 %s", port)

	// 启动独立代理端口（供 NPS 映射，绕过 nginx）
	if cfg.Proxy.TunnelPort != "" {
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

	srv := &http.Server{
		Addr: ":" + port,
		// 在 Handler 层拦截 CONNECT 方法，其余交给 Gin
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodConnect {
				proxyHandler.Tunnel(w, req)
				return
			}
			r.ServeHTTP(w, req)
		}),
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
