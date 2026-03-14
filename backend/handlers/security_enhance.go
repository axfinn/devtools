package handlers

import (
	"devtools/utils/security"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityEnhanceHandler 安全增强处理器
type SecurityEnhanceHandler struct {
	enhancer *security.SecurityEnhancer
	detector *security.SensitiveInfoDetector
	logger   *security.AuditLogger
}

// NewSecurityEnhanceHandler 创建安全增强处理器
func NewSecurityEnhanceHandler(logDir string) *SecurityEnhanceHandler {
	return &SecurityEnhanceHandler{
		enhancer: security.NewSecurityEnhancer(),
		detector: security.NewSensitiveInfoDetector(),
		logger:   security.NewAuditLogger(logDir),
	}
}

// EnhanceRequest 安全增强请求
type EnhanceRequest struct {
	Content string `json:"content" binding:"required"`
}

// EnhanceResponse 安全增强响应
type EnhanceResponse struct {
	IsSafe          bool                   `json:"is_safe"`
	HasSensitive    bool                   `json:"has_sensitive"`
	SensitiveResult *security.DetectionResult `json:"sensitive_result,omitempty"`
	ContentHash     string                 `json:"content_hash"`
	Suggestions     []string               `json:"suggestions"`
	Warnings        []string               `json:"warnings"`
}

// EnhanceSecurity 执行安全增强
func (h *SecurityEnhanceHandler) EnhanceSecurity(c *gin.Context) {
	var req EnhanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 内容限制（1MB）
	if len(req.Content) > 1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过 1MB 限制", "code": 400})
		return
	}

	result := h.enhancer.EnhanceSecurity(req.Content)

	// 记录审计日志
	sourceIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	if result.HasSensitive {
		var items []string
		for _, item := range result.SensitiveResult.FoundItems {
			items = append(items, item.Type)
		}
		h.logger.LogSensitive(sourceIP, userAgent, "", items, result.SensitiveResult.Severity)
	}

	c.JSON(http.StatusOK, EnhanceResponse{
		IsSafe:          result.IsSafe,
		HasSensitive:    result.HasSensitive,
		SensitiveResult: result.SensitiveResult,
		ContentHash:     result.ContentHash,
		Suggestions:     result.Suggestions,
		Warnings:        result.Warnings,
	})
}

// DetectSensitive 敏感信息检测
func (h *SecurityEnhanceHandler) DetectSensitive(c *gin.Context) {
	var req EnhanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	result := h.detector.Detect(req.Content)

	c.JSON(http.StatusOK, result)
}

// ComputeHash 计算内容哈希
func (h *SecurityEnhanceHandler) ComputeHash(c *gin.Context) {
	var req EnhanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	hash := h.enhancer.GetContentHash(req.Content)

	c.JSON(http.StatusOK, gin.H{
		"hash": hash,
	})
}

// MaskSensitive 脱敏处理
func (h *SecurityEnhanceHandler) MaskSensitive(c *gin.Context) {
	var req EnhanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	masked := h.enhancer.MaskSensitive(req.Content)

	c.JSON(http.StatusOK, gin.H{
		"masked_content": masked,
	})
}

// GetSecurityRules 获取安全规则
func (h *SecurityEnhanceHandler) GetSecurityRules(c *gin.Context) {
	rules := h.detector.GetRules()

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"count": len(rules),
	})
}

// QuickValidate 快速验证
func (h *SecurityEnhanceHandler) QuickValidate(c *gin.Context) {
	var req EnhanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	isValid, message := security.QuickValidate(req.Content)

	c.JSON(http.StatusOK, gin.H{
		"is_valid": isValid,
		"message":   message,
	})
}

// AuditLogRequest 审计日志请求
type AuditLogRequest struct {
	EventType string `json:"event_type"`
	SourceIP  string `json:"source_ip"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// GetAuditLogs 获取审计日志
func (h *SecurityEnhanceHandler) GetAuditLogs(c *gin.Context) {
	var req AuditLogRequest
	c.ShouldBindQuery(&req)

	// 设置默认时间范围：最近7天
	startTime := time.Now().AddDate(0, 0, -7)
	endTime := time.Now()

	events, err := h.logger.Query(
		startTime,
		endTime,
		req.EventType,
		req.SourceIP,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  len(events),
	})
}
