package handlers

import (
	"strings"
	"testing"
)

// ssRuleLines 把生成结果拆成非注释、非空的行,便于做精确的成员断言。
func ssRuleLines(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// ssRuleProxyLines 取出生成结果里的**规则行**:排掉段标记,只留真正要写进
// [proxy_list] 的那些行。
//
// ACL 正文有**两个**段标记:[bypass_all](默认直连)与 [proxy_list](走代理)。
// 断言规则行时两个都要排掉 —— 只排 [proxy_list] 会把 [bypass_all] 当成一条
// 规则,行数断言必然对不上。
func ssRuleProxyLines(content string) []string {
	var out []string
	for _, l := range ssRuleLines(content) {
		if l == "[bypass_all]" || l == "[proxy_list]" {
			continue
		}
		out = append(out, l)
	}
	return out
}

func TestBuildACLRules_Parsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string // 期望出现在 [proxy_list] 里的行
	}{
		{
			name:  "裸域名",
			input: "google.com",
			want:  []string{`(^|\.)google\.com$`},
		},
		{
			name:  "完整 URL 自动抽 host",
			input: "https://www.wikipedia.org/wiki/DevTools?q=1#top",
			want:  []string{`(^|\.)www\.wikipedia\.org$`},
		},
		{
			name:  "URL 带端口和 userinfo 只留 host",
			input: "http://user:pass@example.com:8080/path",
			want:  []string{`(^|\.)example\.com$`},
		},
		{
			name:  "通配前缀被剥掉",
			input: "*.githubusercontent.com",
			want:  []string{`(^|\.)githubusercontent\.com$`},
		},
		{
			name:  "裸域名带路径",
			input: "example.com/path",
			want:  []string{`(^|\.)example\.com$`},
		},
		{
			name:  "根域尾点被剥掉",
			input: "example.com.",
			want:  []string{`(^|\.)example\.com$`},
		},
		{
			name:  "CIDR 原样保留",
			input: "10.0.0.0/8",
			want:  []string{"10.0.0.0/8"},
		},
		{
			name:  "IP 原样保留",
			input: "1.1.1.1",
			want:  []string{"1.1.1.1"},
		},
		{
			name:  "注释和空行被忽略",
			input: "# 这是注释\n\n   \ngoogle.com",
			want:  []string{`(^|\.)google\.com$`},
		},
		{
			name:  "重复项去重",
			input: "a.com\na.com\n*.a.com",
			want:  []string{`(^|\.)a\.com$`},
		},
		{
			name:  "正则元字符被转义",
			input: "a-b.com",
			want:  []string{`(^|\.)a-b\.com$`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, skipped, err := buildACLRules(tt.input)
			if err != nil {
				t.Fatalf("buildACLRules 报错: %v", err)
			}
			if len(skipped) != 0 {
				t.Fatalf("不该有跳过项,得到: %v", skipped)
			}

			// 除了段标记本身,应当恰好是期望的那些行
			got := ssRuleProxyLines(content)

			if len(got) != len(tt.want) {
				t.Fatalf("行数不符,想要 %v,得到 %v\n完整内容:\n%s", tt.want, got, content)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("第 %d 行不符,想要 %q,得到 %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestBuildACLRules_DomainSuffixUsesRegexNotPipe(t *testing.T) {
	// ||example.com 是 shadowsocks-rust 扩展,老版 libev 不认。
	// 输出必须走正则形式,这是三端通吃的写法。
	content, _, err := buildACLRules("example.com")
	if err != nil {
		t.Fatalf("buildACLRules 报错: %v", err)
	}
	if strings.Contains(content, "||example.com") {
		t.Errorf("不应输出 || 简写,它只在 shadowsocks-rust 上生效:\n%s", content)
	}
	if !strings.Contains(content, `(^|\.)example\.com$`) {
		t.Errorf("应输出正则形式,实际:\n%s", content)
	}
}

func TestBuildACLRules_SkipsInvalidLines(t *testing.T) {
	input := strings.Join([]string{
		"good.com",
		"[proxy_all]",      // 试图注入新段
		"中文域名.com",         // IDN 未转 punycode
		"localhost",        // 没有点,不是可分流域名
		"evil.com[bypass]", // 方括号注入
		"another-good.com", // 应当仍然生效
	}, "\n")

	content, skipped, err := buildACLRules(input)
	if err != nil {
		t.Fatalf("buildACLRules 报错: %v", err)
	}

	// 4 行非法,2 行合法
	if len(skipped) != 4 {
		t.Fatalf("期望跳过 4 行,实际跳过 %d 行: %v", len(skipped), skipped)
	}

	got := ssRuleProxyLines(content)
	want := []string{`(^|\.)good\.com$`, `(^|\.)another-good\.com$`}
	if len(got) != len(want) {
		t.Fatalf("合法行数不符,想要 %v,得到 %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 行不符,想要 %q,得到 %q", i, want[i], got[i])
		}
	}

	// 被跳过的行绝不能以任何形式进入产物
	for _, bad := range []string{"[proxy_all]", "中文", "localhost", "bypass]"} {
		if strings.Contains(content, bad) {
			t.Errorf("被拒绝的内容 %q 出现在了产物里:\n%s", bad, content)
		}
	}

	// IDN 的报错要能指导用户怎么改
	var sawPunycodeHint bool
	for _, s := range skipped {
		if strings.Contains(s, "punycode") {
			sawPunycodeHint = true
		}
	}
	if !sawPunycodeHint {
		t.Errorf("IDN 被拒时应提示 punycode 转法,实际跳过项: %v", skipped)
	}
}

func TestBuildACLRules_EmptyInputStillProducesValidFile(t *testing.T) {
	content, skipped, err := buildACLRules("")
	if err != nil {
		t.Fatalf("buildACLRules 报错: %v", err)
	}
	if len(skipped) != 0 {
		t.Errorf("空输入不该有跳过项: %v", skipped)
	}
	if !strings.Contains(content, "[bypass_all]") || !strings.Contains(content, "[proxy_list]") {
		t.Errorf("空输入也应产出结构完整的 ACL:\n%s", content)
	}
}

func TestBuildACLRules_RejectsOversizeInput(t *testing.T) {
	huge := strings.Repeat("a.com\n", ssRuleMaxSourceSize)
	if _, _, err := buildACLRules(huge); err == nil {
		t.Fatal("超长输入应当报错,而不是默默接受")
	}
}
