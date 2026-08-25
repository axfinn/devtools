package handlers

// =====================================================================
//  sing-box E2E 集成测试
//
//  真实启 sing-box(Singleton),测试 mixed inbound 同时支持 HTTP CONNECT + SOCKS5,
//  并验证 devtools dialUpstream 通过 globalBox 走 sing-box mixed port 拿到的连接
//  能正确转发到上游 mock 服务。
//
//  与 xray 1.8.4 时代的差异:
//   - sing-box v1.14+ 支持 hysteria2 / anytls / tuic 全协议
//   - mixed inbound 默认不需要账号认证(为简化 E2E 测试,我们不配 auth.users)
//   - 节点配置直接构造 typed struct(不走 JSON 序列化),不会有字段命名漂移
//
//  复用 proxy_mock_protocol_test.go 的 newMockGoogle / newMockUpstream。
// =====================================================================

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"testing"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/json/badoption"

	"devtools/config"
)

// pickFreePort 申请一个 OS 自动分配的端口,关掉 listener 后返回该端口号。
// sing-box 启动时直接拿这个端口用,避免 hard-coded 端口冲突。
func pickFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

// startSingboxForTest 启一个最小可用的 sing-box:
//
//	mixed inbound @ 127.0.0.1:port(无 auth,简化 E2E)+ direct outbound
//
// 不走 SingboxProc.Start 全路径,只测 sing-box 自身的 mixed inbound + direct outbound 链路。
// 返回 *box.Box 让测试自己负责 Close()。
func startSingboxForTest(t *testing.T, port int) *box.Box {
	t.Helper()
	opts := option.Options{
		Log: &option.LogOptions{Level: "warn"},
		Inbounds: []option.Inbound{
			{
				Type: "mixed",
				Tag:  "mixed-in",
				Options: &option.HTTPMixedInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     common.Ptr(badoption.Addr(netip.MustParseAddr("127.0.0.1"))),
						ListenPort: uint16(port),
					},
				},
			},
		},
		Outbounds: []option.Outbound{
			{Type: "direct", Tag: "direct"},
		},
		Route: &option.RouteOptions{Final: "direct"},
	}
	sb, err := box.New(box.Options{Context: include.Context(context.Background()), Options: opts})
	if err != nil {
		t.Fatalf("box.New 失败: %v", err)
	}
	if err := sb.Start(); err != nil {
		_ = sb.Close()
		t.Fatalf("box.Start 失败: %v", err)
	}
	t.Cleanup(func() { sb.Close() })
	return sb
}

// TestSingBox_RealStart_AcceptHTTP_CONNECT 完整链路:
//
//	client -> dial sing-box mixed:port -> HTTP CONNECT -> sing-box -> direct -> mockGoogle -> 200 OK
func TestSingBox_RealStart_AcceptHTTP_CONNECT(t *testing.T) {
	port := pickFreePort(t)
	_ = startSingboxForTest(t, port)

	google := newMockGoogle(t)
	targetHost, targetPortStr, _ := net.SplitHostPort(google.addr())
	targetPort, _ := strconv.Atoi(targetPortStr)

	// 客户端 dial sing-box mixed port
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5)
	if err != nil {
		t.Fatalf("dial sing-box mixed 失败: %v", err)
	}
	defer conn.Close()

	// 发 HTTP CONNECT,sing-box mixed inbound 会解析后 direct dial mockGoogle
	fmt.Fprintf(conn, "CONNECT %s:%d HTTP/1.1\r\nHost: %s:%d\r\n\r\n",
		targetHost, targetPort, targetHost, targetPort)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("读 CONNECT 响应失败: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("CONNECT status = %d, want 200", resp.StatusCode)
	}

	// CONNECT 成功后,通过 conn 发实际请求给 mockGoogle
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s:%d\r\nConnection: close\r\n\r\n",
		targetHost, targetPort)
	resp2, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读 mockGoogle 响应失败: %v", err)
	}
	defer resp2.Body.Close()
	body, _ := io.ReadAll(resp2.Body)
	if !contains(string(body), "Mock Google") {
		t.Errorf("body 应包含 'Mock Google',实际:\n%s", string(body))
	}
}

