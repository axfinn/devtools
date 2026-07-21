package middleware

import (
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

type RequestLogger struct {
	enabled bool
	entries chan models.HTTPRequestLog
}

func NewRequestLogger(db *models.DB, enabled bool) *RequestLogger {
	logger := &RequestLogger{enabled: enabled}
	if !enabled {
		return logger
	}
	logger.entries = make(chan models.HTTPRequestLog, 2048)
	go func() {
		for item := range logger.entries {
			_ = db.CreateHTTPRequestLog(&item)
		}
	}()
	return logger
}

func (l *RequestLogger) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.enabled || shouldSkipMonitoringRequest(c.Request.Method, c.Request.URL.Path, c.GetHeader("Upgrade")) {
			c.Next()
			return
		}

		startedAt := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		requestBytes := c.Request.ContentLength
		if requestBytes < 0 {
			requestBytes = 0
		}
		responseBytes := int64(c.Writer.Size())
		if responseBytes < 0 {
			responseBytes = 0
		}
		item := models.HTTPRequestLog{
			Method:        c.Request.Method,
			Path:          truncateMonitoringValue(c.Request.URL.Path, 512),
			Route:         truncateMonitoringValue(route, 512),
			StatusCode:    c.Writer.Status(),
			LatencyMS:     time.Since(startedAt).Milliseconds(),
			ClientIP:      truncateMonitoringValue(c.ClientIP(), 128),
			UserAgent:     truncateMonitoringValue(c.Request.UserAgent(), 512),
			RequestBytes:  requestBytes,
			ResponseBytes: responseBytes,
			ErrorMessage:  truncateMonitoringValue(c.Errors.String(), 1000),
			CreatedAt:     time.Now().UTC(),
		}
		select {
		case l.entries <- item:
		default:
		}
	}
}

func shouldSkipMonitoringRequest(method, path, upgrade string) bool {
	if method == "OPTIONS" || strings.EqualFold(upgrade, "websocket") {
		return true
	}
	for _, prefix := range []string{
		"/api/monitor",
		"/api/health",
		"/assets/",
		"/neon/",
		"/api/bg/cached/",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, exact := range []string{
		"/favicon.ico",
		"/manifest.webmanifest",
		"/sw.js",
		"/alipay.jpeg",
		"/wxpay.jpeg",
		"/pregnancy-shortcut-192.png",
		"/pregnancy-shortcut-512.png",
	} {
		if path == exact {
			return true
		}
	}
	return false
}

func truncateMonitoringValue(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
