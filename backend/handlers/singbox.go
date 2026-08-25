package handlers

// =====================================================================
//  sing-box 嵌入层
//
//  本文件把 sing-box 作为 in-process Go 库嵌入,接管所有代理协议分发:
//    VLESS + Reality + XTLS-Vision / VMess + WS / Trojan / SS / Hy2 /
//    AnyTLS / TUIC / HTTP / SOCKS5 等全协议支持。
//
//  devtools dialUpstream(proxy.go) 走 sing-box 路径时,自己完成 HTTP CONNECT
//  + Proxy-Authorization 握手(dialAndHandshakeSingbox),返回"已隧道化到目标"
//  的 conn 给上层 io.Copy 透传字节 — sing-box mixed inbound 只接受 CONNECT /
//  SOCKS5 握手,看到裸字节会拒。
//
//  Route.Final 默认指向用户选中的"活动节点"tag(activeTag),让 sing-box 把
//  mixed inbound 进来的所有流量都路由到这个节点;Start 调用时由 ProxyHandler
//  把选中节点名传过来。checkNodeReachability(per-node 独立测)走 TCP ping 而
//  不是 sing-box,因为 sing-box 路由全局生效,没法 per-connection 切。
//
//  设计依据:
//  - sing-box v1.14+ 支持 hysteria2 / anytls / tuic 等新协议
//    (xray-core v1.8.4 不支持这些,故从 xray-core 迁移到 sing-box)
//  - Go 1.25.5+ 是 sing-box v1.14+ 的硬要求
//    (devtools Dockerfile 已升级 golang:1.25.5-alpine)
// =====================================================================

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"sync"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/auth"
	"github.com/sagernet/sing/common/json/badoption"

	"devtools/config"
)

// globalBox 全局单例。Start 后 sing-box mixed inbound 在 127.0.0.1:<MixedPort> 监听,
//
//	dialUpstream 内部 dial 这个地址即可让 sing-box 接管协议分发。
//	(方法名 MixedAddr 沿用历史命名,实际是 mixed inbound 端口)
var globalBox = &SingboxProc{}

// SingboxProc sing-box in-process 实例封装。
// 生命周期:Start(nodes, cfg, pwd, activeTag) 生成配置 + 启 sing-box;Stop() 关闭;节点变化时再 Start。
type SingboxProc struct {
	mu        sync.Mutex
	box       *box.Box
	mixedPort int
	mixedUser string
	mixedPass string
	activeTag string // 当前 Route.Final 指向的 outbound tag(用户选中的节点);为空则用第一个节点
	enabled   bool
}

// Enabled 报告 sing-box 是否启用并正在运行。dialUpstream 在写字节给 sing-box 前调用,
//
//	避免在 sing-box 启动失败 / 配置未加载时拿不到端口去 dial 返回 connection refused。
func (s *SingboxProc) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled && s.box != nil && s.mixedPort > 0
}

// MixedAddr 返回 sing-box mixed inbound 监听地址(供 dialUpstream 调用),
//
//	sing-box 未启用或未启动时返回空字符串。
func (s *SingboxProc) MixedAddr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled || s.box == nil || s.mixedPort == 0 {
		return ""
	}
	return fmt.Sprintf("127.0.0.1:%d", s.mixedPort)
}

// MixedAuth 返回当前 sing-box mixed inbound 的 Basic 鉴权凭据(用户名 + 密码),
// 供 dialUpstream 在 HTTP CONNECT / SOCKS5 握手时带 Proxy-Authorization 头。
// sing-box 未启动时 user/pass 都返回空字符串。
func (s *SingboxProc) MixedAuth() (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mixedUser, s.mixedPass
}

