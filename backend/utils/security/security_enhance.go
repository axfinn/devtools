package security

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// SecurityEnhancer 安全增强模块
type SecurityEnhancer struct {
	hasher            *Hasher
	sensitiveDetector *SensitiveInfoDetector
}

// NewSecurityEnhancer 创建安全增强模块
func NewSecurityEnhancer() *SecurityEnhancer {
	return &SecurityEnhancer{
		hasher:            NewHasher("sha256"),
		sensitiveDetector: NewSensitiveInfoDetector(),
	}
}

// SecurityResult 安全增强结果
type SecurityResult struct {
	IsSafe          bool                   `json:"is_safe"`           // 是否安全
	HasSensitive    bool                   `json:"has_sensitive"`    // 是否包含敏感信息
	SensitiveResult *DetectionResult       `json:"sensitive_result"` // 敏感信息检测结果
	ContentHash     string                 `json:"content_hash"`     // 内容哈希
	Suggestions     []string               `json:"suggestions"`      // 建议
	Warnings        []string               `json:"warnings"`         // 警告
}

// EnhanceSecurity 执行安全增强
func (s *SecurityEnhancer) EnhanceSecurity(content string) *SecurityResult {
	result := &SecurityResult{
		IsSafe:      true,
		Suggestions: []string{},
		Warnings:    []string{},
	}

	// 1. 计算内容哈希
	hash := s.hasher.ComputeHash(content)
	result.ContentHash = hash.Hash

	// 2. 检测敏感信息
	sensitiveResult := s.sensitiveDetector.Detect(content)
	result.SensitiveResult = sensitiveResult
	result.HasSensitive = sensitiveResult.IsDetected

	if sensitiveResult.IsDetected {
		result.IsSafe = false
		result.Suggestions = append(result.Suggestions, sensitiveResult.Suggestions...)
		result.Warnings = append(result.Warnings, sensitiveResult.Summary)
	}

	// 3. XSS 检测
	if containsXSS(content) {
		result.IsSafe = false
		result.Warnings = append(result.Warnings, "检测到潜在 XSS 攻击")
		result.Suggestions = append(result.Suggestions, "请移除可能包含恶意脚本的内容")
	}

	return result
}

// ValidateContent 验证内容安全性
func (s *SecurityEnhancer) ValidateContent(content string) (bool, string) {
	result := s.EnhanceSecurity(content)

	if !result.IsSafe {
		return false, strings.Join(result.Warnings, "; ")
	}

	return true, ""
}

// MaskSensitive 脱敏内容
func (s *SecurityEnhancer) MaskSensitive(content string) string {
	result := s.sensitiveDetector.Detect(content)
	if !result.IsDetected {
		return content
	}

	masked := content
	for _, item := range result.FoundItems {
		// 使用正则替换原始内容
		re := regexp.MustCompile(regexp.QuoteMeta(item.Value))
		masked = re.ReplaceAllString(masked, "[脱敏]")
	}

	return masked
}

// GetContentHash 获取内容哈希
func (s *SecurityEnhancer) GetContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// containsXSS 检测 XSS 特征
func containsXSS(content string) bool {
	xssPatterns := []string{
		`<script[^>]*>`,
		`javascript:`,
		`onerror=`,
		`onclick=`,
		`onload=`,
		`eval\(`,
		`expression\(`,
		`<iframe`,
		`<object`,
		`<embed`,
		`data:text/html`,
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range xssPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(contentLower) {
			return true
		}
	}

	return false
}

// QuickValidate 快速验证
func QuickValidate(content string) (bool, string) {
	enhancer := NewSecurityEnhancer()
	return enhancer.ValidateContent(content)
}
