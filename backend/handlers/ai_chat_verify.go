package handlers

import (
	"github.com/gin-gonic/gin"
)

// AIChatVerify POST /api/ai-chat/verify
func (h *AIGatewayHandler) AIChatVerify(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	expected := h.cfg.Console.AdminPassword
	if expected == "" || req.Password != expected {
		c.JSON(401, gin.H{"error": "密码错误"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
