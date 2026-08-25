package handlers

// =====================================================================
//  E2E 验证:用真实 shydx 订阅节点(从 /tmp/shydx_subscription.b64 读 base64),
//  跑通 sing-box Start → nodeToOutbound 全链路
//
//  数据来源:2026-08-19 从 https://port.shydx.com/link/6174eb1bd2e21475c945c80016ac4ec1
//  拉取保存到 /tmp/shydx_subscription.b64 (base64 原文)。
//  53 个节点 = 6 vless + 15 trojan + 32 anytls。
//
//  关键验证:
//   1. vless Reality 节点 → VLESSOutbound(uuid+flow+pbk+sid+sni+fp 正确)
//   2. anytls 节点(2026-08-19 fix parseUnknownProxyURL 后)→ AnyTLSOutbound
//   3. trojan 节点 → TrojanOutbound
//   4. box.New + Start + Stop 全链路通(verify include.Context fix)
// =====================================================================

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sagernet/sing-box/option"

	"devtools/config"
)

const shydxSubFile = "/tmp/shydx_subscription.b64"

func loadShydxNodes(t *testing.T) []ProxyNode {
	t.Helper()
	data, err := os.ReadFile(shydxSubFile)
	if err != nil {
		t.Skipf("订阅缓存文件不存在(%s),先跑: curl -sL 'https://port.shydx.com/link/6174eb1bd2e21475c945c80016ac4ec1' > %s",
			shydxSubFile, shydxSubFile)
	}
	nodes, err := parseShadowrocketSubscription(string(data))
	if err != nil {
		t.Fatalf("解析订阅失败: %v", err)
	}
	return nodes
}

func TestShydx_SingboxStart_FullChain(t *testing.T) {
	nodes := loadShydxNodes(t)

	// 类型分布
	counts := map[string]int{}
	for _, n := range nodes {
		counts[n.Type]++
	}
	t.Logf("节点类型分布: %v (共 %d 个)", counts, len(nodes))

	port := pickFreePort(t)
	sp := &SingboxProc{}
	cfg := config.XrayConfig{
		Enabled:   true,
		MixedPort: port,
		LogLevel:  "warn",
	}
	start := time.Now()
	if err := sp.Start(nodes, cfg, "test-pwd", ""); err != nil {
		t.Fatalf("SingboxProc.Start 失败: %v", err)
	}
	defer sp.Stop()
	t.Logf("✅ sing-box Start 成功,耗时 %v", time.Since(start))

	if !sp.Enabled() {
		t.Fatal("Start 后 Enabled 应为 true")
	}
	// mixed port 可达
	conn, err := net.DialTimeout("tcp", sp.MixedAddr(), 3*time.Second)
	if err != nil {
		t.Fatalf("dial sing-box mixed 失败: %v", err)
	}
	conn.Close()
	t.Logf("✅ mixed port @ %s 可达", sp.MixedAddr())
}

func TestShydx_AnyTLS_PasswordPreserved_NodeToOutboundOK(t *testing.T) {
	nodes := loadShydxNodes(t)

	var anytlsNode *ProxyNode
	for i := range nodes {
		if nodes[i].Type == "unknown:anytls" {
			anytlsNode = &nodes[i]
			break
		}
	}
	if anytlsNode == nil {
		t.Fatal("订阅里没 anytls 节点")
	}
	ob, err := nodeToOutbound(*anytlsNode)
	if err != nil {
		t.Fatalf("nodeToOutbound(AnyTLS) 失败: %v", err)
	}
	if ob.Type != "anytls" {
		t.Errorf("outbound Type = %q, want anytls", ob.Type)
	}
	opts := ob.Options.(*option.AnyTLSOutboundOptions)
	if opts.Password == "" {
		t.Error("AnyTLSOutboundOptions.Password 为空")
	}
	if opts.Server == "" {
		t.Error("AnyTLSOutboundOptions.Server 为空")
	}
	t.Logf("✅ AnyTLS: %s:%d password_len=%d",
		opts.Server, opts.ServerPort, len(opts.Password))
}

