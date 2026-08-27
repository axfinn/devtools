package routes

import "github.com/gin-gonic/gin"

func RegisterAPIGatewayRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	apigw := api.Group("/api-gateway")
	{
		apigw.Any("/cpa/v1", h.CpaProxyHandler.ProxyCPA)
		apigw.Any("/cpa/v1/*proxyPath", h.CpaProxyHandler.ProxyCPA)
	}
}
