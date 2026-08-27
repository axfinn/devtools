package routes

import (
	"github.com/gin-gonic/gin"
)

func RegisterHealthRoute(api *gin.RouterGroup) {
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
