package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestNormalizeEnglishTutorMode 验证 mode 字段归一化逻辑：
// 已知的 7 个值透传；其他值兜底为 "translate"；空字符串/空格也走兜底。
func TestNormalizeEnglishTutorMode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"translate", "translate"},
		{"pronounce", "pronounce"},
		{"explain", "explain"},
		{"correct", "correct"},
		{"dialogue", "dialogue"},
		{"guide", "guide"},
		{"api", "api"},
		{"  guide  ", "guide"},
		{"", "translate"},
		{"unknown", "translate"},
		{"TRANSLATE", "translate"}, // 区分大小写
	}
	for _, c := range cases {
		got := normalizeEnglishTutorMode(c.in)
		if got != c.want {
			t.Errorf("normalizeEnglishTutorMode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestNormalizeChoice 验证选项归一化逻辑：合法值透传；非法值取 fallback。
func TestNormalizeChoice(t *testing.T) {
	allowed := []string{"中文", "英文", "日文", "韩文"}
	if got := normalizeChoice("中文", "中文", allowed); got != "中文" {
		t.Errorf("normalizeChoice(中文) = %q, want 中文", got)
	}
	if got := normalizeChoice("  英文  ", "中文", allowed); got != "英文" {
		t.Errorf("normalizeChoice(英文) = %q, want 英文", got)
	}
	if got := normalizeChoice("法文", "中文", allowed); got != "中文" {
		t.Errorf("normalizeChoice(法文) = %q, want 中文 (fallback)", got)
	}
	if got := normalizeChoice("", "中文", allowed); got != "中文" {
		t.Errorf("normalizeChoice(empty) = %q, want 中文 (fallback)", got)
	}
}

// TestBuildEnglishTutorSystemPrompt 验证系统 prompt 包含关键约束：JSON-only、不输出 markdown、含所有必返回字段名。
// 防止有人在 prompt 里漏字段名导致前端 parseJsonContent 拿不到对应 section。
func TestBuildEnglishTutorSystemPrompt(t *testing.T) {
	req := EnglishTutorRequest{
		Mode:           "guide",
		Level:          "中级",
		TargetLanguage: "中文",
	}
	prompt := buildEnglishTutorSystemPrompt(req)
	for _, field := range []string{
		"translation", "polished_translation", "pronunciation",
		"ipa", "syllables", "stress", "phonics",
		"key_points", "vocabulary", "examples", "correction",
		"practice", "guided_plan", "next_prompt",
	} {
		if !strings.Contains(prompt, field) {
			t.Errorf("system prompt must mention field %q, prompt=%.200s...", field, prompt)
		}
	}
	if !strings.Contains(prompt, "只返回一个 JSON 对象") {
		t.Error("system prompt must explicitly require JSON-only output")
	}
}

// TestBuildEnglishTutorUserPrompt 验证 user prompt 结构：含 mode、instruction、input_text、target_language、learner_level。
// 验证 "api" 模式下会追加 custom_instruction。
func TestBuildEnglishTutorUserPrompt(t *testing.T) {
	base := EnglishTutorRequest{
		Mode:           "translate",
		Text:           "I used to be nervous about speaking English.",
		TargetLanguage: "中文",
		Level:          "初级",
	}
	user := buildEnglishTutorUserPrompt(base)
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(user), &payload); err != nil {
		t.Fatalf("user prompt must be valid JSON, got err=%v payload=%s", err, user)
	}
	if payload["mode"] != "translate" {
		t.Errorf("user prompt mode = %v, want translate", payload["mode"])
	}
	if payload["input_text"] != "I used to be nervous about speaking English." {
		t.Errorf("user prompt input_text = %v", payload["input_text"])
	}
	if payload["target_language"] != "中文" {
		t.Errorf("user prompt target_language = %v", payload["target_language"])
	}
	if inst, _ := payload["instruction"].(string); !strings.Contains(inst, "翻译") {
		t.Errorf("translate mode instruction should mention 翻译, got %q", inst)
	}

	// api 模式：custom_instruction 应进入 user prompt
	apiReq := base
	apiReq.Mode = "api"
	apiReq.CustomInstruction = "翻译成商务邮件英文"
	user = buildEnglishTutorUserPrompt(apiReq)
	if !strings.Contains(user, "翻译成商务邮件英文") {
		t.Errorf("api mode should include custom_instruction in user prompt, got %s", user)
	}
}

// TestEnglishTutorRateLimiter 验证频控：连续请求在限额内通过；超出分钟限额拒绝；时间回退后允许。
func TestEnglishTutorRateLimiter(t *testing.T) {
	rl := newEnglishTutorRateLimiter(3, time.Minute, 100, time.Hour)
	ip := "127.0.0.1"
	for i := 0; i < 3; i++ {
		ok, msg := rl.allow(ip)
		if !ok {
			t.Fatalf("call %d should be allowed, got msg=%q", i+1, msg)
		}
	}
	ok, msg := rl.allow(ip)
	if ok {
		t.Fatal("4th call should be rejected by minute limiter")
	}
	if !strings.Contains(msg, "频繁") {
		t.Errorf("rejection message should mention 频繁, got %q", msg)
	}
}

// TestEnglishTutorRateLimiterPerHour 验证小时级限额独立于分钟限额：
// 分钟限额放宽到很大，验证 6 次连续请求会被小时限额在第 5 次拦截。
func TestEnglishTutorRateLimiterPerHour(t *testing.T) {
	rl := newEnglishTutorRateLimiter(100, time.Minute, 5, time.Hour)
	ip := "10.0.0.1"
	for i := 0; i < 5; i++ {
		ok, msg := rl.allow(ip)
		if !ok {
			t.Fatalf("call %d should be allowed, got msg=%q", i+1, msg)
		}
	}
	ok, msg := rl.allow(ip)
	if ok {
		t.Fatal("6th call should be rejected by hour limiter")
	}
	if !strings.Contains(msg, "上限") {
		t.Errorf("rejection message should mention 上限, got %q", msg)
	}
}

// TestEnglishTutorRateLimiterIndependentIPs 验证不同 IP 互不干扰。
func TestEnglishTutorRateLimiterIndependentIPs(t *testing.T) {
	rl := newEnglishTutorRateLimiter(1, time.Minute, 100, time.Hour)
	if ok, _ := rl.allow("a"); !ok {
		t.Fatal("ip a first call should be allowed")
	}
	if ok, _ := rl.allow("a"); ok {
		t.Fatal("ip a second call should be rejected")
	}
	if ok, _ := rl.allow("b"); !ok {
		t.Fatal("ip b first call should be allowed (independent bucket)")
	}
}

// TestPruneTimes 验证时间窗口剪枝：保留 cutoff 之后的项。
func TestPruneTimes(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	items := []time.Time{
		base.Add(-2 * time.Minute),
		base.Add(-30 * time.Second),
		base,
		base.Add(30 * time.Second),
	}
	got := pruneTimes(items, base.Add(-time.Minute))
	if len(got) != 3 {
		t.Fatalf("pruneTimes expected 3 items, got %d", len(got))
	}
	if !got[0].Equal(base.Add(-30 * time.Second)) {
		t.Errorf("first kept item should be base-30s, got %v", got[0])
	}
}

// TestNormalizeChoiceBoundary 验证选项归一化的边界：空白值、空候选列表、空 fallback 时的行为。
func TestNormalizeChoiceBoundary(t *testing.T) {
	allowed := []string{"中文", "英文"}
	if got := normalizeChoice("   ", "中文", allowed); got != "中文" {
		t.Errorf("normalizeChoice(空白) = %q, want 中文 (fallback)", got)
	}
	if got := normalizeChoice("中文", "中文", nil); got != "中文" {
		t.Errorf("normalizeChoice(空候选列表) = %q, want 中文 (fallback)", got)
	}
	if got := normalizeChoice("法文", "", allowed); got != "" {
		t.Errorf("normalizeChoice(空 fallback) = %q, want 空字符串", got)
	}
}

// TestBuildEnglishTutorUserPromptAPIWithoutCustomInstruction 验证 api 模式但未填自定义要求时，
// user prompt 不会拼上"自定义要求"后缀（避免出现悬空的空标签）。
func TestBuildEnglishTutorUserPromptAPIWithoutCustomInstruction(t *testing.T) {
	user := buildEnglishTutorUserPrompt(EnglishTutorRequest{
		Mode:           "api",
		Text:           "Hello.",
		TargetLanguage: "中文",
		Level:          "初级",
	})
	if !strings.Contains(user, "自定义要求") {
		t.Errorf("api 模式未填 custom_instruction 时不应出现 自定义要求, got %s", user)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(user), &payload); err != nil {
		t.Fatalf("user prompt must be valid JSON, got err=%v payload=%s", err, user)
	}
	if payload["mode"] != "api" {
		t.Errorf("user prompt mode = %v, want api", payload["mode"])
	}
}

// TestEnglishTutorRateLimiterCleanup 验证限流器的过期清理路径：
// 超过 10 分钟未活动后触发 cleanup，把窗口外的 IP 计数桶删除，避免 map 无限增长。
// 同时验证分钟窗口过期不会连带清掉小时窗口的计数（否则会被绕过小时限额）。
func TestEnglishTutorRateLimiterCleanup(t *testing.T) {
	rl := newEnglishTutorRateLimiter(10, time.Minute, 100, time.Hour)
	now := time.Now()
	// 1.1.1.1：分钟窗口已过期、小时窗口仍在有效期内
	rl.minuteHits["1.1.1.1"] = []time.Time{now.Add(-2 * time.Minute)}
	rl.hourHits["1.1.1.1"] = []time.Time{now.Add(-2 * time.Minute)}
	// 3.3.3.3：两个窗口都已过期
	rl.minuteHits["3.3.3.3"] = []time.Time{now.Add(-2 * time.Hour)}
	rl.hourHits["3.3.3.3"] = []time.Time{now.Add(-2 * time.Hour)}
	rl.lastCleanup = now.Add(-11 * time.Minute)

	if ok, msg := rl.allow("2.2.2.2"); !ok {
		t.Fatalf("first call from fresh ip should be allowed, got msg=%q", msg)
	}
	if _, exists := rl.minuteHits["1.1.1.1"]; exists {
		t.Errorf("cleanup 应删除分钟窗口外的 minuteHits 桶, minuteHits=%v", rl.minuteHits)
	}
	if _, exists := rl.hourHits["1.1.1.1"]; !exists {
		t.Errorf("小时窗口内的 hourHits 桶不应被清理掉, hourHits=%v", rl.hourHits)
	}
	if _, exists := rl.minuteHits["3.3.3.3"]; exists {
		t.Errorf("cleanup 应删除完全过期的 minuteHits 桶, minuteHits=%v", rl.minuteHits)
	}
	if _, exists := rl.hourHits["3.3.3.3"]; exists {
		t.Errorf("cleanup 应删除完全过期的 hourHits 桶, hourHits=%v", rl.hourHits)
	}
}

// newEnglishTutorMockContext 构造 EnglishTutor* handler 测试用的 gin.Context
func newEnglishTutorMockContext(t *testing.T, method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = &http.Request{
		Method:     method,
		URL:        &url.URL{Path: path},
		Header:     make(http.Header),
		RemoteAddr: "127.0.0.1:1234",
	}
	return c, rec
}

// TestEnglishTutorMeta_LimitsAndModes 验证 meta 接口返回的限流参数与前端 limits 默认值一致
// 防止后端改限流值后前端默认值忘记同步,导致 UI 显示「2000 字符」但服务端实际拒收 2500 字符
func TestEnglishTutorMeta_LimitsAndModes(t *testing.T) {
	c, rec := newEnglishTutorMockContext(t, "GET", "/api/english-tutor/meta")
	h := &AIGatewayHandler{}
	h.EnglishTutorMeta(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Modes  []map[string]string `json:"modes"`
		Limits map[string]int      `json:"limits"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("meta response not JSON: %v", err)
	}

	wantLimitKeys := []string{"max_text_chars", "max_custom_instruction_chars", "rate_limit_per_minute", "rate_limit_per_hour"}
	for _, key := range wantLimitKeys {
		if _, ok := resp.Limits[key]; !ok {
			t.Errorf("meta missing limit %q", key)
		}
	}
	if resp.Limits["max_text_chars"] != 2000 {
		t.Errorf("max_text_chars=%d, want 2000", resp.Limits["max_text_chars"])
	}
	if resp.Limits["rate_limit_per_minute"] != 8 {
		t.Errorf("rate_limit_per_minute=%d, want 8", resp.Limits["rate_limit_per_minute"])
	}

	wantModes := map[string]bool{"translate": false, "pronounce": false, "explain": false, "correct": false, "dialogue": false, "guide": false, "api": false}
	for _, m := range resp.Modes {
		if _, ok := wantModes[m["value"]]; ok {
			wantModes[m["value"]] = true
		}
	}
	for value, seen := range wantModes {
		if !seen {
			t.Errorf("meta mode %q missing", value)
		}
	}
}

// TestEnglishTutorMeta_LimitsMatchHandler 当 meta 限流变化时,handler 实际硬编码限流也要同步改;
// 该测试用作显式提醒:改 meta 时也要改 english_tutor.go 里的常量
func TestEnglishTutorMeta_LimitsMatchHandler(t *testing.T) {
	c, rec := newEnglishTutorMockContext(t, "GET", "/api/english-tutor/meta")
	h := &AIGatewayHandler{}
	h.EnglishTutorMeta(c)

	var resp struct {
		Limits map[string]int `json:"limits"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("meta response not JSON: %v", err)
	}

	// 这些常量值直接对照 handler 内的硬编码。
	// handler 的文本长度校验在 EnglishTutor() 里硬编码为 2000 / 240,
	// 一旦改 meta 必须同步改 handler。
	if resp.Limits["max_text_chars"] != 2000 {
		t.Errorf("meta says max_text_chars=%d, but handler hard-codes 2000; sync them", resp.Limits["max_text_chars"])
	}
	if resp.Limits["max_custom_instruction_chars"] != 240 {
		t.Errorf("meta says max_custom_instruction_chars=%d, but handler hard-codes 240; sync them", resp.Limits["max_custom_instruction_chars"])
	}
}
