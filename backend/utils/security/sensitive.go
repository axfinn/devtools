package security

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// SensitiveInfoDetector 敏感信息检测器
type SensitiveInfoDetector struct {
	rules []DetectionRule
}

// DetectionRule 检测规则
type DetectionRule struct {
	Name        string `json:"name"`         // 规则名称
	Pattern     string `json:"pattern"`      // 正则表达式
	Severity    string `json:"severity"`     // 严重程度: high, medium, low
	Description string `json:"description"`  // 描述
}

// DetectionResult 检测结果
type DetectionResult struct {
	IsDetected  bool              `json:"is_detected"`  // 是否检测到
	FoundItems []FoundItem       `json:"found_items"`  // 检测到的项目
	Severity    string           `json:"severity"`     // 最高严重程度
	Suggestions []string         `json:"suggestions"`   // 建议
	Summary     string           `json:"summary"`      // 摘要
	CountByType map[string]int   `json:"count_by_type"` // 按类型统计
}

// FoundItem 检测到的项目
type FoundItem struct {
	Type        string `json:"type"`        // 类型
	Value       string `json:"value"`       // 检测到的值（脱敏后）
	Position    int    `json:"position"`   // 位置
	Severity    string `json:"severity"`   // 严重程度
	Description string `json:"description"`  // 描述
}

// NewSensitiveInfoDetector 创建敏感信息检测器
func NewSensitiveInfoDetector() *SensitiveInfoDetector {
	detector := &SensitiveInfoDetector{
		rules: []DetectionRule{
			// 身份证号
			{
				Name:        "china_id_card",
				Pattern:     `(?i)(1[1-5]|2[1-5]|3[1-6]|4[1-7]|5[0-4]|6[1-5])\d{4}(19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]`,
				Severity:    "high",
				Description: "中国身份证号",
			},
			// 手机号（中国）
			{
				Name:        "china_phone",
				Pattern:     `(?i)1[3-9]\d{9}`,
				Severity:    "high",
				Description: "中国手机号",
			},
			// 邮箱
			{
				Name:        "email",
				Pattern:     `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
				Severity:    "medium",
				Description: "电子邮箱",
			},
			// 银行卡
			{
				Name:        "bank_card",
				Pattern:     `(?i)\b([4-6]\d{3}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4})\b`,
				Severity:    "high",
				Description: "银行卡号",
			},
			// 密码相关
			{
				Name:        "password",
				Pattern:     `(?i)(password|passwd|pwd|secret|api_key|apikey|api-key)\s*[:=]\s*["']?[^"'\s]+["']?`,
				Severity:    "high",
				Description: "密码或密钥",
			},
			// API Key
			{
				Name:        "api_key",
				Pattern:     `(?i)(api[_-]?key|apikey|access[_-]?token|auth[_-]?token)\s*[:=]\s*["']?[a-zA-Z0-9_\-]{20,}["']?`,
				Severity:    "high",
				Description: "API 密钥",
			},
			// JWT Token
			{
				Name:        "jwt_token",
				Pattern:     `eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`,
				Severity:    "high",
				Description: "JWT Token",
			},
			// AWS Access Key
			{
				Name:        "aws_key",
				Pattern:     `(?i)(AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}`,
				Severity:    "high",
				Description: "AWS Access Key",
			},
			// 私钥
			{
				Name:        "private_key",
				Pattern:     `-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
				Severity:    "high",
				Description: "私钥",
			},
			// IP 地址
			{
				Name:        "ip_address",
				Pattern:     `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
				Severity:    "low",
				Description: "IP 地址",
			},
			// URL 中的凭据
			{
				Name:        "url_credential",
				Pattern:     `(?i)https?://[^:]+:[^@]+@`,
				Severity:    "high",
				Description: "URL 中的凭据",
			},
			// 信用卡号（简单检测）
			{
				Name:        "credit_card",
				Pattern:     `(?i)\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`,
				Severity:    "high",
				Description: "信用卡号",
			},
			// 社保号
			{
				Name:        "ssn",
				Pattern:     `(?i)\b\d{3}-\d{2}-\d{4}\b`,
				Severity:    "high",
				Description: "社保号",
			},
			// 比特币地址
			{
				Name:        "bitcoin_address",
				Pattern:     `[13][a-km-zA-HJ-NP-Z1-9]{25,34}`,
				Severity:    "medium",
				Description: "比特币地址",
			},
			// GitHub Token
			{
				Name:        "github_token",
				Pattern:     `(?i)gh[pousr]_[a-zA-Z0-9]{36,}`,
				Severity:    "high",
				Description: "GitHub Token",
			},
		},
	}

	return detector
}

// Detect 检测敏感信息
func (d *SensitiveInfoDetector) Detect(content string) *DetectionResult {
	result := &DetectionResult{
		IsDetected:  false,
		FoundItems:  []FoundItem{},
		Severity:    "none",
		Suggestions: []string{},
		CountByType: make(map[string]int),
	}

	if content == "" {
		result.Summary = "内容为空"
		return result
	}

	// 按规则检测
	for _, rule := range d.rules {
		re := regexp.MustCompile(rule.Pattern)
		matches := re.FindAllStringIndex(content, -1)

		for _, match := range matches {
			value := content[match[0]:match[1]]

			// 脱敏处理
			maskedValue := maskValue(value, rule.Name)

			item := FoundItem{
				Type:        rule.Name,
				Value:       maskedValue,
				Position:    match[0],
				Severity:    rule.Severity,
				Description: rule.Description,
			}

			result.FoundItems = append(result.FoundItems, item)
			result.CountByType[rule.Name]++
		}
	}

	// 设置结果
	if len(result.FoundItems) > 0 {
		result.IsDetected = true
		result.Severity = d.calculateMaxSeverity(result.FoundItems)
		result.Summary = d.generateSummary(result)
		result.Suggestions = d.generateSuggestions(result)
	} else {
		result.Summary = "未检测到敏感信息"
	}

	return result
}

