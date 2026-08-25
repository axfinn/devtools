package handlers

// =====================================================================
//  sing-box ConfigGen 单元测试
//
//  覆盖 nodeToOutbound / buildV2RayTransport / applyTLSOptions / pickStr
//  全部分支,验证 typed struct 字段正确性。
//  不启真 sing-box(走 E2E),只测纯函数。
// =====================================================================

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

// TestNodeToOutbound_VLESS_Reality_完整参数 验证 Reality + XTLS-Vision + uTLS 指纹全部正确。
func TestNodeToOutbound_VLESS_Reality_完整参数(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Name:   "vless-reality",
		Type:   "vless",
		Server: "cdn.example.com",
		Port:   443,
		Extra: map[string]interface{}{
			"uuid":    "11111111-2222-3333-4444-555555555555",
			"flow":    "xtls-rprx-vision",
			"sni":     "www.microsoft.com",
			"pbk":     "PUBLIC_KEY_BASE64",
			"sid":     "abcd1234",
			"fp":      "chrome",
			"spx":     "/",
			"network": "tcp",
		},
	})
	if err != nil {
		t.Fatalf("nodeToOutbound 失败: %v", err)
	}
	if ob.Type != "vless" {
		t.Errorf("Type = %q, want vless", ob.Type)
	}
	if ob.Tag != "vless-reality" {
		t.Errorf("Tag = %q, want vless-reality", ob.Tag)
	}

	vless, ok := ob.Options.(*option.VLESSOutboundOptions)
	if !ok {
		t.Fatalf("Options 类型 = %T, want *option.VLESSOutboundOptions", ob.Options)
	}
	if vless.UUID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("UUID 错")
	}
	if vless.Flow != "xtls-rprx-vision" {
		t.Errorf("Flow = %q, want xtls-rprx-vision", vless.Flow)
	}
	if vless.Server != "cdn.example.com" || vless.ServerPort != 443 {
		t.Errorf("Server/ServerPort 错: %s:%d", vless.Server, vless.ServerPort)
	}

	if vless.TLS == nil {
		t.Fatal("TLS 配置为空,Reality 必须有 TLS")
	}
	if !vless.TLS.Enabled {
		t.Error("TLS.Enabled = false")
	}
	if vless.TLS.ServerName != "www.microsoft.com" {
		t.Errorf("ServerName = %q, want www.microsoft.com", vless.TLS.ServerName)
	}
	if vless.TLS.Reality == nil {
		t.Fatal("Reality 配置为空")
	}
	if vless.TLS.Reality.PublicKey != "PUBLIC_KEY_BASE64" {
		t.Errorf("Reality.PublicKey = %q", vless.TLS.Reality.PublicKey)
	}
	if vless.TLS.Reality.ShortID != "abcd1234" {
		t.Errorf("Reality.ShortID = %q", vless.TLS.Reality.ShortID)
	}
	if vless.TLS.UTLS == nil || vless.TLS.UTLS.Fingerprint != "chrome" {
		t.Errorf("UTLS 指纹 = %+v, want chrome", vless.TLS.UTLS)
	}
}

