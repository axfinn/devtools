package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"devtools/config"
	"devtools/handlers"
	"devtools/middleware"
	"devtools/utils"
)

func buildRouteHandlers(rt *appRuntime) (*routeHandlers, error) {
	cfg := rt.cfg
	db := rt.db

	pasteHandler := handlers.NewPasteHandler(db, cfg, rt.transientStore)
	dnsHandler := handlers.NewDNSHandler()
	chatHandler := handlers.NewChatHandler(db, cfg.Chat.AdminPassword, cfg.MiniMax, cfg.Chat.TTSServiceURL)
	shortURLHandler := handlers.NewShortURLHandler(db, cfg.ShortURL.Password)
	mockAPIHandler := handlers.NewMockAPIHandler(db)
	mdShareHandler := handlers.NewMDShareHandler(db, cfg.MDShare.AdminPassword, cfg.MDShare.DefaultMaxViews, cfg.MDShare.DefaultExpiresDays)
	excalidrawHandler := handlers.NewExcalidrawHandler(db, cfg.Excalidraw.AdminPassword, cfg.Excalidraw.DefaultExpiresDays, cfg.Excalidraw.MaxContentSize)
	pregnancyHandler := handlers.NewPregnancyHandler(db, cfg.Pregnancy.DefaultExpiresDays, cfg.Pregnancy.MaxDataSize)
	expenseHandler := handlers.NewExpenseHandler(db, cfg)
	glucoseHandler := handlers.NewGlucoseHandler(db, cfg)
	plannerHandler := handlers.NewPlannerHandler(db, cfg)
	recipeHandler := handlers.NewRecipeHandler(db, 365, 1024*1024)
	householdHandler := handlers.NewHouseholdHandler(db, cfg)
	photoWallHandler := handlers.NewPhotoWallHandler(db, cfg)
	consoleHandler := handlers.NewConsoleHandler(db, cfg.Console.AdminPassword)
	monitoringHandler := handlers.NewMonitoringHandler(db, cfg)

	encryptionService, err := newEncryptionService(cfg)
	if err != nil {
		return nil, err
	}

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
	imageUnderstandingHandler := handlers.NewImageUnderstandingHandler(cfg, rt.transientStore)
	aiGatewayHandler := handlers.NewAIGatewayHandler(db, cfg, bailianHandler, imageUnderstandingHandler)
	cpaProxyHandler := handlers.NewCPAProxyHandler()
	autoDevHandler := handlers.NewAutoDevHandler(db, cfg.AutoDev.AdminPassword, cfg.AutoDev.AutodevPath, cfg.AutoDev.DataDir)
	mermaidHandler := handlers.NewMermaidHandler(db, cfg)
	npsHandler := handlers.NewNPSHandler(cfg.NPS, cfg.Proxy.TunnelPort)
	proxyHandler := handlers.NewProxyHandler(db, cfg, npsHandler)
	hermesHandler := handlers.NewHermesHandler(cfg.Hermes)
	edgeTTSHandler := handlers.NewEdgeTTSHandler(cfg.Chat.TTSServiceURL)
	gameHandler := handlers.NewGameHandler(db)
	voiceMemoHandler := handlers.NewVoiceMemoHandler(db, plannerHandler, envOrDefault("ASR_SERVICE_URL", "http://asr-service:9000"), os.Getenv("DIARIZE_SERVICE_URL"))
	askitSyncHandler := handlers.NewAskitSyncHandler(db, cfg.AskitSync)
	screenHandler := handlers.NewScreenHandler(db, *cfg)

	// Skills 模块:对外工具入口,默认未启用(cfg.Skills.Enabled = false),
	// 由管理员在 config.yaml 显式打开后才对外暴露。
	// Guard 在 db 初始化后单独构造,AttachGuard 给 SkillsHandler 注入写库专项限流。
	skillsHandler := handlers.NewSkillsHandler(db)
	skillsGuard := middleware.NewSkillsGuard(cfg.Skills, rt.transientStore)
	skillsHandler.AttachGuard(skillsGuard)
	// paste_create 复用 PasteHandler 全套逻辑(title/language/password/expires_in/max_views/admin_password)
	skillsHandler.AttachPasteHandler(pasteHandler)

	return &routeHandlers{
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
		monitoringHandler:         monitoringHandler,
		terminalHandler:           terminalHandler,
		nfsShareHandler:           nfsShareHandler,
		ocrHandler:                ocrHandler,
		bailianHandler:            bailianHandler,
		aiGatewayHandler:          aiGatewayHandler,
		imageUnderstandingHandler: imageUnderstandingHandler,
		cpaProxyHandler:           cpaProxyHandler,
		autoDevHandler:            autoDevHandler,
		mermaidHandler:            mermaidHandler,
		npsHandler:                npsHandler,
		proxyHandler:              proxyHandler,
		hermesHandler:             hermesHandler,
		edgeTTSHandler:            edgeTTSHandler,
		gameHandler:               gameHandler,
		voiceMemoHandler:          voiceMemoHandler,
		askitSyncHandler:          askitSyncHandler,
		screenHandler:             screenHandler,
		skillsHandler:             skillsHandler,
		skillsGuard:               skillsGuard,
	}, nil
}

func newEncryptionService(cfg *config.Config) (*utils.EncryptionService, error) {
	encryptionKey := cfg.SSH.EncryptionKey
	if encryptionKey == "" {
		encryptionKey = os.Getenv("TERMINAL_ENCRYPTION_KEY")
	}
	if encryptionKey == "" {
		randomKey, _ := utils.GenerateRandomKey()
		log.Printf("WARNING: 未设置加密密钥，使用临时随机密钥，重启后之前保存的 SSH 密码将无法解密")
		log.Printf("WARNING: 请在配置文件中设置 ssh.encryption_key 或设置环境变量 TERMINAL_ENCRYPTION_KEY")
		encryptionKey = randomKey
	}
	encryptionService, err := utils.NewEncryptionService(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("创建加密服务失败: %w", err)
	}
	return encryptionService, nil
}