// Start 根据最新节点列表 + sing-box 配置,生成 sing-box 配置并启动。
// 已有实例会被优雅关闭再重启 — 节点变化 / 默认节点切换都走这里。
// adminPwd 是 Proxy.AdminPassword,用于 mixed inbound 鉴权(同 devtools 18081 一致)。
// activeTag 是 devtools 当前选中的"活动节点"tag;Route.Final 用它让 sing-box 把
// mixed inbound 进来的所有流量都路由到这个节点。activeTag 为空时退回第一个 outbound,
// 都没有时退到 "direct"(兜底直连,GFW 网络下会失败,用于测速/兜底)。
//
// 2026-08-24 修复:之前 Route.Final 硬编码 "direct",导致 dialUpstream → sing-box
// 出去的流量全部走直连,GFW 网络下 google.com 不可达,/api/proxy/check 显示 0/52
// 可用。改为跟随用户选中的节点,Start 真实代理场景也能跑通。
func (s *SingboxProc) Start(nodes []ProxyNode, cfg config.XrayConfig, adminPwd string, activeTag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 关闭旧实例。sing-box 启动 <500ms, 切换节点无感知。
	if s.box != nil {
		_ = s.box.Close()
		s.box = nil
		s.mixedPort = 0
	}

	// 2. 应用配置默认值
	port := cfg.MixedPort
	if port == 0 {
		port = 18099
	}
	user := cfg.MixedUser
	if user == "" {
		user = "proxy"
	}
	pass := cfg.MixedPassword
	if pass == "" {
		pass = adminPwd
	}
	logLevel := cfg.LogLevel
	if logLevel == "" {
		logLevel = "warn"
	}

	// 3. 生成 outbounds 列表
	outbounds := make([]option.Outbound, 0, len(nodes)+1)
	outboundTags := make([]string, 0, len(nodes)+1)
	for _, n := range nodes {
		ob, err := nodeToOutbound(n)
		if err != nil {
			// 解析失败的节点(SSR 等)跳过,前端仍可见但不做代理
			log.Printf("proxy: sing-box 跳过节点 %s (%s): %v", n.Name, n.Type, err)
			continue
		}
		outbounds = append(outbounds, ob)
		outboundTags = append(outboundTags, ob.Tag)
	}
	// 兜底 direct outbound:让 sing-box 自己直连(国内流量 / 自检)。
	// 参考 sing-box/test/tuic_test.go:direct 类型不设 Options 也能跑。
	outbounds = append(outbounds, option.Outbound{
		Type: "direct",
		Tag:  "direct",
	})
	outboundTags = append(outboundTags, "direct")

	// 4. 解析 Route.Final:优先用户选的 activeTag,其次第一个 outbound,再退 direct。
	finalTag := "direct"
	for _, tag := range outboundTags {
		if tag == activeTag {
			finalTag = activeTag
			break
		}
	}
	if finalTag == "direct" && activeTag != "" {
		// activeTag 提供了但当前节点列表里没有(刚 LoadConfig 完还没解析)
		// 直接用 activeTag,sing-box 启动时会因缺 outbound 报错,日志更直观
		finalTag = activeTag
	}
	if finalTag == "direct" && len(outboundTags) > 1 {
		// 兜底:第一个真实节点(忽略末位 "direct")
		for _, tag := range outboundTags {
			if tag != "direct" {
				finalTag = tag
				break
			}
		}
	}

	// 5. 构造 sing-box 配置(typed struct,不走 JSON 序列化)
	opts := option.Options{
		Log: &option.LogOptions{Level: logLevel},
		Inbounds: []option.Inbound{
			{
				Type: "mixed",
				Tag:  "mixed-in",
				Options: &option.HTTPMixedInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     addrPtr("127.0.0.1"),
						ListenPort: uint16(port),
					},
					Users: []auth.User{
						{Username: user, Password: pass},
					},
				},
			},
		},
		Outbounds: outbounds,
		Route: &option.RouteOptions{
			Final: finalTag,
		},
	}

	// 5. 启动 sing-box
	// box.Options 嵌入 option.Options(无字段名),struct 字面量里写 Options: opts
	// v1.14+ 需要 include.Context() 注册所有协议(vless/vmess/ss/hy2/anytls/tuic 等)的
	// inbound/outbound/endpoint registries,否则 box.New 报 "missing endpoint registry in context"。
	ctx := include.Context(context.Background())
	sb, err := box.New(box.Options{Context: ctx, Options: opts})
	if err != nil {
		return fmt.Errorf("sing-box New 拒绝配置: %w", err)
	}
	if err := sb.Start(); err != nil {
		_ = sb.Close()
		return fmt.Errorf("sing-box Start 失败: %w", err)
	}

	s.box = sb
	s.mixedPort = port
	s.mixedUser = user
	s.mixedPass = pass
	s.activeTag = finalTag
	s.enabled = true

	successOutbounds := 0
	for _, ob := range outbounds {
		if ob.Type != "direct" {
			successOutbounds++
		}
	}
	log.Printf("proxy: sing-box 已启动,mixed inbound @ 127.0.0.1:%d,代理 outbound %d 个,Route.Final=%s",
		port, successOutbounds, finalTag)
	return nil
}

