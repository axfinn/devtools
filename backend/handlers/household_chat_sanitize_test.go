package handlers

import (
	"strings"
	"testing"

	"devtools/utils"
)

// TestSanitizeHTML_StripsScripts 验证 LLM 输出里的 <script> 被 bluemonday StrictPolicy
// 完全移除,不留任何 tag / 内容(这是防 stored XSS 的核心)。
//
// 与 household Chat 端点配合:utils.SanitizeHTML(resp.Reply) 在保存到
// HouseholdConversation.Content 前调用(handlers/household.go:2685),
// 防止 prompt injection 之后前端 v-html 渲染恶意脚本。
func TestSanitizeHTML_StripsScripts(t *testing.T) {
	cases := []struct {
		name  string
		input string
		// 不应出现在输出里
		mustNot []string
		// 应保留的子串(可空)
		mustKeep []string
	}{
		{
			name:     "纯 script 标签",
			input:    `<script>alert(1)</script>`,
			mustNot:  []string{"<script", "</script>", "alert(1)"},
			mustKeep: nil,
		},
		{
			name:     "script 标签带属性",
			input:    `<script src="evil.js">alert(1)</script>`,
			mustNot:  []string{"<script", "alert", "evil.js"},
			mustKeep: nil,
		},
		{
			name:     "大小写混用 SCript",
			input:    `<SCript>alert('xss')</sCRipT>`,
			mustNot:  []string{"alert", "<SCript", "</sCRipT"},
			mustKeep: nil,
		},
		{
			name:     "script 与正常文本混合,正常文本应保留",
			input:    `hello <script>alert(1)</script> world`,
			mustNot:  []string{"<script", "alert"},
			mustKeep: []string{"hello", "world"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := utils.SanitizeHTML(c.input)
			for _, bad := range c.mustNot {
				if strings.Contains(strings.ToLower(got), strings.ToLower(bad)) {
					t.Errorf("SanitizeHTML 残留 %q in output: %q", bad, got)
				}
			}
			for _, keep := range c.mustKeep {
				if !strings.Contains(got, keep) {
					t.Errorf("SanitizeHTML 丢失了 %q, 输出: %q", keep, got)
				}
			}
		})
	}
}

// TestSanitizeHTML_StripsEventHandlers 验证 on* 事件处理器属性被剥掉,
// 即使出现在白名单标签(如 img/a)上也一样。
func TestSanitizeHTML_StripsEventHandlers(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		mustNot  []string
		mustKeep []string
	}{
		{
			name:    "img onerror",
			input:   `<img src=x onerror=alert(1)>`,
			mustNot: []string{"onerror", "alert"},
		},
		{
			name:    "img onload",
			input:   `<img src=x onload=alert(1)>`,
			mustNot: []string{"onload", "alert"},
		},
		{
			name:     "a onclick",
			input:    `<a href="https://example.com" onclick="evil()">click</a>`,
			mustNot:  []string{"onclick", "evil"},
			mustKeep: []string{"click"},
		},
		{
			name:     "div onmouseover",
			input:    `<div onmouseover=steal()>hover</div>`,
			mustNot:  []string{"onmouseover", "steal"},
			mustKeep: []string{"hover"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := utils.SanitizeHTML(c.input)
			for _, bad := range c.mustNot {
				if strings.Contains(strings.ToLower(got), strings.ToLower(bad)) {
					t.Errorf("SanitizeHTML 残留 %q in output: %q", bad, got)
				}
			}
			for _, keep := range c.mustKeep {
				if !strings.Contains(got, keep) {
					t.Errorf("SanitizeHTML 丢失了 %q, 输出: %q", keep, got)
				}
			}
		})
	}
}

// TestSanitizeHTML_JavaScriptURI 验证 javascript: 协议被剥离,
// 这条对 <a href="javascript:..."> 类钓鱼尤其关键。
//
// bluemonday StrictPolicy 对未通过 AllowURLSchemes 显式声明的 scheme 会拒绝,
// javascript: 一定在拒绝列表里。
func TestSanitizeHTML_JavaScriptURI(t *testing.T) {
	cases := []string{
		`<a href="javascript:alert(1)">click</a>`,
		`<a href='javascript:alert(1)'>click</a>`,
		`<a href=JAVASCRIPT:alert(1)>click</a>`,
		`<a href="  javascript:alert(1)">click</a>`,
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := utils.SanitizeHTML(input)
			if strings.Contains(strings.ToLower(got), "javascript:") {
				t.Errorf("javascript: 协议未剥离, 输出: %q", got)
			}
			if !strings.Contains(got, "click") {
				t.Errorf("可见文本丢失, 输出: %q", got)
			}
		})
	}
}

