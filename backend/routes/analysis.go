package routes

import (
	"devtools/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterAnalysisRoutes(api *gin.RouterGroup, h *RouteHandlers) {
	analysisHandler := handlers.NewAnalysisHandler()
	analysis := api.Group("/analysis")
	{
		analysis.POST("/code", analysisHandler.AnalyzeCode)
		analysis.POST("/scan", analysisHandler.ScanContent)
		analysis.POST("/validate", analysisHandler.ValidateFile)
		analysis.GET("/languages", analysisHandler.GetSupportedLanguages)
	}
}