// Stop 优雅关闭 sing-box 实例。devtools 关闭或降级时调用。
func (s *SingboxProc) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.box != nil {
		_ = s.box.Close()
		s.box = nil
	}
	s.mixedPort = 0
	s.mixedUser = ""
	s.mixedPass = ""
	s.activeTag = ""
	s.enabled = false
}

// addrPtr 把字符串 IP 转为 *badoption.Addr。mixed inbound Listen 字段要这个指针类型,
// listenOptions.Listen.Build() 默认回退到 127.0.0.1,但显式写更可读。
// 参考 sing-box/test/tuic_test.go:Listen: common.Ptr(badoption.Addr(netip.IPv4Unspecified()))。
func addrPtr(ip string) *badoption.Addr {
	parsed, err := netip.ParseAddr(ip)
	if err != nil {
		parsed = netip.MustParseAddr("127.0.0.1")
	}
	return common.Ptr(badoption.Addr(parsed))
}

// nodeToOutbound 把单个 ProxyNode 转 sing-box outbound 配置(typed struct)。
// 字段命名遵循 clash YAML 标准(clash 是机场主流订阅格式,字段最稳定),
// pickStr / pickFloat 等辅助函数容忍别名(serverName / sni / servername 等)。
func nodeToOutbound(n ProxyNode) (option.Outbound, error) {
	out := option.Outbound{
		Tag:  n.Name,
		Type: mapProxyType(n.Type),
	}

	switch n.Type {
	case "vless":
		uuid, _ := n.Extra["uuid"].(string)
		if uuid == "" {
			return option.Outbound{}, fmt.Errorf("vless 节点缺 uuid")
		}
		opts := &option.VLESSOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
			UUID: uuid,
		}
		if flow := pickStr(n.Extra, "flow"); flow != "" {
			opts.Flow = flow
		}
		if t := buildV2RayTransport(n.Extra); t != nil {
			opts.Transport = t
		}
		out.Options = opts

	case "vmess":
		uuid, _ := n.Extra["uuid"].(string)
		if uuid == "" {
			return option.Outbound{}, fmt.Errorf("vmess 节点缺 uuid")
		}
		opts := &option.VMessOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
			UUID:     uuid,
			Security: pickStr(n.Extra, "security"),
			AlterId:  int(pickFloat(n.Extra, "alterId", "alter_id")),
		}
		if opts.Security == "" {
			opts.Security = "auto"
		}
		if t := buildV2RayTransport(n.Extra); t != nil {
			opts.Transport = t
		}
		out.Options = opts

	case "trojan":
		password := pickStr(n.Extra, "password")
		if password == "" {
			return option.Outbound{}, fmt.Errorf("trojan 节点缺 password")
		}
		out.Options = &option.TrojanOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
			Password: password,
		}

	case "ss", "shadowsocks":
		method := pickStr(n.Extra, "cipher", "method")
		password := pickStr(n.Extra, "password")
		if method == "" || password == "" {
			return option.Outbound{}, fmt.Errorf("ss 节点缺 method/password")
		}
		out.Options = &option.ShadowsocksOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
			Method:   method,
			Password: password,
		}

	case "hysteria2", "hy2":
		password := pickStr(n.Extra, "password")
		if password == "" {
			return option.Outbound{}, fmt.Errorf("hysteria2 节点缺 password")
		}
		opts := &option.Hysteria2OutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
			Password: password,
		}
		// 混淆:clash 用 obfs + obfs-password,转 sing-box 的 Hysteria2Obfs
		if obfsType := pickStr(n.Extra, "obfs", "obfs-type"); obfsType != "" {
			opts.Obfs = &option.Hysteria2Obfs{
				Type:     obfsType,
				Password: pickStr(n.Extra, "obfs-password", "obfs_password"),
			}
		}
		out.Options = opts

	case "anytls":
		password := pickStr(n.Extra, "password")
		if password == "" {
			return option.Outbound{}, fmt.Errorf("anytls 节点缺 password")
		}
		out.Options = &option.AnyTLSOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
			Password: password,
		}

	case "tuic":
		uuid, _ := n.Extra["uuid"].(string)
		password := pickStr(n.Extra, "password")
		if uuid == "" && password == "" {
			return option.Outbound{}, fmt.Errorf("tuic 节点缺 uuid/password")
		}
		opts := &option.TUICOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
			UUID:              uuid,
			Password:          password,
			CongestionControl: pickStr(n.Extra, "congestion_control", "cc"),
		}
		if opts.CongestionControl == "" {
			opts.CongestionControl = "cubic"
		}
		out.Options = opts

	case "socks", "socks5":
		server := option.SOCKSOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
			Version: "5",
		}
		if u := pickStr(n.Extra, "username"); u != "" {
			server.Username = u
			server.Password = pickStr(n.Extra, "password")
		}
		out.Options = &server

	case "http":
		server := option.HTTPOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     n.Server,
				ServerPort: uint16(n.Port),
			},
		}
		if u := pickStr(n.Extra, "username"); u != "" {
			server.Username = u
			server.Password = pickStr(n.Extra, "password")
		}
		out.Options = &server

	default:
		// unknown:协议 — parseUnknownProxyURL 把协议名存到 Extra["scheme"],
		// 这里递归一次就能复用全部分支
		if scheme, _ := n.Extra["scheme"].(string); scheme != "" {
			cloned := n
			cloned.Type = scheme
			return nodeToOutbound(cloned)
		}
		return option.Outbound{}, fmt.Errorf("sing-box 不支持该协议: %s", n.Type)
	}

	// TLS / Reality 通用块(任何 outbound 都能套,不只是 vless)
	applyTLSOptions(&out, n)

	return out, nil
}