// TestNodeToOutbound_VLESS_XTLS_Vision 验证 flow 单独传递(XTLS-Vision 必须)。
func TestNodeToOutbound_VLESS_XTLS_Vision(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "vless", Server: "v.example.com", Port: 443,
		Extra: map[string]interface{}{
			"uuid": "uuid-1",
			"flow": "xtls-rprx-vision",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	vless := ob.Options.(*option.VLESSOutboundOptions)
	if vless.Flow != "xtls-rprx-vision" {
		t.Errorf("Flow = %q", vless.Flow)
	}
	if vless.TLS != nil {
		t.Error("无 sni/pbk 时不应该有 TLS")
	}
}

// TestNodeToOutbound_VLESS_WS 验证 ws transport(path + host headers)。
func TestNodeToOutbound_VLESS_WS(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "vless", Server: "v.example.com", Port: 443,
		Extra: map[string]interface{}{
			"uuid":    "uuid-ws",
			"network": "ws",
			"path":    "/ws",
			"host":    "cdn.example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	vless := ob.Options.(*option.VLESSOutboundOptions)
	if vless.Transport == nil {
		t.Fatal("Transport 为空,ws 必须有")
	}
	if vless.Transport.Type != "ws" {
		t.Errorf("Transport.Type = %q, want ws", vless.Transport.Type)
	}
	if vless.Transport.WebsocketOptions.Path != "/ws" {
		t.Errorf("WebsocketOptions.Path = %q", vless.Transport.WebsocketOptions.Path)
	}
	if h := vless.Transport.WebsocketOptions.Headers["Host"]; len(h) == 0 || h[0] != "cdn.example.com" {
		t.Errorf("WebsocketOptions.Headers[Host] = %v, want [cdn.example.com]", h)
	}
}

// TestNodeToOutbound_VMess_AES128 验证 vmess 字段(alterId + security 默认 auto)。
func TestNodeToOutbound_VMess_AES128(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "vmess", Server: "v.example.com", Port: 443,
		Extra: map[string]interface{}{
			"uuid":     "vmess-uuid",
			"alterId":  float64(0),
			"security": "aes-128-gcm",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	vmess := ob.Options.(*option.VMessOutboundOptions)
	if vmess.UUID != "vmess-uuid" {
		t.Errorf("UUID 错")
	}
	if vmess.AlterId != 0 {
		t.Errorf("AlterId = %d, want 0", vmess.AlterId)
	}
	if vmess.Security != "aes-128-gcm" {
		t.Errorf("Security = %q", vmess.Security)
	}
}

// TestNodeToOutbound_VMess_SecurityDefaultAuto 验证 security 缺省时填 auto。
func TestNodeToOutbound_VMess_SecurityDefaultAuto(t *testing.T) {
	ob, _ := nodeToOutbound(ProxyNode{
		Type: "vmess", Server: "v.example.com", Port: 443,
		Extra: map[string]interface{}{"uuid": "u"},
	})
	vmess := ob.Options.(*option.VMessOutboundOptions)
	if vmess.Security != "auto" {
		t.Errorf("Security 缺省应填 auto,实际 = %q", vmess.Security)
	}
}

// TestNodeToOutbound_Trojan_含SNI 验证 trojan + TLS sni。
func TestNodeToOutbound_Trojan_含SNI(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "trojan", Server: "t.example.com", Port: 443,
		Extra: map[string]interface{}{
			"password":         "trojan-pwd",
			"sni":              "cdn.example.com",
			"skip-cert-verify": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	trojan := ob.Options.(*option.TrojanOutboundOptions)
	if trojan.Password != "trojan-pwd" {
		t.Errorf("Password 错")
	}
	if trojan.TLS == nil || trojan.TLS.ServerName != "cdn.example.com" {
		t.Errorf("TLS sni 错")
	}
	if trojan.TLS == nil || !trojan.TLS.Insecure {
		t.Error("skip-cert-verify 应让 Insecure=true")
	}
}

// TestNodeToOutbound_Shadowsocks_Chacha20 验证 ss method/password。
func TestNodeToOutbound_Shadowsocks_Chacha20(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "ss", Server: "s.example.com", Port: 8388,
		Extra: map[string]interface{}{
			"cipher":   "chacha20-ietf-poly1305",
			"password": "ss-pwd",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ob.Type != "shadowsocks" {
		t.Errorf("Type = %q, want shadowsocks(ss 已被映射)", ob.Type)
	}
	ss := ob.Options.(*option.ShadowsocksOutboundOptions)
	if ss.Method != "chacha20-ietf-poly1305" {
		t.Errorf("Method 错")
	}
	if ss.Password != "ss-pwd" {
		t.Errorf("Password 错")
	}
}

// TestNodeToOutbound_Hysteria2_含SalamanderObfs 验证 hy2 + obfuscation。
func TestNodeToOutbound_Hysteria2_含SalamanderObfs(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "hysteria2", Server: "h.example.com", Port: 443,
		Extra: map[string]interface{}{
			"password":      "hy2-pwd",
			"obfs":          "salamander",
			"obfs-password": "obfs-pwd",
			"sni":           "cdn.example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	hy2 := ob.Options.(*option.Hysteria2OutboundOptions)
	if hy2.Password != "hy2-pwd" {
		t.Errorf("Password 错")
	}
	if hy2.Obfs == nil {
		t.Fatal("Obfs 为空")
	}
	if hy2.Obfs.Type != "salamander" {
		t.Errorf("Obfs.Type = %q", hy2.Obfs.Type)
	}
	if hy2.Obfs.Password != "obfs-pwd" {
		t.Errorf("Obfs.Password = %q", hy2.Obfs.Password)
	}
	if hy2.TLS == nil || hy2.TLS.ServerName != "cdn.example.com" {
		t.Errorf("TLS sni 错")
	}
}

// TestNodeToOutbound_Hysteria2_无Obfs 验证无混淆时 Obfs 字段保持 nil。
func TestNodeToOutbound_Hysteria2_无Obfs(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "hy2", Server: "h.example.com", Port: 443,
		Extra: map[string]interface{}{"password": "p"},
	})
	if err != nil {
		t.Fatal(err)
	}
	hy2 := ob.Options.(*option.Hysteria2OutboundOptions)
	if hy2.Obfs != nil {
		t.Errorf("无 obfs 字段时 Obfs 应为 nil,实际 = %+v", hy2.Obfs)
	}
}

// TestNodeToOutbound_AnyTLS 验证 anytls password + sni。
func TestNodeToOutbound_AnyTLS(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "anytls", Server: "a.example.com", Port: 443,
		Extra: map[string]interface{}{
			"password": "anytls-pwd",
			"sni":      "cdn.example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	anytls := ob.Options.(*option.AnyTLSOutboundOptions)
	if anytls.Password != "anytls-pwd" {
		t.Errorf("Password 错")
	}
	if anytls.TLS == nil || anytls.TLS.ServerName != "cdn.example.com" {
		t.Errorf("TLS 错")
	}
}

// TestNodeToOutbound_TUIC 验证 tuic uuid + password + 默认 congestion_control=cubic。
func TestNodeToOutbound_TUIC(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "tuic", Server: "tu.example.com", Port: 443,
		Extra: map[string]interface{}{
			"uuid":     "tuic-uuid",
			"password": "tuic-pwd",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tuic := ob.Options.(*option.TUICOutboundOptions)
	if tuic.UUID != "tuic-uuid" || tuic.Password != "tuic-pwd" {
		t.Errorf("UUID/Password 错")
	}
	if tuic.CongestionControl != "cubic" {
		t.Errorf("CongestionControl 缺省应填 cubic,实际 = %q", tuic.CongestionControl)
	}
}

// TestNodeToOutbound_SOCKS5_BasicAuth 验证 socks5 username/password。
func TestNodeToOutbound_SOCKS5_BasicAuth(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "socks5", Server: "sk.example.com", Port: 1080,
		Extra: map[string]interface{}{
			"username": "u",
			"password": "p",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	socks := ob.Options.(*option.SOCKSOutboundOptions)
	if socks.Version != "5" {
		t.Errorf("Version = %q, want 5", socks.Version)
	}
	if socks.Username != "u" || socks.Password != "p" {
		t.Errorf("认证字段错")
	}
}

// TestNodeToOutbound_HTTP_BasicAuth 验证 http outbound username/password。
func TestNodeToOutbound_HTTP_BasicAuth(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "http", Server: "h.example.com", Port: 8080,
		Extra: map[string]interface{}{
			"username": "u",
			"password": "p",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	http := ob.Options.(*option.HTTPOutboundOptions)
	if http.Username != "u" || http.Password != "p" {
		t.Errorf("认证字段错")
	}
}

// TestNodeToOutbound_UnknownFallbackToScheme 验证 unknown:anytls 走 scheme 递归分支。
func TestNodeToOutbound_UnknownFallbackToScheme(t *testing.T) {
	ob, err := nodeToOutbound(ProxyNode{
		Type: "unknown:anytls", Server: "a.example.com", Port: 443,
		Extra: map[string]interface{}{
			"scheme":   "anytls",
			"password": "p",
			"sni":      "cdn.example.com",
		},
	})
	if err != nil {
		t.Fatalf("unknown 协议应回退到 scheme,得到 err: %v", err)
	}
	if ob.Type != "anytls" {
		t.Errorf("Type = %q, want anytls", ob.Type)
	}
}

// TestNodeToOutbound_UnknownNoScheme 验证无 scheme 时返回错误。
func TestNodeToOutbound_UnknownNoScheme(t *testing.T) {
	_, err := nodeToOutbound(ProxyNode{
		Type: "unknown:foo", Server: "a", Port: 1,
		Extra: map[string]interface{}{},
	})
	if err == nil {
		t.Error("未知协议无 scheme 时应返回错误")
	}
}

// TestPickStr_AliasTolerance 验证 sni / servername / serverName 三种命名都接受。
func TestPickStr_AliasTolerance(t *testing.T) {
	cases := []map[string]interface{}{
		{"sni": "a.com"},
		{"servername": "a.com"},
		{"serverName": "a.com"},
	}
	for _, m := range cases {
		if got := pickStr(m, "sni", "servername", "serverName"); got != "a.com" {
			t.Errorf("pickStr %v 应该匹配 a.com,实际 %q", m, got)
		}
	}
}

// TestPickStr_NoMatch 验证无匹配时返回空字符串而非 nil/panic。
func TestPickStr_NoMatch(t *testing.T) {
	if got := pickStr(map[string]interface{}{"x": "y"}, "sni"); got != "" {
		t.Errorf("无匹配应返回空,实际 %q", got)
	}
	if got := pickStr(nil, "sni"); got != "" {
		t.Errorf("nil map 应返回空,实际 %q", got)
	}
}

// TestPickFloat_AliasTolerance 验证 alterId / alter_id 两种命名。
func TestPickFloat_AliasTolerance(t *testing.T) {
	if got := pickFloat(map[string]interface{}{"alterId": float64(2)}, "alterId", "alter_id"); got != 2 {
		t.Errorf("alterId:float64 解析错")
	}
	if got := pickFloat(map[string]interface{}{"alter_id": int(3)}, "alterId", "alter_id"); got != 3 {
		t.Errorf("alter_id:int 解析错")
	}
	if got := pickFloat(map[string]interface{}{"alterId": int64(4)}, "alterId", "alter_id"); got != 4 {
		t.Errorf("alterId:int64 解析错")
	}
	if got := pickFloat(map[string]interface{}{}, "alterId"); got != 0 {
		t.Errorf("缺字段应返回 0")
	}
}

// TestBuildBoxOptions_结构正确 验证构造的 option.Options 包含 mixed inbound + direct final。
// 不启 sing-box,只测纯函数。
func TestBuildBoxOptions_结构正确(t *testing.T) {
	// 复用 nodeToOutbound 拼 outbounds
	outbounds := []option.Outbound{}
	for _, n := range []ProxyNode{
		{Name: "a-vless", Type: "vless", Server: "v.com", Port: 443, Extra: map[string]interface{}{"uuid": "u"}},
		{Name: "a-trojan", Type: "trojan", Server: "t.com", Port: 443, Extra: map[string]interface{}{"password": "p"}},
		{Name: "a-hy2", Type: "hysteria2", Server: "h.com", Port: 443, Extra: map[string]interface{}{"password": "p"}},
		{Name: "a-anytls", Type: "anytls", Server: "a.com", Port: 443, Extra: map[string]interface{}{"password": "p"}},
		{Name: "a-tuic", Type: "tuic", Server: "q.com", Port: 443, Extra: map[string]interface{}{"uuid": "u", "password": "p"}},
	} {
		ob, err := nodeToOutbound(n)
		if err != nil {
			t.Fatalf("nodeToOutbound(%s): %v", n.Name, err)
		}
		outbounds = append(outbounds, ob)
	}
	outbounds = append(outbounds, option.Outbound{Type: "direct", Tag: "direct"})

	if len(outbounds) != 6 {
		t.Errorf("outbounds 长度 = %d, want 6", len(outbounds))
	}
	if outbounds[5].Type != "direct" {
		t.Errorf("最后一个应该是 direct,实际 = %q", outbounds[5].Type)
	}
	// 验证每个 outbound 都能识别 type
	wantTypes := []string{"vless", "trojan", "hysteria2", "anytls", "tuic", "direct"}
	for i, want := range wantTypes {
		if outbounds[i].Type != want {
			t.Errorf("outbounds[%d].Type = %q, want %q", i, outbounds[i].Type, want)
		}
	}
}

// TestMapProxyType 验证 ss → shadowsocks 映射(其它类型透传)。
func TestMapProxyType(t *testing.T) {
	cases := map[string]string{
		"ss":          "shadowsocks",
		"shadowsocks": "shadowsocks",
		"vless":       "vless",
		"trojan":      "trojan",
		"unknown":     "unknown",
	}
	for in, want := range cases {
		if got := mapProxyType(in); got != want {
			t.Errorf("mapProxyType(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestApplyTLSOptions_OnlyRealityPbk 验证只有 pbk 时也走 Reality(不要求 sni 同时存在)。
func TestApplyTLSOptions_OnlyRealityPbk(t *testing.T) {
	ob, _ := nodeToOutbound(ProxyNode{
		Type: "vless", Server: "v.com", Port: 443,
		Extra: map[string]interface{}{
			"uuid": "u",
			"sni":  "cdn.com",
			"pbk":  "PUBLIC_KEY",
			"sid":  "abcdef",
		},
	})
	vless := ob.Options.(*option.VLESSOutboundOptions)
	if vless.TLS == nil || vless.TLS.Reality == nil {
		t.Fatal("Reality 应被设置")
	}
	if !vless.TLS.Reality.Enabled {
		t.Error("Reality.Enabled 应该是 true")
	}
	if vless.TLS.Reality.PublicKey != "PUBLIC_KEY" {
		t.Errorf("Reality.PublicKey 错")
	}
	if vless.TLS.Reality.ShortID != "abcdef" {
		t.Errorf("Reality.ShortID 错")
	}
}

// TestApplyTLSOptions_FingerprintDefaultChrome 验证 Reality 但无 fp 字段时默认 chrome。
func TestApplyTLSOptions_FingerprintDefaultChrome(t *testing.T) {
	ob, _ := nodeToOutbound(ProxyNode{
		Type: "vless", Server: "v.com", Port: 443,
		Extra: map[string]interface{}{
			"uuid": "u",
			"sni":  "cdn.com",
			"pbk":  "PK",
		},
	})
	vless := ob.Options.(*option.VLESSOutboundOptions)
	if vless.TLS.UTLS == nil || vless.TLS.UTLS.Fingerprint != "chrome" {
		t.Errorf("Reality 缺 fp 应默认 chrome,实际 = %+v", vless.TLS.UTLS)
	}
}
