package routes

import (
	"devtools/handlers"
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// routeHandlers mirrors the main package routeHandlers struct.
// We re-declare it here to avoid import cycles (routes cannot import main).
type RouteHandlers struct {
	PasteHandler              *handlers.PasteHandler
	DnsHandler                *handlers.DNSHandler
	ChatHandler               *handlers.ChatHandler
	ShortURLHandler           *handlers.ShortURLHandler
	MockAPIHandler            *handlers.MockAPIHandler
	MdShareHandler            *handlers.MDShareHandler
	ExcalidrawHandler         *handlers.ExcalidrawHandler
	PregnancyHandler          *handlers.PregnancyHandler
	ExpenseHandler            *handlers.ExpenseHandler
	GlucoseHandler            *handlers.GlucoseHandler
	PlannerHandler            *handlers.PlannerHandler
	RecipeHandler             *handlers.RecipeHandler
	HouseholdHandler          *handlers.HouseholdHandler
	PhotoWallHandler          *handlers.PhotoWallHandler
	ConsoleHandler            *handlers.ConsoleHandler
	MonitoringHandler         *handlers.MonitoringHandler
	TerminalHandler           *handlers.SSHHandler
	NfsShareHandler           *handlers.NFSShareHandler
	OcrHandler                *handlers.OCRHandler
	AiGatewayHandler          *handlers.AIGatewayHandler
	ImageUnderstandingHandler *handlers.ImageUnderstandingHandler
	CpaProxyHandler           *handlers.CPAProxyHandler
	AutoDevHandler            *handlers.AutoDevHandler
	MermaidHandler            *handlers.MermaidHandler
	NpsHandler                *handlers.NPSHandler
	ProxyHandler              *handlers.ProxyHandler
	HermesHandler             *handlers.HermesHandler
	EdgeTTSHandler            *handlers.EdgeTTSHandler
	VoiceMemoHandler          *handlers.VoiceMemoHandler
	GameHandler               *handlers.GameHandler
	AskitSyncHandler          *handlers.AskitSyncHandler
	ScreenHandler             *handlers.ScreenHandler
	SkillsHandler             *handlers.SkillsHandler
	SkillsGuard               *middleware.SkillsGuard
}

// RegisterAllRoutes wires up all domain route groups.
func RegisterAllRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	RegisterPasteRoutes(api, h, createRateLimiter)
	RegisterAnalysisRoutes(api, h)
	RegisterDNSAndImageProxyRoutes(api, h)
	RegisterChatRoutes(api, h, createRateLimiter)
	RegisterEdgeTTSRoutes(api, h)
	RegisterGameRoutes(api, h)
	RegisterVoiceMemoRoutes(api, h)
	RegisterShortURLRoutes(api, h, createRateLimiter)
	RegisterMockAPIRoutes(api, h, createRateLimiter)
	RegisterMDShareRoutes(api, h, createRateLimiter)
	RegisterExcalidrawRoutes(api, h, createRateLimiter)
	RegisterPregnancyRoutes(api, h, createRateLimiter)
	RegisterExpenseRoutes(api, h, createRateLimiter)
	RegisterGlucoseRoutes(api, h, createRateLimiter)
	RegisterPlannerRoutes(api, h, createRateLimiter)
	RegisterRecipeRoutes(api, h, createRateLimiter)
	RegisterHouseholdRoutes(api, h, createRateLimiter)
	RegisterPhotoWallRoutes(api, h, createRateLimiter)
	RegisterTerminalRoutes(api, h, createRateLimiter)
	RegisterNFSShareRoutes(api, h)
	RegisterImageUnderstandingRoutes(api, h, createRateLimiter)
	RegisterAIGatewayRoutes(api, h, createRateLimiter)
	RegisterAskitRoutes(api, h, createRateLimiter)
	RegisterScreenRoutes(api, h)
	RegisterAPIGatewayRoutes(api, h)
	RegisterAutoDevRoutes(api, h, createRateLimiter)
	RegisterMermaidRoutes(api, h)
	RegisterProxyRoutes(api, h)
	RegisterNPSRoutes(api, h)
	RegisterHermesRoutes(api, h)
	RegisterBackgroundRoutes(api, h)
	RegisterMonitorRoutes(api, h)
	RegisterConsoleRoutes(api, h)
	RegisterSkillsRoutes(api, h, h.SkillsGuard)
	RegisterOCRRoutes(api, h, createRateLimiter)
	RegisterHealthRoute(api)
}
