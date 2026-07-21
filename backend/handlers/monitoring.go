package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

type MonitoringHandler struct {
	db        *models.DB
	cfg       *config.Config
	startedAt time.Time
}

func NewMonitoringHandler(db *models.DB, cfg *config.Config) *MonitoringHandler {
	return &MonitoringHandler{db: db, cfg: cfg, startedAt: time.Now()}
}

func (h *MonitoringHandler) Verify(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *MonitoringHandler) Overview(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	rangeName, since := monitoringRange(c.Query("range"))
	overview, err := h.db.GetMonitoringOverview(since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	endpoints, err := h.db.GetMonitoringTopEndpoints(since, monitorInt(c.Query("limit"), 12, 1, 100))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"range":     rangeName,
		"since":     since,
		"overview":  overview,
		"endpoints": endpoints,
	})
}

func (h *MonitoringHandler) Responses(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	rangeName, since := monitoringRange(c.Query("range"))
	overview, err := h.db.GetMonitoringOverview(since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	endpoints, err := h.db.GetMonitoringTopEndpoints(since, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	errors := make([]models.MonitoringEndpointStat, 0)
	for _, endpoint := range endpoints {
		if endpoint.Errors > 0 {
			errors = append(errors, endpoint)
		}
		if len(errors) == 20 {
			break
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"range": rangeName,
		"status": gin.H{
			"2xx": overview.Status2xx,
			"3xx": overview.Status3xx,
			"4xx": overview.Status4xx,
			"5xx": overview.Status5xx,
		},
		"average_latency_ms": overview.AverageMS,
		"max_latency_ms":     overview.MaxLatencyMS,
		"request_bytes":      overview.RequestBytes,
		"response_bytes":     overview.ResponseBytes,
		"error_endpoints":    errors,
	})
}

func (h *MonitoringHandler) Logs(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	_, since := monitoringRange(c.Query("range"))
	limit := monitorInt(c.Query("limit"), 50, 1, 200)
	offset := monitorInt(c.Query("offset"), 0, 0, 1000000)
	logs, total, err := h.db.ListHTTPRequestLogs(since, c.Query("status"), c.Query("keyword"), limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total, "limit": limit, "offset": offset})
}

func (h *MonitoringHandler) AI(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	_, since := monitoringRange(c.Query("range"))
	limit := monitorInt(c.Query("limit"), 50, 1, 200)
	offset := monitorInt(c.Query("offset"), 0, 0, 1000000)
	summary, err := h.db.GetAIMonitoringSummary(since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logs, err := h.db.ListAIAPIRequestLogs(c.Query("api_key_id"), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary, "logs": logs, "limit": limit, "offset": offset})
}

func (h *MonitoringHandler) Service(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	stats, err := h.db.GetMonitoringStorageStats()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "database": "error", "error": err.Error()})
		return
	}
	sessions, sessErr := h.db.GetProxySessionSummary(time.Now().Add(-24 * time.Hour))
	if sessErr != nil {
		sessions = &models.ProxySessionSummary{}
	}
	c.JSON(http.StatusOK, gin.H{
		"status":                 "ok",
		"database":               "ok",
		"uptime_seconds":         int64(time.Since(h.startedAt).Seconds()),
		"server_time":            time.Now().UTC(),
		"monitoring_enabled":     h.cfg.Monitoring.Enabled,
		"detail_retention_days":  h.cfg.Monitoring.DetailRetentionDays,
		"archive_retention_days": h.cfg.Monitoring.ArchiveRetentionDays,
		"storage":                stats,
		"sessions":               sessions,
	})
}

func (h *MonitoringHandler) Sessions(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	limit := monitorInt(c.Query("limit"), 50, 1, 200)
	offset := monitorInt(c.Query("offset"), 0, 0, 1000000)
	logs, total, err := h.db.ListProxySessionLogs(c.Query("type"), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	since := time.Now().Add(-24 * time.Hour)
	if r := c.Query("range"); r != "" {
		if _, since2 := monitoringRange(r); true {
			since = since2
		}
	}
	summary, err := h.db.GetProxySessionSummary(since)
	if err != nil {
		summary = &models.ProxySessionSummary{}
	}
	c.JSON(http.StatusOK, gin.H{
		"summary": summary,
		"logs":    logs,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *MonitoringHandler) Archive(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	result, err := h.db.ArchiveHTTPRequestLogs(time.Now().UTC(), h.cfg.Monitoring.DetailRetentionDays, h.cfg.Monitoring.ArchiveRetentionDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "result": result})
}

func (h *MonitoringHandler) DeleteLogs(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	var request struct {
		Before  string `json:"before"`
		Scope   string `json:"scope"`
		Confirm bool   `json:"confirm"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}
	if !request.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除前必须明确确认"})
		return
	}
	before := time.Now().UTC()
	if strings.TrimSpace(request.Before) != "" {
		parsed, err := time.Parse(time.RFC3339, request.Before)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "before 必须是 RFC3339 时间"})
			return
		}
		before = parsed
	}
	archiveResult, err := h.db.ArchiveHTTPRequestLogs(before, 0, h.cfg.Monitoring.ArchiveRetentionDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted, err := h.db.DeleteHTTPRequestDetails(before, request.Scope)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "archived": archiveResult.ArchivedDetails, "deleted": deleted})
}

func (h *MonitoringHandler) requireAdmin(c *gin.Context) bool {
	configured := strings.TrimSpace(h.cfg.Monitoring.AdminPassword)
	if configured == "" {
		configured = strings.TrimSpace(h.cfg.AIGateway.SuperAdminPassword)
	}
	if configured == "" {
		configured = strings.TrimSpace(h.cfg.Console.AdminPassword)
	}
	if configured == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "未配置 monitoring.admin_password、ai_gateway.super_admin_password 或 console.admin_password"})
		return false
	}
	password := c.GetHeader("X-Super-Admin-Password")
	if password == "" {
		password = c.Query("super_admin_password")
	}
	if password != configured {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误"})
		return false
	}
	return true
}

func monitoringRange(value string) (string, time.Time) {
	now := time.Now().UTC()
	switch value {
	case "1h":
		return value, now.Add(-time.Hour)
	case "7d":
		return value, now.AddDate(0, 0, -7)
	case "30d":
		return value, now.AddDate(0, 0, -30)
	case "90d":
		return value, now.AddDate(0, 0, -90)
	case "24h", "":
		return "24h", now.Add(-24 * time.Hour)
	default:
		return "24h", now.Add(-24 * time.Hour)
	}
}

func monitorInt(value string, fallback, min, max int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || parsed > max {
		return fallback
	}
	return parsed
}
