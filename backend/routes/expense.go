package routes

import (
	"devtools/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterExpenseRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	expense := api.Group("/expense")
	{
		expense.POST("", createRateLimiter.Middleware(), h.ExpenseHandler.Create)
		expense.POST("/login", h.ExpenseHandler.Login)
		expense.GET("/:id", h.ExpenseHandler.Get)
		expense.DELETE("/:id", h.ExpenseHandler.Delete)
		expense.PUT("/:id/extend", h.ExpenseHandler.Extend)
		expense.GET("/:id/accounts", h.ExpenseHandler.GetAccounts)
		expense.POST("/:id/accounts", h.ExpenseHandler.CreateAccount)
		expense.PUT("/:id/accounts/:accountId", h.ExpenseHandler.UpdateAccount)
		expense.DELETE("/:id/accounts/:accountId", h.ExpenseHandler.DeleteAccount)
		expense.GET("/:id/categories", h.ExpenseHandler.GetCategories)
		expense.POST("/:id/categories", h.ExpenseHandler.CreateCategory)
		expense.PUT("/:id/categories/:categoryId", h.ExpenseHandler.UpdateCategory)
		expense.DELETE("/:id/categories/:categoryId", h.ExpenseHandler.DeleteCategory)
		expense.GET("/:id/transactions", h.ExpenseHandler.GetTransactions)
		expense.POST("/:id/transactions", h.ExpenseHandler.CreateTransaction)
		expense.PUT("/:id/transactions/:txId", h.ExpenseHandler.UpdateTransaction)
		expense.DELETE("/:id/transactions/:txId", h.ExpenseHandler.DeleteTransaction)
		expense.GET("/:id/stats", h.ExpenseHandler.GetStats)
		expense.POST("/:id/analyze", h.ExpenseHandler.Analyze)
		expense.GET("/:id/analyze/:jobId", h.ExpenseHandler.GetAnalyzeJob)
		expense.POST("/:id/voice-parse", h.ExpenseHandler.VoiceParse)
	}
}
