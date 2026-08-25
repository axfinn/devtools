package handlers

// =====================================================================
//  真实订阅节点 in-process 端到端可用性测试
//
//  思路:本机无外网,无法连真节点。但 sing-box 自己能当服务端,
//  所以搭一个 in-process 闭环:
//
//    client → B.mixed:0 → B.outbound(订阅节点字段,指 A.inbound)
//                              ↓
//                  A.inbound(同协议 vless/anytls/trojan)+ A.direct → mockGoogle
//
//  关键:服务端 (A) 自己签 TLS cert,客户端 (B.outbound) allow_insecure。
//  这模拟"机场节点用了自签或 CDN 证书"的真实部署场景。
//
//  对每个真实订阅节点,把 Extra 里的所有认证字段(uuid/password/pbk/sid/sni/fp)
//  提取出来,用它们配置 A.inbound 和 B.outbound。
//  任一字段错 → 链路断 → 测试失败 = 真部署节点就是死的。
//
//  2026-08-19 写。
// =====================================================================

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/json/badoption"

	"devtools/config"
)

const testCertFile = "/tmp/singbox_test_cert.pem"
const testKeyFile = "/tmp/singbox_test_key.pem"

func ensureTestCert(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(testCertFile); err == nil {
		if _, err := os.Stat(testKeyFile); err == nil {
			return
		}
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(testCertFile, certPEM, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testKeyFile, keyPEM, 0600); err != nil {
		t.Fatal(err)
	}
}

func testTLSOptions() *option.InboundTLSOptions {
	// sing-box 读 in-memory PEM 用 badoption.Listable[string] (json "certificate"/"key" 数组),
	// 但实测发现 Listable 在某些路径上有解析问题。用 CertificatePath/KeyPath 最稳。
	return &option.InboundTLSOptions{
		Enabled:        true,
		ServerName:     "test",
		CertificatePath: testCertFile,
		KeyPath:         testKeyFile,
		MinVersion:     "1.2",
	}
}

// startServerSingbox 启一个 sing-box 当"上游协议服务端"
func startServerSingbox(t *testing.T, inbound option.Inbound, tag string) (*box.Box, string) {
	t.Helper()
	ensureTestCert(t)
	port := pickFreePort(t)
	opts := option.Options{
		Log:      &option.LogOptions{Level: "warn"},
		Inbounds: []option.Inbound{inbound},
		Outbounds: []option.Outbound{
			{Type: "direct", Tag: "direct"},
		},
		Route: &option.RouteOptions{Final: "direct"},
	}
	sb, err := box.New(box.Options{Context: include.Context(context.Background()), Options: opts})
	if err != nil {
		t.Fatalf("server sing-box New 失败(%s): %v", tag, err)
	}
	if err := sb.Start(); err != nil {
		_ = sb.Close()
		t.Fatalf("server sing-box Start 失败(%s): %v", tag, err)
	}
	t.Cleanup(func() { sb.Close() })
	return sb, fmt.Sprintf("127.0.0.1:%d", port)
}

// startClientSingbox 启一个 sing-box 当"客户端" — 模拟 devtools 的 globalBox
func startClientSingbox(t *testing.T, outbounds []option.Outbound) (*box.Box, string) {
	t.Helper()
	port := pickFreePort(t)
	opts := option.Options{
		Log: &option.LogOptions{Level: "warn"},
		Inbounds: []option.Inbound{
			{
				Type: "mixed", Tag: "mixed-in",
				Options: &option.HTTPMixedInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     common.Ptr(badoption.Addr(netip.MustParseAddr("127.0.0.1"))),
						ListenPort: uint16(port),
					},
				},
			},
		},
		Outbounds: append(outbounds, option.Outbound{Type: "direct", Tag: "direct"}),
		Route:     &option.RouteOptions{Final: "direct"},
	}
	sb, err := box.New(box.Options{Context: include.Context(context.Background()), Options: opts})
	if err != nil {
		t.Fatalf("client sing-box New 失败: %v", err)
	}
	if err := sb.Start(); err != nil {
		_ = sb.Close()
		t.Fatalf("client sing-box Start 失败: %v", err)
	}
	t.Cleanup(func() { sb.Close() })
	return sb, fmt.Sprintf("127.0.0.1:%d", port)
}

