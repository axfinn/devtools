package handlers

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// ssRuleMaxSourceSize 限制输入体积,避免有人把整个 GFWList 粘进来把行撑爆。
const ssRuleMaxSourceSize = 512 * 1024

// ssRuleDomainRe 域名白名单:ASCII 字母数字 + 连字符 + 点,且必须至少有一个点。
//
// 白名单而非黑名单是有意为之:本函数的输出会被拼进 ACL/正则并在客户端解析,
// 放任换行、方括号、反斜杠等字符进库等于允许往规则文件里注入任意内容。
var ssRuleDomainRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)

// ssRuleACLHeader 规则文件抬头。
//
// [bypass_all] 表示「默认直连,只有下面 [proxy_list] 里的走代理」。
const ssRuleACLHeader = `# Shadowsocks 分流规则 —— 由 DevTools 生成
# [bypass_all] = 默认直连,仅 [proxy_list] 中的域名走代理

[bypass_all]

[proxy_list]
`

// buildACLRules 把用户输入的域名/URL 列表渲染成 shadowsocks ACL 文件正文。
//
// 每行可以是:裸域名(google.com)、通配域名(*.google.com)、完整 URL
// (https://github.com/xxx?a=1)、IP 或 CIDR。URL 会自动抽出主机名。
//
// 第二个返回值是被跳过的行及原因:贴在页面上的东西必须逐行有交代,
// 静默丢弃会让用户以为规则生效了、实际漏了。
func buildACLRules(source string) (string, []string, error) {
	if len(source) > ssRuleMaxSourceSize {
		return "", nil, fmt.Errorf("输入过长(超过 %d KB),请分批处理", ssRuleMaxSourceSize/1024)
	}

	var proxyLines []string
	var skipped []string
	seen := make(map[string]bool)

	for i, raw := range strings.Split(source, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		host, err := ssRuleExtractHost(line)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("第 %d 行 %s —— %s", i+1, ssRuleTruncate(line, 40), err.Error()))
			continue
		}
		if seen[host] {
			continue // 重复项静默去重,不值得占用一行警告
		}
		seen[host] = true

		proxyLines = append(proxyLines, ssRuleHostToACLLine(host))
	}

	var sb strings.Builder
	sb.WriteString(ssRuleACLHeader)
	if len(proxyLines) == 0 {
		sb.WriteString("# (暂无规则)\n")
	} else {
		for _, l := range proxyLines {
			sb.WriteString(l)
			sb.WriteString("\n")
		}
	}

	return sb.String(), skipped, nil
}

// ssRuleExtractHost 把一行输入规整成主机名 / IP / CIDR。
func ssRuleExtractHost(line string) (string, error) {
	value := line

	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", fmt.Errorf("URL 解析失败")
		}
		// Hostname() 已顺带剥掉端口和 user:pass@ 部分。
		value = parsed.Hostname()
		if value == "" {
			return "", fmt.Errorf("URL 里没有主机名")
		}
	} else {
		// 裸域名/IP 可能带路径、查询串,先截断。
		if idx := strings.IndexAny(value, "/?#"); idx >= 0 {
			value = value[:idx]
		}
		// 带端口的(example.com:8080)剥掉端口;没有端口时 SplitHostPort 报错,
		// 保持原值即可。裸 IPv6(::1)同样会报错,交给下面的 ParseIP 处理。
		if host, _, err := net.SplitHostPort(value); err == nil {
			value = host
		}
	}

	// 统一去掉根域尾点和通配前缀,让 example.com / *.example.com / .example.com
	// 落到同一条规则上(配合 seen 去重)。
	value = strings.TrimSuffix(value, ".")
	value = strings.TrimPrefix(value, "*.")
	value = strings.TrimPrefix(value, ".")

	if value == "" {
		return "", fmt.Errorf("解析后为空")
	}

	// IP 与 CIDR 原样保留,ACL 原生支持这两种形态。
	if _, _, err := net.ParseCIDR(value); err == nil {
		return value, nil
	}
	if net.ParseIP(value) != nil {
		return value, nil
	}

	if !ssRuleDomainRe.MatchString(value) {
		if !ssRuleIsASCII(value) {
			// IDN 必须转 punycode 客户端才认,而静默不生效比报错难查得多。
			return "", fmt.Errorf("含非 ASCII 字符,请先转成 punycode(如 xn----dqo34k.com)")
		}
		return "", fmt.Errorf("不是合法域名")
	}
	return value, nil
}

// ssRuleHostToACLLine 把主机名映射成 ACL 行。
//
// 域名一律输出正则形式 (^|\.)example\.com$ 而不是 shadowsocks-rust 的
// ||example.com 简写:后者在老版本 shadowsocks-libev 上不被识别,
// 正则则是 libev / rust / android 三者通吃的写法。
func ssRuleHostToACLLine(host string) string {
	if _, _, err := net.ParseCIDR(host); err == nil {
		return host
	}
	if net.ParseIP(host) != nil {
		return host
	}
	return `(^|\.)` + regexp.QuoteMeta(host) + `$`
}

func ssRuleIsASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func ssRuleTruncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
