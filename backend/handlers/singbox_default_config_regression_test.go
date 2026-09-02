package handlers

// =====================================================================
//  默认配置回归测试 — 防止"XrayConfig.Enabled 默认 false"致命 bug 复发
//
//  历史教训:
//    2026-08-20 用户部署后,54 个 shydx 订阅节点全"不可用"。
//    根因:config.DefaultConfig() 里 Proxy.Xray 字段零值,Enabled 默认 false。
//    部署环境未在 config.yaml 显式启用 xray.enabled:true,
//    导致 sing-box 永远不启动,dialUpstream 退到自研 3 协议(http/socks5/trojan),
//    vless/reality/anytls/hy2/tuic/vmess/ss 全报"暂未实现"(38/54 节点直接死)。
//
//  本测试三条护栏:
//    1) DefaultConfig().Proxy.Xray.Enabled 必须是 true(代码层硬约束)
//    2) 无 config.yaml 时 Load 出来仍必须启用(部署开箱即用)
//    3) DefaultConfig() 配 + real anytls 节点 → globalBox.Start + dialUpstream 全链路通
//
//  跑通 = 用户部署后不用动 config.yaml,sing-box 自动接管全部协议。
// =====================================================================

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"devtools/config"
)

// TestDefaultConfig_ProxyXrayEnabled_Regression 护栏 1:
// 任何 commit 都不许把 XrayConfig.Enabled 默认值改回 false。
//
// 历史:2026-08-20 因为该字段零值 false,导致部署后 0/54 节点可用,用户崩溃。
// 现在用测试硬约束:DefaultConfig() 必须启用 xray(因为 devtools 部署环境不一定
// 改 config.yaml,只能靠默认值兜底)。
func TestDefaultConfig_ProxyXrayEnabled_Regression(t *testing.T) {
	cfg := config.DefaultConfig()

	if !cfg.Proxy.Xray.Enabled {
		t.Fatalf("REGRESSION: DefaultConfig().Proxy.Xray.Enabled == false\n" +
			"这是 2026-08-20 部署后 0/54 节点可用的 root cause。\n" +
			"修复方案:把 config/config.go DefaultConfig() 里 Xray: XrayConfig{Enabled: true}。\n" +
			"绝不能改回 false,除非你有 macOS 开发 + 不跑 sing-box in-process 的明确理由。")
	}

	// 顺便验证 mixed_port 默认值,部署里 dialUpstream 依赖它
	if cfg.Proxy.Xray.MixedPort <= 0 || cfg.Proxy.Xray.MixedPort > 65535 {
		t.Errorf("DefaultConfig().Proxy.Xray.MixedPort=%d 非法,应在 1-65535 之间(默认 18099)", cfg.Proxy.Xray.MixedPort)
	}
}

// TestLoadConfig_NoYaml_XrayEnabled 护栏 2:
// 完全不挂 config.yaml(模拟全新部署),Load 后 xray 必须默认启用。
//
// 用户最初发布版本没把 config.yaml 的 xray.enabled 写进去,导致 sing-box 永远不启动。
// 这条测试模拟"全新容器 + 默认 Load"路径,验证生产配置开箱即用。
func TestLoadConfig_NoYaml_XrayEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistent := tmpDir + "/does_not_exist.yaml"
	t.Setenv("CONFIG_PATH", nonexistent)

	cfg, err := config.Load(nonexistent)
	if err != nil {
		t.Fatalf("config.Load(不存在 yaml) 失败: %v", err)
	}

	if !cfg.Proxy.Xray.Enabled {
		t.Fatalf("无 config.yaml 时 Load 出来的 xray.enabled 必须是 true,实际 false。\n" +
			"这是部署后 0/54 节点可用的 root cause,必须在 DefaultConfig() 里硬设 true。")
	}

	if cfg.Proxy.Xray.MixedPort <= 0 {
		t.Errorf("无 config.yaml 时 Load 出来的 MixedPort=%d 非法", cfg.Proxy.Xray.MixedPort)
	}
}