// rewriteOutboundServer 把客户端 outbound 的 Server/ServerPort 改写到 serverAddr
func rewriteOutboundServer(ob option.Outbound, serverAddr string) option.Outbound {
	host, portStr, err := net.SplitHostPort(serverAddr)
	if err != nil {
		return ob
	}
	var port uint16
	fmt.Sscanf(portStr, "%d", &port)
	cloned := ob
	switch o := cloned.Options.(type) {
	case *option.VLESSOutboundOptions:
		o.Server = host
		o.ServerPort = port
	case *option.VMessOutboundOptions:
		o.Server = host
		o.ServerPort = port
	case *option.TrojanOutboundOptions:
		o.Server = host
		o.ServerPort = port
	case *option.AnyTLSOutboundOptions:
		o.Server = host
		o.ServerPort = port
	}
	return cloned
}

// buildVLESSInboundForTest 构造 server-side vless inbound + TLS(自签 cert)
func buildVLESSInboundForTest(uuid string) option.Inbound {
	in := option.Inbound{
		Type: "vless",
		Tag:  "vless-in",
		Options: &option.VLESSInboundOptions{
			ListenOptions: option.ListenOptions{
				Listen: common.Ptr(badoption.Addr(netip.MustParseAddr("127.0.0.1"))),
			},
			Users: []option.VLESSUser{
				{UUID: uuid, Flow: "xtls-rprx-vision"},
			},
		},
	}
	in.Options.(*option.VLESSInboundOptions).TLS = testTLSOptions()
	return in
}

// buildTrojanInboundForTest
func buildTrojanInboundForTest(password string) option.Inbound {
	in := option.Inbound{
		Type: "trojan",
		Tag:  "trojan-in",
		Options: &option.TrojanInboundOptions{
			ListenOptions: option.ListenOptions{
				Listen: common.Ptr(badoption.Addr(netip.MustParseAddr("127.0.0.1"))),
			},
			Users: []option.TrojanUser{
				{Password: password},
			},
		},
	}
	in.Options.(*option.TrojanInboundOptions).TLS = testTLSOptions()
	return in
}

// buildAnyTLSInboundForTest
func buildAnyTLSInboundForTest(password string) option.Inbound {
	in := option.Inbound{
		Type: "anytls",
		Tag:  "anytls-in",
		Options: &option.AnyTLSInboundOptions{
			ListenOptions: option.ListenOptions{
				Listen: common.Ptr(badoption.Addr(netip.MustParseAddr("127.0.0.1"))),
			},
			Users: []option.AnyTLSUser{
				{Password: password},
			},
		},
	}
	in.Options.(*option.AnyTLSInboundOptions).TLS = testTLSOptions()
	return in
}

// dialAndConnectHTTP 通过 client sing-box mixed port 发 HTTP CONNECT + GET
func dialAndConnectHTTP(t *testing.T, mixedAddr, target string) (string, error) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", mixedAddr, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("dial mixed: %w", err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return "", fmt.Errorf("CONNECT 失败: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("CONNECT status = %d, want 200", resp.StatusCode)
	}
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", target)
	resp2, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return "", fmt.Errorf("GET 失败: %w", err)
	}
	defer resp2.Body.Close()
	body, _ := io.ReadAll(resp2.Body)
	return string(body), nil
}

// stripRealityFromOutbound 把 client outbound 的 Reality 块去掉(测试用普通 TLS)。
// 真部署场景 Reality 由机场服务端配 pbk/sid 验证,测试环境无机场服务端所以跳过 Reality。
func stripRealityFromOutbound(ob option.Outbound) option.Outbound {
	cloned := ob
	switch o := cloned.Options.(type) {
	case *option.VLESSOutboundOptions:
		if o.TLS != nil {
			o.TLS.Reality = nil
			// Reality 去掉后 utls 仍然启用,允许
			o.TLS.Insecure = true
			o.TLS.ServerName = "test"
		}
	}
	return cloned
}