// TestSanitizeHTML_PreservesFormatting 验证 bluemonday allowlist 里的
// 格式标签(b/i/em/strong/code/pre 等)与文本被保留。
// 这些是 Chat 助手输出"代码块/强调"必须保留的语义。
func TestSanitizeHTML_PreservesFormatting(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		mustKeep []string
	}{
		{
			name:     "粗体",
			input:    `<b>bold</b>`,
			mustKeep: []string{"<b>", "bold", "</b>"},
		},
		{
			name:     "斜体",
			input:    `<i>italic</i>`,
			mustKeep: []string{"<i>", "italic", "</i>"},
		},
		{
			name:     "强调",
			input:    `<em>emphasize</em>`,
			mustKeep: []string{"<em>", "emphasize"},
		},
		{
			name:     "strong",
			input:    `<strong>strong</strong>`,
			mustKeep: []string{"<strong>", "strong"},
		},
		{
			name:     "代码块",
			input:    `<pre><code>fmt.Println("hi")</code></pre>`,
			mustKeep: []string{"<pre>", "<code>", `Println`},
		},
		{
			name:     "纯文本",
			input:    `hello world`,
			mustKeep: []string{"hello world"},
		},
		{
			name:     "空字符串",
			input:    ``,
			mustKeep: nil, // 期望返回 ""
		},
		{
			name:     "段落",
			input:    `<p>paragraph</p>`,
			mustKeep: []string{"<p>", "paragraph", "</p>"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := utils.SanitizeHTML(c.input)
			for _, keep := range c.mustKeep {
				if !strings.Contains(got, keep) {
					t.Errorf("SanitizeHTML 丢失了 %q, 输出: %q", keep, got)
				}
			}
		})
	}
}

// TestSanitizeHTML_StripsDangerousTags 验证 iframe/object/embed/form/input/button/meta/link/base/applet
// 危险标签被完全剥掉(utils/sanitizer.go:22 dangerousTags)。
//
// 注意:这部分逻辑在 utils.SanitizeContent 里是显式 regex 剥离;
// SanitizeHTML 走 bluemonday,bluemonday StrictPolicy 已经天然不识别这些标签,
// 因此输出里不应保留任何 form/input/iframe 等标签。
func TestSanitizeHTML_StripsDangerousTags(t *testing.T) {
	cases := []string{
		`<iframe src="https://evil.com"></iframe>`,
		`<form action="/steal"><input name="x"></form>`,
		`<object data="evil.swf"></object>`,
		`<embed src="evil.swf">`,
		`<button onclick="evil()">click</button>`,
		`<meta http-equiv="refresh" content="0;url=evil">`,
		`<link rel="stylesheet" href="evil.css">`,
		`<base href="https://evil.com/">`,
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := utils.SanitizeHTML(input)
			lower := strings.ToLower(got)
			for _, tag := range []string{"<iframe", "<form", "<input", "<object", "<embed", "<button", "<meta", "<link", "<base"} {
				if strings.Contains(lower, tag) {
					t.Errorf("危险标签 %q 未被 SanitizeHTML 剥掉, 输出: %q", tag, got)
				}
			}
		})
	}
}

// TestSanitizeHTML_XSSAttackVectors 集中测试几组常见 XSS payload,
// 这些是真实 prompt injection 后注入到 LLM 输出的形态。
func TestSanitizeHTML_XSSAttackVectors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"img onerror with quoted attr", `<img src="x" onerror="alert('xss')">`},
		{"svg onload", `<svg onload=alert(1)>`},
		{"body onload", `<body onload=alert(1)>`},
		{"iframe javascript src", `<iframe src="javascript:alert(1)"></iframe>`},
		{"a tag javascript href", `<a href="javascript:alert(1)">x</a>`},
		{"data URI base64 payload", `<a href="data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==">x</a>`},
		{"mixed case nested", `<ScRiPt>alert(1)</sCrIpT>`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := utils.SanitizeHTML(c.input)
			lower := strings.ToLower(got)
			// 不应出现可执行 JS 痕迹
			for _, dangerous := range []string{"alert(", "javascript:", "<script", "onerror=", "onload=", "data:text/html"} {
				if strings.Contains(lower, dangerous) {
					t.Errorf("SanitizeHTML 漏过 %q, 输出: %q", dangerous, got)
				}
			}
		})
	}
}