// TestGlobalBox_StartWithDefaultConfig_AnyTLS_FullChain 护栏 3:
// 用 DefaultConfig() 直接 Start,跑一个真实 anytls 节点 → mockGoogle 全链路。
//
// 这是用户最初看不到节点可用的核心场景:
//   - 用户用 shydx 订阅 → 32 个 anytls + 6 个 vless + 15 个 trojan
//   - 用户部署没改 config.yaml
//   - 期望:DefaultConfig() 兜底启用 sing-box,所有 53 个节点都能被 sing-box 接管
//
// 测试只验 anytls(最常见的 shydx 节点类型),其它协议(vless/trojan/hy2)
// 由 TestShydx_DevtoolsCheckEndToEnd_FullChain 覆盖。
func TestGlobalBox_StartWithDefaultConfig_AnyTLS_FullChain(t *testing.T) {
	// 1. DefaultConfig() — 不挂任何 yaml
	cfg := config.DefaultConfig()
	if !cfg.Proxy.Xray.Enabled {
		t.Fatalf("前置失败:DefaultConfig().Proxy.Xray.Enabled 必须 true,这是 2026-08-20 root cause 的护栏")
	}

	// 2. 启 mockGoogle
	mockGoogle := newMockGoogle(t)
	mockGoogleAddr := mockGoogle.addr() // "127.0.0.1:NNNNN"
	t.Logf("mockGoogle @ %s", mockGoogleAddr)

	// mockGoogle 是裸 HTTP 服务,port 就是 :80 语义
	googleHost, googlePort, err := net.SplitHostPort(mockGoogleAddr)
	if err != nil {
		t.Fatalf("解析 mockGoogle 地址失败: %v", err)
	}

	// 3. 搭一个 anytls 服务端,挂到 127.0.0.1:随机端口(模拟"机场 anytls 服务器")
	//    用 in-process sing-box 当服务端,因为本机无外网
	srvPort := pickFreePort(t)
	anytlsInbound := option.Inbound{
		Type: "anytls",
		Tag:  "anytls-in",
		Options: &option.AnyTLSInboundOptions{
			ListenOptions: option.ListenOptions{
				Listen:     addrPtr("127.0.0.1"),
				ListenPort: uint16(srvPort),
			},
			Users: []option.AnyTLSUser{{Name: "u", Password: "test-pwd"}},
		},
	}
	serverSB, err := box.New(box.Options{
		Context: include.Context(context.Background()),
		Options: option.Options{
			Log:      &option.LogOptions{Level: "warn"},
			Inbounds: []option.Inbound{anytlsInbound},
			Outbounds: []option.Outbound{
				{Type: "direct", Tag: "direct"},
			},
			Route: &option.RouteOptions{Final: "direct"},
		},
	})
	if err != nil {
		t.Fatalf("服务端 sing-box New 失败: %v", err)
	}
	if err := serverSB.Start(); err != nil {
		t.Fatalf("服务端 sing-box Start 失败: %v", err)
	}
	defer serverSB.Close()
	t.Logf("anytls 服务端 @ 127.0.0.1:%d", srvPort)

	// 4. 构造一个 anytls 节点(指向我们的 in-process 服务端)
	testNode := ProxyNode{
		Name:   "regression-anytls",
		Type:   "anytls",
		Server: "127.0.0.1",
		Port:   srvPort,
		Extra: map[string]interface{}{
			"password": "test-pwd",
		},
	}

	// 5. ★ 用 DefaultConfig() 启 globalBox — 模拟"用户全新部署"
	adminPwd := "test-admin-pwd"
	// 2026-08-24:SingboxProc.Start 多一个 activeTag 参数(选中的节点 tag,Route.Final 用它)
	if err := globalBox.Start([]ProxyNode{testNode}, cfg.Proxy.Xray, adminPwd, testNode.Name); err != nil {
		t.Fatalf("globalBox.Start(DefaultConfig()) 失败(2026-08-20 root cause): %v", err)
	}
	defer globalBox.Stop()

	// 6. 验证 MixedAddr(核心断言)
	addr := globalBox.MixedAddr()
	if addr == "" {
		t.Fatalf("globalBox.MixedAddr() 返回空(2026-08-20 root cause 的关键症状):\n" +
			"可能原因:1)DefaultConfig().Proxy.Xray.Enabled=false 2)MixedPort=0")
	}
	t.Logf("globalBox.MixedAddr() = %s (DefaultConfig() 兜底生效)", addr)

	// 7. 拨 dialUpstream → 应该走到 globalBox.MixedAddr 分支
	//    2026-08-24:dialUpstream 现在内部完成 CONNECT + 鉴权握手,直接返回
	//    已隧道化到目标的 conn,测试不需要再手写 CONNECT。
	conn, err := dialUpstream(&testNode, googleHost+":"+googlePort)
	if err != nil {
		t.Fatalf("dialUpstream(anytls) 失败: %v\n"+
			"DefaultConfig() 必须启用 sing-box 才能让此调用走 globalBox 路径", err)
	}
	defer conn.Close()
	t.Logf("dialUpstream 成功,目标: %s:%s", googleHost, googlePort)

	// 8. 发 GET,验证拿到 mockGoogle body
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: " + googleHost + "\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatalf("写 GET 失败: %v", err)
	}
	httpResp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读 HTTP 响应失败: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != 200 {
		t.Fatalf("HTTP 状态码=%d,期望 200", httpResp.StatusCode)
	}
	body := make([]byte, 1024)
	n, _ := httpResp.Body.Read(body)
	if !strings.Contains(string(body[:n]), "Mock Google") {
		t.Fatalf("响应体不含 'Mock Google': %s", body[:n])
	}

	t.Logf("✓ DefaultConfig() → globalBox.Start → dialUpstream → mockGoogle 全链路通")
	t.Logf("✓ 用户全新部署 config.yaml 不改 xray,也能让节点可用")
	t.Logf("✓ mockGoogle hits: %d", mockGoogle.hits.Load())
}

// connReader 已不再使用 — http.ReadResponse 接受 bufio.Reader
type _connReaderUnused struct{}
