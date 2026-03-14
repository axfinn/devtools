package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditLogger 安全审计日志记录器
type AuditLogger struct {
	logDir  string
	mu      sync.RWMutex
	maxSize int64 // 单个日志文件最大大小
}

// AuditEvent 审计事件
type AuditEvent struct {
	EventID     string                 `json:"event_id"`      // 事件ID
	EventType   string                 `json:"event_type"`    // 事件类型
	Timestamp   string                 `json:"timestamp"`     // 时间戳
	SourceIP    string                 `json:"source_ip"`    // 来源IP
	UserAgent   string                 `json:"user_agent"`    // 用户代理
	Action      string                 `json:"action"`        // 操作
	Resource    string                 `json:"resource"`      // 资源
	Result      string                 `json:"result"`        // 结果: success, failure, blocked
	Details     map[string]interface{} `json:"details"`      // 详细信息
	Severity    string                 `json:"severity"`      // 严重程度: info, warning, error, critical
	ContentHash string                 `json:"content_hash"` // 内容哈希
}

// EventType 事件类型常量
var (
	EventTypeCreatePaste  = "create_paste"
	EventTypeAccessPaste  = "access_paste"
	EventTypeDeletePaste  = "delete_paste"
	EventTypeUploadFile   = "upload_file"
	EventTypeScanContent = "scan_content"
	EventTypeSensitive   = "sensitive_detected"
	EventTypeXSS         = "xss_detected"
	EventTypeAuth        = "authentication"
	EventTypeAdmin       = "admin_action"
)

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(logDir string) *AuditLogger {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建审计日志目录失败: %v\n", err)
	}

	return &AuditLogger{
		logDir:  logDir,
		maxSize: 10 * 1024 * 1024, // 10MB
	}
}

// Log 记录审计事件
func (l *AuditLogger) Log(event *AuditEvent) error {
	if event == nil {
		return fmt.Errorf("事件不能为空")
	}

	// 设置默认值
	if event.EventID == "" {
		event.EventID = generateEventID()
	}
	if event.Timestamp == "" {
		event.Timestamp = time.Now().Format(time.RFC3339)
	}
	if event.Severity == "" {
		event.Severity = "info"
	}

	// 转换为 JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败: %v", err)
	}

	// 添加换行符
	data = append(data, '\n')

	// 写入日志文件
	logFile := l.getLogFile()

	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// LogCreate 记录创建事件
func (l *AuditLogger) LogCreate(sourceIP, userAgent, resourceID, result string, details map[string]interface{}) error {
	event := &AuditEvent{
		EventType: EventTypeCreatePaste,
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Action:    "create",
		Resource:  resourceID,
		Result:    result,
		Details:   details,
	}
	return l.Log(event)
}

// LogAccess 记录访问事件
func (l *AuditLogger) LogAccess(sourceIP, userAgent, resourceID, result string, details map[string]interface{}) error {
	event := &AuditEvent{
		EventType: EventTypeAccessPaste,
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Action:    "access",
		Resource:  resourceID,
		Result:    result,
		Details:   details,
	}
	return l.Log(event)
}

// LogSensitive 记录敏感信息检测事件
func (l *AuditLogger) LogSensitive(sourceIP, userAgent, resourceID string, foundItems []string, severity string) error {
	event := &AuditEvent{
		EventType: EventTypeSensitive,
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Action:    "detect_sensitive",
		Resource:  resourceID,
		Result:    "detected",
		Severity:  severity,
		Details: map[string]interface{}{
			"found_items": foundItems,
		},
	}
	return l.Log(event)
}

// LogXSS 记录 XSS 检测事件
func (l *AuditLogger) LogXSS(sourceIP, userAgent, contentPreview string) error {
	event := &AuditEvent{
		EventType: EventTypeXSS,
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Action:    "xss_blocked",
		Result:    "blocked",
		Severity:  "warning",
		Details: map[string]interface{}{
			"content_preview": contentPreview[:min(len(contentPreview), 100)],
		},
	}
	return l.Log(event)
}

// LogAuth 记录认证事件
func (l *AuditLogger) LogAuth(sourceIP, userAgent, action, result string) error {
	event := &AuditEvent{
		EventType: EventTypeAuth,
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Action:    action,
		Result:    result,
	}
	return l.Log(event)
}

// LogAdmin 记录管理员操作
func (l *AuditLogger) LogAdmin(sourceIP, userAgent, action, resourceID, result string) error {
	event := &AuditEvent{
		EventType: EventTypeAdmin,
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Action:    action,
		Resource:  resourceID,
		Result:    result,
		Severity:  "warning",
	}
	return l.Log(event)
}

// getLogFile 获取日志文件路径
func (l *AuditLogger) getLogFile() string {
	date := time.Now().Format("2006-01-02")
	return filepath.Join(l.logDir, fmt.Sprintf("audit_%s.log", date))
}

// Query 查询审计日志
func (l *AuditLogger) Query(startTime, endTime time.Time, eventType string, sourceIP string) ([]*AuditEvent, error) {
	events := []*AuditEvent{}

	// 遍历日期范围内的日志文件
	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {
		logFile := filepath.Join(l.logDir, fmt.Sprintf("audit_%s.log", d.Format("2006-01-02")))

		data, err := os.ReadFile(logFile)
		if err != nil {
			continue // 文件不存在或无法读取
		}

		lines := splitLines(string(data))
		for _, line := range lines {
			line = trimSpace(line)
			if line == "" {
				continue
			}

			var event AuditEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			// 过滤条件
			if eventType != "" && event.EventType != eventType {
				continue
			}
			if sourceIP != "" && event.SourceIP != sourceIP {
				continue
			}

			eventTime, err := time.Parse(time.RFC3339, event.Timestamp)
			if err != nil {
				continue
			}

			if eventTime.Before(startTime) || eventTime.After(endTime) {
				continue
			}

			events = append(events, &event)
		}
	}

	return events, nil
}

// GetStatistics 获取审计统计信息
func (l *AuditLogger) GetStatistics(startTime, endTime time.Time) (map[string]interface{}, error) {
	events, err := l.Query(startTime, endTime, "", "")
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_events":    len(events),
		"by_type":         map[string]int{},
		"by_result":       map[string]int{},
		"by_severity":     map[string]int{},
		"by_source_ip":    map[string]int{},
	}

	typeCounts := stats["by_type"].(map[string]int)
	resultCounts := stats["by_result"].(map[string]int)
	severityCounts := stats["by_severity"].(map[string]int)
	ipCounts := stats["by_source_ip"].(map[string]int)

	for _, event := range events {
		typeCounts[event.EventType]++
		resultCounts[event.Result]++
		severityCounts[event.Severity]++
		ipCounts[event.SourceIP]++
	}

	return stats, nil
}

// CleanOldLogs 清理旧日志
func (l *AuditLogger) CleanOldLogs(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) < 12 || name[:6] != "audit_" {
			continue
		}

		dateStr := name[6 : len(name)-4] // 去掉 audit_ 和 .log
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if date.Before(cutoffDate) {
			os.Remove(filepath.Join(l.logDir, name))
		}
	}

	return nil
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("%d-%s", time.Now().Unix(), generateRandomString(8))
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