func TestShydx_VLESS_Reality_AllFieldsMapped(t *testing.T) {
	nodes := loadShydxNodes(t)

	var vlessNode *ProxyNode
	for i := range nodes {
		if nodes[i].Type == "vless" {
			vlessNode = &nodes[i]
			break
		}
	}
	if vlessNode == nil {
		t.Fatal("订阅里没 vless 节点")
	}
	ob, err := nodeToOutbound(*vlessNode)
	if err != nil {
		t.Fatalf("nodeToOutbound(VLESS) 失败: %v", err)
	}
	opts := ob.Options.(*option.VLESSOutboundOptions)
	if opts.UUID == "" || opts.Flow != "xtls-rprx-vision" {
		t.Errorf("UUID/Flow 错: uuid=%q flow=%q", opts.UUID, opts.Flow)
	}
	if opts.TLS == nil || opts.TLS.Reality == nil {
		t.Fatal("Reality 配置缺失")
	}
	if opts.TLS.Reality.PublicKey == "" || opts.TLS.Reality.ShortID == "" {
		t.Errorf("Reality pbk/sid 错: pbk=%q sid=%q",
			opts.TLS.Reality.PublicKey, opts.TLS.Reality.ShortID)
	}
	if opts.TLS.UTLS == nil || opts.TLS.UTLS.Fingerprint != "chrome" {
		t.Errorf("UTLS 指纹错: %+v", opts.TLS.UTLS)
	}
	t.Logf("✅ VLESS Reality: UUID=%s Flow=%s PBK_prefix=%s SID=%s SNI=%s",
		opts.UUID, opts.Flow,
		safePrefix(opts.TLS.Reality.PublicKey, 12),
		opts.TLS.Reality.ShortID,
		opts.TLS.ServerName)
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// TestShydx_All53Nodes_NodeToOutboundSuccessRate 真实可用性验证:
// 对订阅里全部 53 个节点逐一跑 nodeToOutbound,统计成功/失败,
// 失败节点打印 Type + Server:Port + Extra 关键字段,便于排查。
//
// 这是用户最关心的"这些节点到底能不能用"的真实指标。
// 成功 = sing-box 配置能生成;失败 = 字段缺失/类型错,sing-box 启动会跳过该节点。
func TestShydx_All53Nodes_NodeToOutboundSuccessRate(t *testing.T) {
	nodes := loadShydxNodes(t)
	t.Logf("订阅解析: %d 节点", len(nodes))

	type fail struct {
		Name    string
		Type    string
		Server  string
		Port    int
		Err     string
		Missing string
	}
	var fails []fail
	var byTypeSucc, byTypeTotal = map[string]int{}, map[string]int{}

	for _, n := range nodes {
		byTypeTotal[n.Type]++
		ob, err := nodeToOutbound(n)
		if err != nil {
			missing := guessMissingField(n)
			fails = append(fails, fail{
				Name:    n.Name,
				Type:    n.Type,
				Server:  n.Server,
				Port:    n.Port,
				Err:     err.Error(),
				Missing: missing,
			})
			continue
		}
		_ = ob // sing-box 接受了配置
		byTypeSucc[n.Type]++
	}

	t.Log("==== 节点 → sing-box outbound 转换成功率 ====")
	for typ, total := range byTypeTotal {
		succ := byTypeSucc[typ]
		pct := 100 * succ / total
		t.Logf("  %-15s %3d/%-3d (%d%%)", typ, succ, total, pct)
	}

	if len(fails) > 0 {
		t.Logf("==== %d 个节点转换失败 ====", len(fails))
		for _, f := range fails {
			t.Logf("❌ [%s] %s | %s:%d | %s (missing=%q)",
				f.Type, f.Name, f.Server, f.Port, f.Err, f.Missing)
		}
	} else {
		t.Logf("✅ 全部 %d 个节点转换成功", len(nodes))
	}

	// 硬断言:任何协议成功率应 ≥ 95%(允许 1-2 个机场配置畸形)
	for typ, total := range byTypeTotal {
		succ := byTypeSucc[typ]
		if total >= 5 && succ*100 < total*95 {
			t.Errorf("协议 %s 成功率太低: %d/%d (%d%% < 95%%)",
				typ, succ, total, 100*succ/total)
		}
	}
}

// guessMissingField 从节点的 Extra 推断哪个关键字段缺失。
// 用于让"❌ 节点"日志一眼看出问题,不靠读源码。
func guessMissingField(n ProxyNode) string {
	switch n.Type {
	case "vless":
		if _, ok := n.Extra["uuid"]; !ok {
			return "uuid"
		}
		if _, ok := n.Extra["pbk"]; !ok {
			return "pbk(reality)"
		}
		if _, ok := n.Extra["sid"]; !ok {
			return "sid(reality)"
		}
		if _, ok := n.Extra["sni"]; !ok {
			return "sni"
		}
		return ""
	case "vmess":
		if _, ok := n.Extra["uuid"]; !ok {
			return "uuid"
		}
		return ""
	case "trojan":
		if _, ok := n.Extra["password"]; !ok {
			return "password"
		}
		return ""
	case "ss", "shadowsocks":
		if _, ok := n.Extra["cipher"]; !ok {
			if _, ok2 := n.Extra["method"]; !ok2 {
				return "cipher/method"
			}
		}
		if _, ok := n.Extra["password"]; !ok {
			return "password"
		}
		return ""
	case "hysteria2", "hy2":
		if _, ok := n.Extra["password"]; !ok {
			return "password"
		}
		return ""
	case "anytls":
		if _, ok := n.Extra["password"]; !ok {
			return "password"
		}
		return ""
	case "tuic":
		if _, ok := n.Extra["uuid"]; !ok && n.Extra["password"] == nil {
			return "uuid+password"
		}
		return ""
	default:
		if strings.HasPrefix(n.Type, "unknown:") {
			return "scheme=" + n.Type + " (parseUnknownProxyURL fix not applied?)"
		}
		return ""
	}
}
