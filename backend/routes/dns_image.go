package routes

import (
	"devtools/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterDNSAndImageProxyRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	api.GET("/ip", h.DnsHandler.GetIP)
	api.GET("/dns", h.DnsHandler.Lookup)
	api.GET("/proxy-image", handlers.ProxyImage)
}