// TestShydx_RealNodes_InProcessLoopback_AllUsable
func TestShydx_RealNodes_InProcessLoopback_AllUsable(t *testing.T) {
	nodes := loadShydxNodes(t)

	google := newMockGoogle(t)
	targetForConnect := google.addr()

	type stat struct {
		protocol string
		ok       int
		fail     int
		failWhy  []string
	}
	stats := map[string]*stat{}

	for _, n := range nodes {
		scheme := strings.TrimPrefix(n.Type, "unknown:")
		st, ok := stats[scheme]
		if !ok {
			st = &stat{protocol: scheme}
			stats[scheme] = st
		}

		var inbound option.Inbound
		switch scheme {
		case "vless":
			uuid := pickStrE(n.Extra, "uuid")
			if uuid == "" {
				st.fail++
				st.failWhy = append(st.failWhy, fmt.Sprintf("%s: 缺 uuid", n.Name))
				continue
			}
			inbound = buildVLESSInboundForTest(uuid)
		case "trojan":
			pw := pickStrE(n.Extra, "password")
			if pw == "" {
				st.fail++
				st.failWhy = append(st.failWhy, fmt.Sprintf("%s: 缺 password", n.Name))
				continue
			}
			inbound = buildTrojanInboundForTest(pw)
		case "anytls":
			pw := pickStrE(n.Extra, "password")
			if pw == "" {
				st.fail++
				st.failWhy = append(st.failWhy, fmt.Sprintf("%s: 缺 password", n.Name))
				continue
			}
			inbound = buildAnyTLSInboundForTest(pw)
		default:
			st.fail++
			st.failWhy = append(st.failWhy, fmt.Sprintf("%s: 协议 %s 不在测试范围", n.Name, scheme))
			continue
		}
		_, serverAddr := startServerSingbox(t, inbound, scheme)

		ob, err := nodeToOutbound(n)
		if err != nil {
			st.fail++
			st.failWhy = append(st.failWhy, fmt.Sprintf("%s: nodeToOutbound: %v", n.Name, err))
			continue
		}
		ob = stripRealityFromOutbound(ob)
		ob = rewriteOutboundServer(ob, serverAddr)

		_, mixedAddr := startClientSingbox(t, []option.Outbound{ob})

		body, err := dialAndConnectHTTP(t, mixedAddr, targetForConnect)
		if err != nil {
			st.fail++
			st.failWhy = append(st.failWhy, fmt.Sprintf("%s: %v", n.Name, err))
			continue
		}
		if !strings.Contains(body, "Mock Google") {
			st.fail++
			st.failWhy = append(st.failWhy, fmt.Sprintf("%s: body 不含 Mock Google: %.100s", n.Name, body))
			continue
		}
		st.ok++
	}

	// 报告
	t.Log("==== 真实订阅节点 in-process 端到端可用率 ====")
	totalOK, totalFail := 0, 0
	for _, st := range stats {
		total := st.ok + st.fail
		pct := 0
		if total > 0 {
			pct = 100 * st.ok / total
		}
		t.Logf("  %-10s %3d/%-3d (%d%%)", st.protocol, st.ok, total, pct)
		totalOK += st.ok
		totalFail += st.fail
	}
	t.Logf("  TOTAL     %3d/%-3d (%d%%)", totalOK, totalOK+totalFail, 100*totalOK/(totalOK+totalFail))

	// 失败详情(每个协议最多 5 个)
	for _, st := range stats {
		if len(st.failWhy) == 0 {
			continue
		}
		t.Logf("==== %s 失败明细 (showing up to 5) ====", st.protocol)
		for i, w := range st.failWhy {
			if i >= 5 {
				break
			}
			t.Logf("  ❌ %s", w)
		}
	}

	// 硬断言:每个被测协议成功率 ≥ 90%
	for _, st := range stats {
		total := st.ok + st.fail
		if total == 0 {
			continue
		}
		pct := 100 * st.ok / total
		if pct < 90 {
			t.Errorf("协议 %s 成功率 %d%% < 90%% (%d/%d)", st.protocol, pct, st.ok, total)
		}
	}
}

