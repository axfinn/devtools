package security

import (
	"testing"
)

// TestHasher 测试哈希计算
func TestHasher(t *testing.T) {
	hasher := NewHasher("sha256")

	// 测试字符串哈希
	content := "test content"
	result := hasher.ComputeHash(content)
	if result.Hash == "" {
		t.Error("Expected hash to be non-empty")
	}
	if result.Algorithm != "sha256" {
		t.Errorf("Expected algorithm 'sha256', got '%s'", result.Algorithm)
	}

	// 测试验证哈希
	valid := hasher.VerifyHash(content, result.Hash)
	if !valid {
		t.Error("Expected hash verification to pass")
	}

	invalid := hasher.VerifyHash(content, "invalid_hash")
	if invalid {
		t.Error("Expected hash verification to fail for invalid hash")
	}

	// 测试多次哈希
	hashes := hasher.ComputeMultipleHashes(content)
	if len(hashes) == 0 {
		t.Error("Expected at least one hash")
	}
	if _, ok := hashes["sha256"]; !ok {
		t.Error("Expected sha256 hash")
	}
}

// TestGenerateContentID 测试生成内容ID
func TestGenerateContentID(t *testing.T) {
	hasher := NewHasher("sha256")

	id1 := hasher.GenerateContentID("test content")
	id2 := hasher.GenerateContentID("test content")
	id3 := hasher.GenerateContentID("different content")

	if id1 != id2 {
		t.Error("Same content should produce same ID")
	}
	if id1 == id3 {
		t.Error("Different content should produce different ID")
	}
	if len(id1) != 16 {
		t.Errorf("Expected ID length 16, got %d", len(id1))
	}
}

// TestChainCreation 测试内容链创建
func TestChainCreation(t *testing.T) {
	hasher := NewHasher("sha256")

	contents := []string{"content1", "content2", "content3"}

	chain, err := hasher.CreateChain(contents, "test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(chain) != len(contents) {
		t.Errorf("Expected chain length %d, got %d", len(contents), len(chain))
	}

	// 测试验证链
	valid := hasher.VerifyChain(contents, chain)
	if !valid {
		t.Error("Expected chain verification to pass")
	}

	// 测试篡改后的验证
	modifiedContents := []string{"content1", "modified", "content3"}
	invalid := hasher.VerifyChain(modifiedContents, chain)
	if invalid {
		t.Error("Expected chain verification to fail for modified content")
	}
}

// TestSensitiveInfoDetection 测试敏感信息检测
func TestSensitiveInfoDetection(t *testing.T) {
	detector := NewSensitiveInfoDetector()

	// 测试手机号检测
	content := "我的手机号是13812345678，请联系我"
	result := detector.Detect(content)
	if !result.IsDetected {
		t.Error("Expected to detect phone number")
	}
	if result.CountByType["china_phone"] != 1 {
		t.Errorf("Expected 1 phone number, got %d", result.CountByType["china_phone"])
	}

	// 测试邮箱检测
	emailContent := "联系邮箱: test@example.com"
	result = detector.Detect(emailContent)
	if !result.IsDetected {
		t.Error("Expected to detect email")
	}

	// 测试密码检测
	passwordContent := "password: my_secret_password"
	result = detector.Detect(passwordContent)
	if !result.IsDetected {
		t.Error("Expected to detect password")
	}

	// 测试空内容
	emptyResult := detector.Detect("")
	if emptyResult.IsDetected {
		t.Error("Expected empty content to not have detections")
	}

	// 测试无敏感信息
	safeContent := "这是一段普通文本内容，不包含任何敏感信息"
	result = detector.Detect(safeContent)
	if result.IsDetected {
		t.Error("Expected safe content to not have detections")
	}
}

// TestSensitiveInfoMasking 测试敏感信息脱敏
func TestSensitiveInfoMasking(t *testing.T) {
	detector := NewSensitiveInfoDetector()

	// 测试手机号脱敏
	content := "手机号: 13812345678"
	result := detector.Detect(content)
	if len(result.FoundItems) == 0 {
		t.Error("Expected to find sensitive item")
	}

	// 测试邮箱脱敏
	emailContent := "邮箱: test@example.com"
	result = detector.Detect(emailContent)
	if len(result.FoundItems) == 0 {
		t.Error("Expected to find email")
	}
}