// mapProxyType 把 devtools 内部协议名映射到 sing-box type 常量。
// 当前 1:1,但留这个 hook 防止 ss→shadowsocks 这种 clash 残留命名进入。
func mapProxyType(t string) string {
	switch t {
	case "ss":
		return "shadowsocks"
	default:
		return t
	}
}

// buildV2RayTransport 把 clash 风格的 ws 字段(n.network=ws + path + host)
// 转 sing-box 的 V2RayTransportOptions。返回 nil 表示默认 tcp,无需 transport。
//
// sing-box v1.14+ 改用分协议嵌套选项(无顶层 Path/Headers 字段):
//
//	V2RayTransportOptions { Type, WebsocketOptions{V2RayWebsocketOptions{Path, Headers}} }
//
// 参考 option/v2ray_transport.go。
func buildV2RayTransport(extra map[string]interface{}) *option.V2RayTransportOptions {
	network := pickStr(extra, "network", "type")
	switch network {
	case "ws", "websocket":
		ws := option.V2RayWebsocketOptions{
			Path: pickStr(extra, "path", "ws-path"),
		}
		if host := pickStr(extra, "host", "ws-host"); host != "" {
			ws.Headers = badoption.HTTPHeader{"Host": []string{host}}
		}
		return &option.V2RayTransportOptions{
			Type:             "ws",
			WebsocketOptions: ws,
		}
	default:
		return nil
	}
}