// DetectWithContext 检测敏感信息（带上下文）
func (d *SensitiveInfoDetector) DetectWithContext(content string, contextSize int) *DetectionResult {
	result := d.Detect(content)

	// 为每个检测结果添加上下文
	for i := range result.FoundItems {
		item := &result.FoundItems[i]
		start := intMax(0, item.Position-contextSize)
		end := intMin(len(content), item.Position+len(item.Value)+contextSize)
		item.Value = content[start:end]
	}

	return result
}

// maskValue 脱敏处理
func maskValue(value, ruleName string) string {
	length := len(value)

	switch ruleName {
	case "email":
		// 邮箱脱敏: a***@example.com
		parts := strings.Split(value, "@")
		if len(parts) == 2 && len(parts[0]) > 2 {
			return parts[0][:2] + "***@" + parts[1]
		}
		return "***@***.***"

	case "china_phone":
		// 手机号脱敏: 138****1234
		if length >= 11 {
			return value[:3] + "****" + value[7:]
		}
		return "***-****-****"

	case "china_id_card":
		// 身份证脱敏: 110101****12345678
		if length >= 18 {
			return value[:6] + "********" + value[14:]
		}
		return "***********"

	case "password", "api_key", "jwt_token", "aws_key", "private_key", "github_token":
		return "***REDACTED***"

	case "bank_card", "credit_card":
		// 银行卡脱敏: **** **** **** 1234
		if length >= 16 {
			return "**** **** **** " + value[length-4:]
		}
		return "**** **** **** ****"

	case "url_credential":
		// URL 脱敏: https://***:***@example.com
		return "https://***:***@***"

	default:
		if length > 8 {
			return value[:4] + "****" + value[length-4:]
		}
		return "***"
	}
}

// calculateMaxSeverity 计算最高严重程度
func (d *SensitiveInfoDetector) calculateMaxSeverity(items []FoundItem) string {
	severityOrder := map[string]int{
		"high":   3,
		"medium": 2,
		"low":    1,
		"none":   0,
	}

	maxSeverity := "none"
	maxLevel := 0

	for _, item := range items {
		if level, ok := severityOrder[item.Severity]; ok && level > maxLevel {
			maxSeverity = item.Severity
			maxLevel = level
		}
	}

	return maxSeverity
}

// generateSummary 生成摘要
func (d *SensitiveInfoDetector) generateSummary(result *DetectionResult) string {
	var parts []string
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for _, item := range result.FoundItems {
		switch item.Severity {
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		}
	}

	if highCount > 0 {
		parts = append(parts, fmt.Sprintf("检测到 %d 个高风险敏感信息", highCount))
	}
	if mediumCount > 0 {
		parts = append(parts, fmt.Sprintf("%d 个中风险", mediumCount))
	}
	if lowCount > 0 {
		parts = append(parts, fmt.Sprintf("%d 个低风险", lowCount))
	}

	return strings.Join(parts, ", ")
}

// generateSuggestions 生成建议
func (d *SensitiveInfoDetector) generateSuggestions(result *DetectionResult) []string {
	suggestions := []string{}

	switch result.Severity {
	case "high":
		suggestions = append(suggestions,
			"检测到高风险敏感信息，请立即处理",
			"建议移除或脱敏后再分享",
		)
	case "medium":
		suggestions = append(suggestions,
			"检测到中风险敏感信息，请注意保护",
			"建议检查是否需要脱敏",
		)
	case "low":
		suggestions = append(suggestions,
			"检测到低风险信息，请确认是否需要分享",
		)
	}

	if result.CountByType["password"] > 0 || result.CountByType["api_key"] > 0 {
		suggestions = append(suggestions, "检测到密码或密钥，请务必移除后再分享")
	}

	if result.CountByType["private_key"] > 0 {
		suggestions = append(suggestions, "检测到私钥，这可能导致严重的安全问题，请立即移除")
	}

	return suggestions
}

// ScanFile 扫描文件中的敏感信息
func (d *SensitiveInfoDetector) ScanFile(filePath string) (*DetectionResult, error) {
	content, err := readFileContent(filePath)
	if err != nil {
		return nil, err
	}
	return d.Detect(content), nil
}

// QuickCheck 快速检查（只检查高风险）
func (d *SensitiveInfoDetector) QuickCheck(content string) bool {
	result := d.Detect(content)
	return result.Severity == "high"
}

// AddRule 添加自定义规则
func (d *SensitiveInfoDetector) AddRule(rule DetectionRule) {
	d.rules = append(d.rules, rule)
}

// GetRules 获取所有规则
func (d *SensitiveInfoDetector) GetRules() []DetectionRule {
	return d.rules
}

// RemoveRule 移除规则
func (d *SensitiveInfoDetector) RemoveRule(name string) {
	var newRules []DetectionRule
	for _, rule := range d.rules {
		if rule.Name != name {
			newRules = append(newRules, rule)
		}
	}
	d.rules = newRules
}

func readFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// QuickDetect 快速检测（直接调用）
func QuickDetect(content string) *DetectionResult {
	detector := NewSensitiveInfoDetector()
	return detector.Detect(content)
}

// QuickMask 快速脱敏
func QuickMask(content string) string {
	result := QuickDetect(content)
	if !result.IsDetected {
		return content
	}

	// 简单替换 - 实际生产环境应该更精细处理
	masked := content
	for _, item := range result.FoundItems {
		masked = strings.Replace(masked, item.Value, "[敏感信息已脱敏]", -1)
	}
	return masked
}
