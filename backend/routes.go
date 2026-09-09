package main

import (
	"devtools/handlers"
	"devtools/middleware"
	"devtools/routes"

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
	monitoringHandler         *handlers.MonitoringHandler
	terminalHandler           *handlers.SSHHandler
	nfsShareHandler          *handlers.NFSShareHandler
	ocrHandler               *handlers.OCRHandler
	aiGatewayHandler         *handlers.AIGatewayHandler
	imageUnderstandingHandler *handlers.ImageUnderstandingHandler
	cpaProxyHandler          *handlers.CPAProxyHandler
	autoDevHandler           *handlers.AutoDevHandler
	mermaidHandler           *handlers.MermaidHandler
	npsHandler               *handlers.NPSHandler
	proxyHandler             *handlers.ProxyHandler
	hermesHandler            *handlers.HermesHandler
	edgeTTSHandler           *handlers.EdgeTTSHandler
	voiceMemoHandler         *handlers.VoiceMemoHandler
	gameHandler              *handlers.GameHandler
	askitSyncHandler         *handlers.AskitSyncHandler
	screenHandler            *handlers.ScreenHandler
	skillsHandler            *handlers.SkillsHandler
	skillsGuard              *middleware.SkillsGuard
}

func setupRoutes(api *gin.RouterGroup, createRateLimiter *middleware.RateLimiter, h *routeHandlers) {
	// Convert main routeHandlers to routes.routeHandlers (same field types)
	rh := &routes.RouteHandlers{
		PasteHandler:              h.pasteHandler,
		DnsHandler:                h.dnsHandler,
		ChatHandler:               h.chatHandler,
		ShortURLHandler:           h.shortURLHandler,
		MockAPIHandler:            h.mockAPIHandler,
		MdShareHandler:            h.mdShareHandler,
		ExcalidrawHandler:         h.excalidrawHandler,
		PregnancyHandler:          h.pregnancyHandler,
		ExpenseHandler:            h.expenseHandler,
		GlucoseHandler:            h.glucoseHandler,
		PlannerHandler:            h.plannerHandler,
		RecipeHandler:             h.recipeHandler,
		HouseholdHandler:          h.householdHandler,
		PhotoWallHandler:          h.photoWallHandler,
		ConsoleHandler:            h.consoleHandler,
		MonitoringHandler:         h.monitoringHandler,
		TerminalHandler:           h.terminalHandler,
		NfsShareHandler:          h.nfsShareHandler,
		OcrHandler:               h.ocrHandler,
		AiGatewayHandler:         h.aiGatewayHandler,
		ImageUnderstandingHandler: h.imageUnderstandingHandler,
		CpaProxyHandler:          h.cpaProxyHandler,
		AutoDevHandler:           h.autoDevHandler,
		MermaidHandler:           h.mermaidHandler,
		NpsHandler:               h.npsHandler,
		ProxyHandler:             h.proxyHandler,
		HermesHandler:            h.hermesHandler,
		EdgeTTSHandler:           h.edgeTTSHandler,
		VoiceMemoHandler:         h.voiceMemoHandler,
		GameHandler:              h.gameHandler,
		AskitSyncHandler:         h.askitSyncHandler,
		ScreenHandler:            h.screenHandler,
		SkillsHandler:            h.skillsHandler,
		SkillsGuard:              h.skillsGuard,
	}
	routes.RegisterAllRoutes(api, rh, createRateLimiter)
}
