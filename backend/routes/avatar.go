package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAvatarRoutes 注册虚拟形象模块路由
//
// 路由分组:
//   /api/avatar/models            —— 模型库 CRUD
//   /api/avatar/me/assets         —— 我的资产
//   /api/avatar/clips             —— 姿态片段 CRUD
//
// POST 端点接 createRateLimiter 中间件(10/min);GET / DELETE 不限流。
// P0 不实现完整 §1.2 的 12 端点 —— 分享链接 / 完整筛选留到 P1。
func RegisterAvatarRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	avatar := api.Group("/avatar")
	{
		models := avatar.Group("/models")
		{
			models.POST("", createRateLimiter.Middleware(), h.AvatarHandler.UploadModel)
			models.GET("", h.AvatarHandler.ListModels)
			models.GET("/:id", h.AvatarHandler.GetModel)
			models.GET("/:id/file", h.AvatarHandler.ServeModelFile)
			models.DELETE("/:id", h.AvatarHandler.DeleteModel)
		}

		me := avatar.Group("/me")
		{
			meAssets := me.Group("/assets")
			{
				meAssets.POST("", createRateLimiter.Middleware(), h.AvatarHandler.RegisterMeAsset)
				meAssets.GET("", h.AvatarHandler.ListMeAssets)
				meAssets.DELETE("/:id", h.AvatarHandler.DeleteMeAsset)
			}
		}

		clips := avatar.Group("/clips")
		{
			clips.POST("", createRateLimiter.Middleware(), h.AvatarHandler.UploadClip)
			clips.GET("/:id", h.AvatarHandler.GetClip)
			clips.DELETE("/:id", h.AvatarHandler.DeleteClip)
		}
	}
}