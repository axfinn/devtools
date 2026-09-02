package routes

import "github.com/gin-gonic/gin"

func RegisterProxyRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	proxyGroup := api.Group("/proxy")
	{
		proxyGroup.POST("/verify", h.ProxyHandler.VerifyPassword)
		proxyGroup.POST("/config", h.ProxyHandler.LoadConfig)
		proxyGroup.POST("/speedtest", h.ProxyHandler.SpeedTest)
		proxyGroup.POST("/start", h.ProxyHandler.Start)
		proxyGroup.POST("/auto-start", h.ProxyHandler.AutoStart)
		proxyGroup.POST("/stop", h.ProxyHandler.Stop)
		proxyGroup.POST("/check", h.ProxyHandler.CheckNodes)
		proxyGroup.POST("/subscription-refresh", h.ProxyHandler.TriggerSubscriptionRefresh)
		proxyGroup.POST("/nps-tunnel", h.ProxyHandler.CreateNPSTunnel)
		proxyGroup.GET("/status", h.ProxyHandler.Status)
		proxyGroup.GET("/fetch", h.ProxyHandler.Fetch)
		proxyGroup.GET("/resource", h.ProxyHandler.Resource)
		proxyGroup.GET("/subscription", h.ProxyHandler.DownloadSubscription)
		proxyGroup.GET("/subscription/:type", h.ProxyHandler.DownloadSubscription)
		proxyGroup.GET("/extension", h.ProxyHandler.DownloadExtension)
		proxyGroup.GET("/ws-tunnel", h.ProxyHandler.WsTunnel)
		proxyGroup.GET("/client/download", h.ProxyHandler.DownloadClient)
		proxyGroup.GET("/custom-domains", h.ProxyHandler.ListCustomDomains)
		proxyGroup.POST("/custom-domains", h.ProxyHandler.AddCustomDomain)
		proxyGroup.DELETE("/custom-domains", h.ProxyHandler.RemoveCustomDomain)
	}
}
