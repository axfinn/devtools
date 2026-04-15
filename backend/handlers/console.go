package handlers

import (
	"database/sql"
	"encoding/json"

	"github.com/gin-gonic/gin"

	"devtools/models"
)

type ConsoleHandler struct {
	db       *models.DB
	password string
}

func NewConsoleHandler(db *models.DB, password string) *ConsoleHandler {
	return &ConsoleHandler{db: db, password: password}
}

func (h *ConsoleHandler) checkAdmin(password string) bool {
	return h.password != "" && password == h.password
}

// GetSettings GET /api/console/settings  — 无需密码，公开读
func (h *ConsoleHandler) GetSettings(c *gin.Context) {
	val, err := h.db.GetSetting("hidden_routes")
	if err == sql.ErrNoRows {
		c.JSON(200, gin.H{"hidden_routes": []string{}})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var routes []string
	if err := json.Unmarshal([]byte(val), &routes); err != nil {
		routes = []string{}
	}
	c.JSON(200, gin.H{"hidden_routes": routes})
}

// VerifyPassword POST /api/console/verify?admin_password=xxx
func (h *ConsoleHandler) VerifyPassword(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(401, gin.H{"error": "密码错误"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// SaveSettings POST /api/console/settings?admin_password=xxx
func (h *ConsoleHandler) SaveSettings(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(401, gin.H{"error": "密码错误"})
		return
	}
	var req struct {
		HiddenRoutes []string `json:"hidden_routes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	b, _ := json.Marshal(req.HiddenRoutes)
	if err := h.db.SetSetting("hidden_routes", string(b)); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
