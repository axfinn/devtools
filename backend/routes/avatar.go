package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAvatarRoutes 注册虚拟形象模块路由
//
// 路由分组(P1 完整 §1.2 + 分享):
//   /api/avatar/models            —— 模型库 CRUD(5)
//   /api/avatar/me/assets         —— 我的资产(3)
//   /api/avatar/clips             —— 姿态片段 CRUD(3)
//   /api/avatar/share             —— 分享短链(3)
//
// POST 端点接 createRateLimiter 中间件(10/min);GET / DELETE 不限流。
// me/assets 端点要求 X-Creator-Key(R6/R7);share 端点创建可匿名(匿名 share 不可删)。
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

		share := avatar.Group("/share")
		{
			share.POST("", createRateLimiter.Middleware(), h.AvatarHandler.CreateShare)
			share.GET("/:code", h.AvatarHandler.GetShare)
			share.DELETE("/:code", h.AvatarHandler.DeleteShare)
		}
	}
}