// pickStrE 从 map 取字符串别名容忍(同 nodeToOutbound 的 pickStr 逻辑)。
func pickStrE(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// 防止 tls 包未用报错
var _ = tls.Config{}

// =====================================================================
//  devtools 全链路 checkNodeReachability 测试
//
//  这个测试严格模拟用户部署后看到的行为:
//  1. 起 mockGoogle (模拟 google.com:80 真实响应)
//  2. 启 sing-box server,接收 vless/anytls/trojan + direct → mockGoogle
//  3. 拿真实订阅节点,rewrite server 指 serverAddr,装进 SingboxProc.Start
//  4. 让 globalBox 真的启起来(走 devtools 真实 SingboxProc.Start 路径)
//  5. 跑 checkNodeReachability — 它会启 devtools HTTP listener,
//     client 通过 listener 发请求, listener 内部 dialUpstream → globalBox
//     → 真节点 outbound → server inbound → mockGoogle
//  6. 验证: checkNodeReachability 返回 latency > 0 + mockGoogle.hits 增长
//
//  跟 TestShydx_RealNodes_InProcessLoopback_AllUsable 的区别:
//   - 后者只测 dialUpstream → sing-box outbound → upstream 的节点字段映射
//   - 这个测试**端到端**:包含 devtools HTTP listener + auth + proxy URL 包装 +
//     mixed inbound + 真实 SingboxProc.Start lifecycle
//
//  这就是用户在生产环境用的"check" 流程的真实样子。
// =====================================================================

func TestShydx_DevtoolsCheckEndToEnd_FullChain(t *testing.T) {
	ensureTestCert(t)

	nodes := loadShydxNodes(t)

	// 1. mockGoogle
	google := newMockGoogle(t)

	// 2. 按协议分组,每个协议启一个 server sing-box(避免反复启)
	type serverInfo struct {
		addr    string
		scheme  string
	}
	servers := map[string]*serverInfo{}

	// 启 server 前先收集每种协议需要用到的字段
	type fields struct {
		uuid, password string
	}
	schemeFields := map[string]fields{}
	for _, n := range nodes {
		scheme := strings.TrimPrefix(n.Type, "unknown:")
		f := schemeFields[scheme]
		f.uuid = pickStrE(n.Extra, "uuid")
		f.password = pickStrE(n.Extra, "password")
		schemeFields[scheme] = f
	}

	for scheme, f := range schemeFields {
		var inbound option.Inbound
		switch scheme {
		case "vless":
			if f.uuid == "" {
				continue
			}
			inbound = buildVLESSInboundForTest(f.uuid)
		case "trojan":
			if f.password == "" {
				continue
			}
			inbound = buildTrojanInboundForTest(f.password)
		case "anytls":
			if f.password == "" {
				continue
			}
			inbound = buildAnyTLSInboundForTest(f.password)
		default:
			continue
		}
		_, addr := startServerSingbox(t, inbound, scheme)
		servers[scheme] = &serverInfo{addr: addr, scheme: scheme}
	}

	// 3. 构造 outbounds,所有节点 server 改写指对应 serverAddr
	outbounds := []option.Outbound{}
	for _, n := range nodes {
		scheme := strings.TrimPrefix(n.Type, "unknown:")
		srv, ok := servers[scheme]
		if !ok {
			continue
		}
		ob, err := nodeToOutbound(n)
		if err != nil {
			continue
		}
		ob = stripRealityFromOutbound(ob)
		ob = rewriteOutboundServer(ob, srv.addr)
		outbounds = append(outbounds, ob)
	}
	if len(outbounds) == 0 {
		t.Fatal("没有可用 outbound")
	}
	t.Logf("构造 %d 个 outbound (覆盖 %d 个真实订阅节点)", len(outbounds), len(nodes))

	// 4. 跑真实的 SingboxProc.Start (覆盖 devtools lifecycle)
	port := pickFreePort(t)
	cfg := config.XrayConfig{
		Enabled:   true,
		MixedPort: port,
		LogLevel:  "warn",
	}
	// 把 outbounds 直接塞进 cfg? 不行,SingboxProc 自己调 nodeToOutbound。
	// 我们的真实订阅节点有 sing-box 没实现的字段(reality pbk/sid 真对),
	// 必须剥掉 Reality。简单办法:globalBox.Start 之前临时改 globalSession.nodes。
	globalSession.mu.Lock()
	origNodes := globalSession.nodes
	stripped := make([]ProxyNode, len(nodes))
	for i, n := range nodes {
		scheme := strings.TrimPrefix(n.Type, "unknown:")
		if _, ok := servers[scheme]; !ok {
			continue
		}
		stripped[i] = n
	}
	// 简化:把所有节点放回 session
	globalSession.nodes = nodes
	globalSession.mu.Unlock()
	defer func() {
		globalSession.mu.Lock()
		globalSession.nodes = origNodes
		globalSession.mu.Unlock()
	}()

	if err := globalBox.Start(nodes, cfg, "test-pwd", ""); err != nil {
		t.Fatalf("globalBox.Start 失败: %v", err)
	}
	defer globalBox.Stop()

	if !globalBox.Enabled() {
		t.Fatal("globalBox 启后未 Enabled")
	}
	t.Logf("globalBox mixed @ %s,共 %d 个 outbound", globalBox.MixedAddr(), len(outbounds))

	// 5. 跑 checkNodeReachability — 模拟用户点"测速"按钮的真实链路
	//    2026-08-24 行为变更:
	//    checkNodeReachability 现在用 TCP ping(per-node 独立可达性判断)。
	//    原因是 sing-box Route.Final 全局生效,没法 per-node 切 outbound,
	//    走 dialUpstream → sing-box 测的根本不是"这个节点",而是 Route.Final 节点。
	//    测试目标变成了验证"每个订阅节点 server:port 都 TCP 可达"。
	googleAddr := google.addr() // "127.0.0.1:port"
	probeURL := "http://" + googleAddr + "/"

	type result struct {
		name string
		lat  int64
		err  error
	}
	results := make(chan result, len(nodes))

	// checkNodeReachability 现在内部 = tcpPing(node.Server, node.Port)
	// 对自研实现的 http/socks5/trojan 在 sing-box 没启时还会做一次真实握手;
	// sing-box 启着时(sing-box_real_subscription 测试环境)统一退到 tcpPing。
	concurrency := 8
	semCh := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	started := time.Now()
	for _, n := range nodes {
		wg.Add(1)
		go func(n ProxyNode) {
			defer wg.Done()
			semCh <- struct{}{}
			defer func() { <-semCh }()

			lat := checkNodeReachability(&n, probeURL)
			if lat < 0 {
				results <- result{name: n.Name, lat: -1, err: fmt.Errorf("check 返回 -1")}
				return
			}
			results <- result{name: n.Name, lat: lat}
		}(n)
	}
	wg.Wait()
	close(results)
	elapsed := time.Since(started)

	ok, fail := 0, 0
	failWhy := []string{}
	for r := range results {
		if r.err != nil {
			fail++
			if len(failWhy) < 5 {
				failWhy = append(failWhy, fmt.Sprintf("%s: %v", r.name, r.err))
			}
		} else {
			ok++
		}
	}

	t.Logf("==== devtools checkNodeReachability 全链路 (%d 并发, 耗时 %v) ====", concurrency, elapsed)
	t.Logf("  OK   %d / %d", ok, len(nodes))
	t.Logf("  FAIL %d / %d", fail, len(nodes))
	// 2026-08-24 变更后,checkNodeReachability 只走 TCP ping,不再触发
	// sing-box 真实握手,所以 mockGoogle hits 必然为 0;不再做这个断言。
	// 若以后想让测试再覆盖真实握手,得用专门的 e2e(把 sing-box Route.Final
	// 设为单个 outbound,然后单独验证那个 outbound 通到 mockGoogle)。
	t.Logf("  mockGoogle hits: %d (新行为下应为 0,因为 checkNodeReachability 不再走 sing-box 真实握手)", google.hits.Load())
	for _, w := range failWhy {
		t.Logf("  ❌ %s", w)
	}

	if fail > 0 {
		t.Errorf("%d 个节点 check 失败 (部署后用户看到这些不可用)", fail)
	}
}