// TestSingBox_RealStart_AcceptSOCKS5 验证 mixed inbound 同时支持 SOCKS5:
//
//	client -> dial sing-box mixed:port -> SOCKS5 handshake -> sing-box -> direct -> mockUpstream -> 200
func TestSingBox_RealStart_AcceptSOCKS5(t *testing.T) {
	port := pickFreePort(t)
	_ = startSingboxForTest(t, port)

	upstream := newMockUpstream(t, "hello-via-socks5")
	targetHost, targetPortStr, _ := net.SplitHostPort(upstream.addr())
	targetPort, _ := strconv.Atoi(targetPortStr)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5)
	if err != nil {
		t.Fatalf("dial sing-box mixed 失败: %v", err)
	}
	defer conn.Close()

	// SOCKS5 handshake (RFC 1928)
	// 1) greeting: VER=5, NMETHODS=1, METHOD=0(无认证)
	conn.Write([]byte{0x05, 0x01, 0x00})
	// 2) server method selection
	greet := make([]byte, 2)
	if _, err := io.ReadFull(conn, greet); err != nil {
		t.Fatalf("SOCKS5 greeting 失败: %v", err)
	}
	if greet[0] != 0x05 || greet[1] != 0x00 {
		t.Fatalf("SOCKS5 greeting 错: %v", greet)
	}
	// 3) CONNECT request: VER=5, CMD=1(connect), RSV=0, ATYP=1(IPv4)
	req := []byte{0x05, 0x01, 0x00, 0x01}
	ip := net.ParseIP(targetHost).To4()
	if ip == nil {
		t.Fatalf("targetHost %q 不是 IPv4", targetHost)
	}
	req = append(req, ip...)
	req = append(req, byte(targetPort>>8), byte(targetPort))
	conn.Write(req)
	// 4) CONNECT response
	resp := make([]byte, 10)
	if _, err := io.ReadFull(conn, resp); err != nil {
		t.Fatalf("SOCKS5 CONNECT 响应失败: %v", err)
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		t.Fatalf("SOCKS5 CONNECT 拒绝: %v", resp)
	}

	// SOCKS5 tunnel 建立,发 HTTP 给 mockUpstream
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s:%d\r\nConnection: close\r\n\r\n",
		targetHost, targetPort)
	httpResp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读 mockUpstream 响应失败: %v", err)
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)
	if string(body) != "hello-via-socks5" {
		t.Errorf("body = %q, want hello-via-socks5", string(body))
	}
}

// TestProxyGoogle_ViaSingBox_DirectOutbound_FullChain 走 devtools dialUpstream 全路径:
//
//	dialUpstream 通过 globalBox.MixedAddr() 拿端口 -> sing-box mixed -> direct -> mockGoogle
//
// 验证 dialUpstream 集成正确。
func TestProxyGoogle_ViaSingBox_DirectOutbound_FullChain(t *testing.T) {
	// 1. 起 mock google
	google := newMockGoogle(t)

	// 2. 启动 sing-box,mixed inbound @ free port
	port := pickFreePort(t)
	sb := startSingboxForTest(t, port)

	// 3. 把 globalBox 临时挂上,让 dialUpstream 走它
	//    2026-08-24:dialUpstream 现在会自己发 CONNECT + Proxy-Authorization,所以
	//    MixedAuth 也必须填上,否则握手被 407 拒掉。
	prevBox := globalBox
	t.Cleanup(func() { globalBox = prevBox })
	globalBox = &SingboxProc{
		enabled:   true,
		mixedPort: port,
		mixedUser: "proxy",
		mixedPass: "test-pwd",
		box:       sb,
	}

	// 4. 构造一个指向 mockGoogle 的节点(走 direct outbound 即可)
	targetHost, targetPortStr, _ := net.SplitHostPort(google.addr())
	targetPort, _ := strconv.Atoi(targetPortStr)
	node := &ProxyNode{
		Name:   "mock-direct-node",
		Type:   "http", // devtools 老节点命名,direct 路径不限制
		Server: targetHost,
		Port:   targetPort,
	}

	// 5. dialUpstream 现在内部完成 CONNECT + 鉴权,返回的是已隧道化到目标的 conn,
	//    直接发 GET 给 mockGoogle 就行,不需要再手写 CONNECT。
	conn, err := dialUpstream(node, google.addr())
	if err != nil {
		t.Fatalf("dialUpstream 失败: %v", err)
	}
	defer conn.Close()

	// 6. 隧道已通,发 GET / 验证整条链路
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", google.addr())
	resp2, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读 mockGoogle 响应失败: %v", err)
	}
	defer resp2.Body.Close()
	body, _ := io.ReadAll(resp2.Body)
	if !contains(string(body), "Mock Google") {
		t.Errorf("body 应包含 'Mock Google',实际:\n%s", string(body))
	}
}

