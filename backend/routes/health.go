package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterHealthRoute 注册 /api/health。
// 默认形态（无参数）仍为 200 {"status":"ok"}，deploy.sh 与 docker-compose 的
// healthcheck 依赖此行为；详细形态 ?detail=1 走 HealthHandler 的依赖探测 + 鉴权。
func RegisterHealthRoute(api *gin.RouterGroup, h *RouteHandlers) {
	api.GET("/health", h.HealthHandler.Handle)
}