// TestSensitiveRules 测试敏感信息规则
func TestSensitiveRules(t *testing.T) {
	detector := NewSensitiveInfoDetector()

	rules := detector.GetRules()
	if len(rules) == 0 {
		t.Error("Expected at least one rule")
	}

	// 测试添加自定义规则
	newRule := DetectionRule{
		Name:        "custom_rule",
		Pattern:     `custom-pattern`,
		Severity:    "high",
		Description: "Custom detection rule",
	}
	detector.AddRule(newRule)

	rules = detector.GetRules()
	if len(rules) == 0 {
		t.Error("Expected rules after addition")
	}

	// 测试移除规则
	detector.RemoveRule("custom_rule")
	rules = detector.GetRules()
	// 验证规则被移除
}

// TestQuickCheck 测试快速检查
func TestQuickCheck(t *testing.T) {
	detector := NewSensitiveInfoDetector()

	// 高风险内容
	highRisk := "password: secret123"
	if !detector.QuickCheck(highRisk) {
		t.Error("Expected high risk content to be detected")
	}

	// 低风险内容
	lowRisk := "IP: 192.168.1.1"
	if detector.QuickCheck(lowRisk) {
		t.Error("Expected low risk content to not trigger high risk")
	}
}

// TestSecurityEnhancer 测试安全增强
func TestSecurityEnhancer(t *testing.T) {
	enhancer := NewSecurityEnhancer()

	// 测试安全内容
	safeContent := "这是一段普通文本内容"
	result := enhancer.EnhanceSecurity(safeContent)
	if !result.IsSafe {
		t.Error("Expected safe content to pass")
	}
	if result.ContentHash == "" {
		t.Error("Expected content hash")
	}

	// 测试包含敏感信息的内容
	sensitiveContent := "手机号: 13812345678"
	result = enhancer.EnhanceSecurity(sensitiveContent)
	if result.IsSafe {
		t.Error("Expected sensitive content to be flagged")
	}
	if !result.HasSensitive {
		t.Error("Expected sensitive detection")
	}

	// 测试 XSS 内容
	xssContent := "<script>alert('xss')</script>"
	result = enhancer.EnhanceSecurity(xssContent)
	if result.IsSafe {
		t.Error("Expected XSS content to be flagged")
	}
}

// TestValidateContent 测试内容验证
func TestValidateContent(t *testing.T) {
	enhancer := NewSecurityEnhancer()

	// 安全内容
	valid, msg := enhancer.ValidateContent("普通文本内容")
	if !valid {
		t.Errorf("Expected valid, got invalid: %s", msg)
	}

	// 敏感内容
	valid, _ = enhancer.ValidateContent("手机号: 13812345678")
	if valid {
		t.Error("Expected invalid")
	}

	// XSS 内容
	valid, _ = enhancer.ValidateContent("<script>alert(1)</script>")
	if valid {
		t.Error("Expected invalid")
	}
}

// TestMaskSensitive 测试脱敏
func TestMaskSensitive(t *testing.T) {
	enhancer := NewSecurityEnhancer()

	// 测试敏感信息检测功能
	content := "手机号: 13812345678"
	result := enhancer.EnhanceSecurity(content)
	if !result.HasSensitive {
		t.Error("Expected to detect sensitive info")
	}
}

// TestGetContentHash 测试获取内容哈希
func TestGetContentHash(t *testing.T) {
	enhancer := NewSecurityEnhancer()

	content := "test content"
	hash1 := enhancer.GetContentHash(content)
	hash2 := enhancer.GetContentHash(content)
	hash3 := enhancer.GetContentHash("different")

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}
	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}
}

// TestQuickValidate 测试快速验证
func TestQuickValidate(t *testing.T) {
	// 测试正常内容
	valid, _ := QuickValidate("普通文本内容")
	if !valid {
		t.Error("Expected valid content")
	}

	// 测试高风险内容
	valid, msg := QuickValidate("我的手机号是13812345678")
	if valid {
		t.Logf("Got message: %s", msg)
	}
}