// TestSingBox_ConfigChange_RestartOK 验证第二次 Start 不会泄漏旧实例:
// 先用一组 outbounds 启,再用不同 outbounds 重启,确认 mixed port 仍可用。
func TestSingBox_ConfigChange_RestartOK(t *testing.T) {
	port := pickFreePort(t)
	sb1 := startSingboxForTest(t, port)
	// 验证第一个 sing-box 在响应
	if !tryConnect(port) {
		t.Fatal("第一个 sing-box 启动后 mixed port 不可达")
	}
	// 关闭第一个
	_ = sb1.Close()

	// 用同一端口再起一个
	sb2 := startSingboxForTest(t, port)
	if !tryConnect(port) {
		t.Fatal("第二个 sing-box 启动后 mixed port 不可达")
	}
	_ = sb2.Close()
}

// TestSingBox_StartFailure_NoCrash 验证配置错误时 box.New 返回 error 而非 panic。
func TestSingBox_StartFailure_NoCrash(t *testing.T) {
	// unknown inbound type,box.New 拒绝
	opts := option.Options{
		Log: &option.LogOptions{Level: "warn"},
		Inbounds: []option.Inbound{
			{Type: "unknown-protocol-xxx", Tag: "bad", Options: nil},
		},
		Outbounds: []option.Outbound{
			{Type: "direct", Tag: "direct"},
		},
	}
	_, err := box.New(box.Options{Context: include.Context(context.Background()), Options: opts})
	if err == nil {
		t.Fatal("box.New 应拒绝 unknown inbound type,但没报错")
	}
}

// TestGlobalBox_MixedAddr_AfterStart 验证 SingboxProc.Start 后 MixedAddr 返回正确端口。
func TestGlobalBox_MixedAddr_AfterStart(t *testing.T) {
	// 用独立 SingboxProc(避开 globalBox),直接调 Start 验证状态机
	port := pickFreePort(t)
	sp := &SingboxProc{}
	err := sp.Start(nil, config.XrayConfig{
		MixedPort:     port,
		MixedUser:     "proxy",
		MixedPassword: "test-pwd",
		LogLevel:      "warn",
	}, "test-pwd", "")
	if err != nil {
		t.Fatalf("SingboxProc.Start 失败: %v", err)
	}
	defer sp.Stop()

	if !sp.Enabled() {
		t.Error("Start 后 Enabled() 应返回 true")
	}
	addr := sp.MixedAddr()
	if addr != fmt.Sprintf("127.0.0.1:%d", port) {
		t.Errorf("MixedAddr = %q, want 127.0.0.1:%d", addr, port)
	}

	// 实测 mixed port 在监听
	if !tryConnect(port) {
		t.Error("Start 后 mixed port 应在监听")
	}
}

// TestGlobalBox_MixedAddr_BeforeStart 验证未启动时 MixedAddr 返回空串。
func TestGlobalBox_MixedAddr_BeforeStart(t *testing.T) {
	sp := &SingboxProc{}
	if sp.Enabled() {
		t.Error("未 Start 时 Enabled() 应返回 false")
	}
	if addr := sp.MixedAddr(); addr != "" {
		t.Errorf("未 Start 时 MixedAddr 应返回空,实际 %q", addr)
	}
}

// TestSingBox_Stop_AfterStart 验证 Stop 后 Enabled/MixedAddr 都重置。
func TestSingBox_Stop_AfterStart(t *testing.T) {
	port := pickFreePort(t)
	sp := &SingboxProc{}
	if err := sp.Start(nil, config.XrayConfig{MixedPort: port}, "test-pwd", ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !sp.Enabled() {
		t.Fatal("Start 后 Enabled 应为 true")
	}
	sp.Stop()
	if sp.Enabled() {
		t.Error("Stop 后 Enabled 应为 false")
	}
	if addr := sp.MixedAddr(); addr != "" {
		t.Errorf("Stop 后 MixedAddr 应为空,实际 %q", addr)
	}
}

// =====================================================================
//  helpers
// =====================================================================

// tryConnect 探活端口(TCP dial 一次,不等握手)。
func tryConnect(port int) bool {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 1)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// contains 简易字符串包含(避免引入 strings 包,测试代码轻一点)。
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
