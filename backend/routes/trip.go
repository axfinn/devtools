package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterTripRoutes 挂载 /api/trips/* 全部端点。
// 与现有 36 个模块同档:写端点复用全局 createRateLimiter(10/min/IP),
// 中间件链(ContentSizeLimiter + requestLogger + cors)由父 router 统一加。
func RegisterTripRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	if h.TripHandler == nil {
		return
	}
	t := api.Group("/trips")
	{
		t.GET("", h.TripHandler.List)
		t.POST("", createRateLimiter.Middleware(), h.TripHandler.Create)
		t.POST("/seed", createRateLimiter.Middleware(), h.TripHandler.Seed)
		t.POST("/process-now", createRateLimiter.Middleware(), h.TripHandler.ProcessNow)

		t.GET("/:id", h.TripHandler.Get)
		t.GET("/:id/markdown", h.TripHandler.Markdown)
		t.GET("/:id/summary", h.TripHandler.Summary)
		t.GET("/:id/summary/markdown", h.TripHandler.SummaryMarkdown)
		t.DELETE("/:id", createRateLimiter.Middleware(), h.TripHandler.Delete)

		// 预算(只动 budget 字段,不重置 name / dates)
		t.PUT("/:id/budget", createRateLimiter.Middleware(), h.TripHandler.UpdateBudget)

		// 费用账本
		t.GET("/:id/expenses", h.TripHandler.ListExpenses)
		t.POST("/:id/expenses", createRateLimiter.Middleware(), h.TripHandler.AddExpense)
		t.PUT("/expenses/:expenseId", createRateLimiter.Middleware(), h.TripHandler.UpdateExpense)
		t.DELETE("/expenses/:expenseId", createRateLimiter.Middleware(), h.TripHandler.DeleteExpense)

		t.POST("/:id/days", createRateLimiter.Middleware(), h.TripHandler.AddDay)
		t.PATCH("/days/:dayId", createRateLimiter.Middleware(), h.TripHandler.UpdateDay)
		// 给 DayPlan 加活动:用 /trips/days/:dayId/activities 路径,
		// 避免 /trips/:id/days/:dayId/activities 这种容易混淆的活动作用域。
		t.POST("/days/:dayId/activities", createRateLimiter.Middleware(), h.TripHandler.AddActivity)
		// 单独改提醒配置 + 支持 reset 重发。
		t.PUT("/activities/:activityId/reminder", createRateLimiter.Middleware(), h.TripHandler.SetReminder)
		// 出行中 inline edit:改 title / time / location / duration / note
		t.PATCH("/activities/:activityId", createRateLimiter.Middleware(), h.TripHandler.UpdateActivity)
	}
}