// applyTLSOptions 把 TLS / Reality / uTLS 配置塞进 outbound(若存在)。
// Reality:有 pbk 即走 reality;sni 用 serverName(servername 别名容错);
// fp 用 fingerprint 别名;fingerprint 缺省 chrome(Reality 几乎都配 chrome)。
//
// 2026-08-20 修复:之前只在 sni/hasReality 时才开启 TLS,但 anytls / trojan / hy2 / tuic
// 这些协议即使 clash YAML 没显式给 sni,sing-box 也要求 TLS 必须启用,否则报
// "initialize outbound[0]: TLS required" 节点直接被拒(全死)。
// 现在对需要 TLS 的协议默认开启,ServerName 缺省回落 node host,跳过证书校验由
// clash YAML 的 skip-cert-verify / allowInsecure 显式控制(绝不敢默认 Insecure=true)。
func applyTLSOptions(out *option.Outbound, n ProxyNode) {
	extra := n.Extra
	sni := pickStr(extra, "sni", "servername", "serverName")
	hasReality := pickStr(extra, "pbk") != ""
	requiresTLS := requiresTLSByType(out.Type)

	if sni == "" && !hasReality && !requiresTLS {
		return
	}

	tls := &option.OutboundTLSOptions{
		Enabled:    true,
		ServerName: sni,
	}

	// 跳过证书校验 — 必须显式配置(机场 YAML 一般带 skip-cert-verify: true)
	if v, ok := extra["skip-cert-verify"].(bool); ok && v {
		tls.Insecure = true
	} else if v, ok := extra["allowInsecure"].(bool); ok && v {
		tls.Insecure = true
	}

	// 服务端名兜底:TLS 协议没配 sni 时用节点 host 当 SNI,否则 sing-box 会用空 SNI 触发握手失败
	if tls.ServerName == "" && requiresTLS {
		tls.ServerName = n.Server
	}

	if hasReality {
		// Reality 流程:客户端发 ClientHello,服务端用 ECDH 自签证书"伪装"成 sni 域名
		fp := pickStr(extra, "fp", "fingerprint", "client-fingerprint")
		if fp == "" {
			fp = "chrome"
		}
		tls.Reality = &option.OutboundRealityOptions{
			Enabled:   true,
			PublicKey: pickStr(extra, "pbk", "public-key", "publicKey"),
			ShortID:   pickStr(extra, "sid", "short-id", "shortId"),
		}
		tls.UTLS = &option.OutboundUTLSOptions{
			Enabled:     true,
			Fingerprint: fp,
		}
	} else if sni != "" || requiresTLS {
		// 普通 TLS:有 sni 或协议强制要求 TLS 时配 uTLS 指纹
		// (否则客户端 ALPN/指纹被代理服务器识别)
		fp := pickStr(extra, "fp", "fingerprint", "client-fingerprint")
		if fp != "" {
			tls.UTLS = &option.OutboundUTLSOptions{
				Enabled:     true,
				Fingerprint: fp,
			}
		}
	}

	// 把 tls 塞进对应 outbound(根据 out.Type 取正确的 Container 接口)
	switch o := out.Options.(type) {
	case *option.VLESSOutboundOptions:
		o.TLS = tls
	case *option.VMessOutboundOptions:
		o.TLS = tls
	case *option.TrojanOutboundOptions:
		o.TLS = tls
	case *option.Hysteria2OutboundOptions:
		o.TLS = tls
	case *option.AnyTLSOutboundOptions:
		o.TLS = tls
	case *option.TUICOutboundOptions:
		o.TLS = tls
	case *option.HTTPOutboundOptions:
		o.TLS = tls
	}
}

// requiresTLSByType 报告指定协议是否强制要求 TLS(sing-box outbound 必须配 TLS block,
// 否则启动期就报 "TLS required" — 整个节点被拒)。
// 参照 sing-box v1.14+ protocol 实现:
//   anytls / trojan / hy2 / tuic — 协议层就是 TLS(没 plain 模式);
//   vless — 通常配 reality,但 plain tcp 也支持,这里保守不强制;
//   shadowsocks / socks5 / http — 可选 TLS,不强制;
//   hysteria — legacy,实际未出现在订阅里。
func requiresTLSByType(t string) bool {
	switch t {
	case "anytls", "trojan", "hysteria2", "hy2", "tuic":
		return true
	default:
		return false
	}
}

// pickStr 从 Extra map 取第一个非空字符串。机场订阅字段命名五花八门
// (sni / servername / serverName / SNI),这里容忍最常见的别名,避免
// 单个机场字段差异直接导致全部节点被跳过。
func pickStr(m map[string]interface{}, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// pickFloat 从 Extra map 取第一个数字(int/float64),alterId 等整数字段用。
func pickFloat(m map[string]interface{}, keys ...string) float64 {
	if m == nil {
		return 0
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch n := v.(type) {
			case float64:
				return n
			case int:
				return float64(n)
			case int64:
				return float64(n)
			}
		}
	}
	return 0
}
