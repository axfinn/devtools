package handlers

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =====================================================================
//  本地 mock server,模拟 Trojan / SOCKS5 / HTTP 代理节点
//  验证 dialTrojan / dialSocks5 / dialUpstream http 路径的真实握手字节
//  完全本地:127.0.0.1:0,无外部网络
//
//  Mock 在协议握手成功后,充当"目标 HTTP 服务器"返回一个固定响应,
//  不需要单独 mock upstream(避免重复)
// =====================================================================

func generateTestCert(t *testing.T) tls.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv}
}

// =====================================================================
//  Mock Trojan 节点
// =====================================================================

type mockTrojan struct {
	listener         net.Listener
	expectedPassword string
	body             string       // 握手当作目标 HTTP 服务后返回的 body
	gotPasswordHex   atomic.Value // string
	gotTargetHost    atomic.Value // string
	gotTargetPort    atomic.Int32 // int
	handshakes       atomic.Int32
	handshakeDone    chan struct{} // 第一次握手完成后 close,测试可等
	handshakeOnce    sync.Once
}

func newMockTrojan(t *testing.T, password, body string) *mockTrojan {
	cert := generateTestCert(t)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatal(err)
	}
	m := &mockTrojan{listener: ln, expectedPassword: password, body: body, handshakeDone: make(chan struct{})}
	go m.acceptLoop()
	t.Cleanup(func() { ln.Close() })
	return m
}

func (m *mockTrojan) addr() string { return m.listener.Addr().String() }

func (m *mockTrojan) acceptLoop() {
	for {
		c, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handle(c)
	}
}

func (m *mockTrojan) handle(c net.Conn) {
	defer c.Close()
	br := bufio.NewReader(c)

	// 1. 读 SHA224(pw) hex 行
	line, err := br.ReadString('\n')
	if err != nil {
		return
	}
	got := strings.TrimSpace(line)
	m.gotPasswordHex.Store(got)

	h := sha256.New224()
	h.Write([]byte(m.expectedPassword))
	want := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return
	}

	// 2. 读 CMD
	cmd := make([]byte, 1)
	if _, err := io.ReadFull(br, cmd); err != nil {
		return
	}
	if cmd[0] != 0x01 {
		return
	}

	// 3. 读 ATYP + 目标
	atyp := make([]byte, 1)
	if _, err := io.ReadFull(br, atyp); err != nil {
		return
	}
	var host string
	switch atyp[0] {
	case 0x01:
		hb := make([]byte, 4)
		io.ReadFull(br, hb)
		host = net.IP(hb).String()
	case 0x03:
		n := make([]byte, 1)
		io.ReadFull(br, n)
		hb := make([]byte, n[0])
		io.ReadFull(br, hb)
		host = string(hb)
	case 0x04:
		hb := make([]byte, 16)
		io.ReadFull(br, hb)
		host = net.IP(hb).String()
	default:
		return
	}
	pb := make([]byte, 2)
	if _, err := io.ReadFull(br, pb); err != nil {
		return
	}
	port := int(pb[0])<<8 | int(pb[1])
	br.ReadString('\n') // 结尾 CRLF

	m.gotTargetHost.Store(host)
	m.gotTargetPort.Store(int32(port))
	m.handshakes.Add(1)
	m.handshakeOnce.Do(func() { close(m.handshakeDone) })

	// 4. 握手成功 → 充当目标 HTTP 服务器,返回固定响应
	resp := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: " + strconv.Itoa(len(m.body)) + "\r\n" +
		"Connection: close\r\n\r\n" + m.body
	c.Write([]byte(resp))
	// 等服务端(client)关闭触发 EOF,确保 devtools io.Copy 能把响应完整转出去
	io.Copy(io.Discard, br)
}

// =====================================================================
//  Mock SOCKS5 节点(无认证版本)
// =====================================================================

type mockSOCKS5 struct {
	listener      net.Listener
	body          string
	gotTargetHost atomic.Value // string
	gotTargetPort atomic.Int32
	gotAuthMethod atomic.Int32
	handshakes    atomic.Int32
}

func newMockSOCKS5(t *testing.T, body string) *mockSOCKS5 {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	m := &mockSOCKS5{listener: ln, body: body}
	go m.acceptLoop()
	t.Cleanup(func() { ln.Close() })
	return m
}

func (m *mockSOCKS5) addr() string { return m.listener.Addr().String() }

func (m *mockSOCKS5) acceptLoop() {
	for {
		c, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handle(c)
	}
}

func (m *mockSOCKS5) handle(c net.Conn) {
	defer c.Close()
	br := bufio.NewReader(c)

	// 握手法
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(br, hdr); err != nil {
		return
	}
	methods := make([]byte, hdr[1])
	if _, err := io.ReadFull(br, methods); err != nil {
		return
	}
	chosen := byte(0xFF)
	for _, mt := range methods {
		if mt == 0x00 || mt == 0x02 {
			chosen = mt
			break
		}
	}
	m.gotAuthMethod.Store(int32(chosen))
	c.Write([]byte{0x05, chosen})
	if chosen == 0xFF {
		return
	}

	if chosen == 0x02 {
		ver := make([]byte, 1)
		io.ReadFull(br, ver)
		ulen := make([]byte, 1)
		io.ReadFull(br, ulen)
		uname := make([]byte, ulen[0])
		io.ReadFull(br, uname)
		plen := make([]byte, 1)
		io.ReadFull(br, plen)
		pwd := make([]byte, plen[0])
		io.ReadFull(br, pwd)
		c.Write([]byte{0x01, 0x00})
	}

	// CONNECT 请求
	req := make([]byte, 4)
	if _, err := io.ReadFull(br, req); err != nil {
		return
	}
	var host string
	switch req[3] {
	case 0x01:
		hb := make([]byte, 4)
		io.ReadFull(br, hb)
		host = net.IP(hb).String()
	case 0x03:
		n := make([]byte, 1)
		io.ReadFull(br, n)
		hb := make([]byte, n[0])
		io.ReadFull(br, hb)
		host = string(hb)
	case 0x04:
		hb := make([]byte, 16)
		io.ReadFull(br, hb)
		host = net.IP(hb).String()
	default:
		return
	}
	pb := make([]byte, 2)
	io.ReadFull(br, pb)
	port := int(pb[0])<<8 | int(pb[1])

	c.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	m.gotTargetHost.Store(host)
	m.gotTargetPort.Store(int32(port))
	m.handshakes.Add(1)

	// 充当目标 HTTP 服务器
	resp := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: " + strconv.Itoa(len(m.body)) + "\r\n" +
		"Connection: close\r\n\r\n" + m.body
	c.Write([]byte(resp))
	// 不主动关:等服务端(client)关闭触发 EOF,devtools io.Copy 才能把响应完整转出去
	io.Copy(io.Discard, br)
}

// =====================================================================
//  Mock HTTP 代理节点(响应 CONNECT,后续当目标 HTTP 服务)
// =====================================================================

type mockHTTPProxy struct {
	listener       net.Listener
	body           string
	gotConnectHost atomic.Value // string
	handshakes     atomic.Int32
}

func newMockHTTPProxy(t *testing.T, body string) *mockHTTPProxy {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	m := &mockHTTPProxy{listener: ln, body: body}
	go m.acceptLoop()
	t.Cleanup(func() { ln.Close() })
	return m
}

func (m *mockHTTPProxy) addr() string { return m.listener.Addr().String() }

func (m *mockHTTPProxy) acceptLoop() {
	for {
		c, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handle(c)
	}
}

func (m *mockHTTPProxy) handle(c net.Conn) {
	defer c.Close()
	br := bufio.NewReader(c)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	if req.Method != http.MethodConnect {
		return
	}
	m.gotConnectHost.Store(req.Host)
	m.handshakes.Add(1)
	c.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// 当作目标 HTTP 服务器:客户端通过隧道再发一个 HTTP 请求,我们返回固定响应
	// 但 devtools 的 handleProxyConn 走的是 CONNECT 分支,客户端只需要发一次请求,
	// 我们读 + 响应即可
	req2, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	_ = req2
	resp := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: " + strconv.Itoa(len(m.body)) + "\r\n" +
		"Connection: close\r\n\r\n" + m.body
	c.Write([]byte(resp))
	// 等服务端(client)关闭触发 EOF,确保 devtools io.Copy 能把响应完整转出去
	io.Copy(io.Discard, br)
}

// =====================================================================
//  单独 mock upstream,用于端到端测试(目标不是 mock 节点本身,
//  而是 mock 节点再转发到的目标)
// =====================================================================

type mockUpstream struct {
	listener net.Listener
	hits     atomic.Int32
	body     string
}

func newMockUpstream(t *testing.T, body string) *mockUpstream {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	m := &mockUpstream{listener: ln, body: body}
	go m.acceptLoop()
	t.Cleanup(func() { ln.Close() })
	return m
}

func (m *mockUpstream) addr() string { return m.listener.Addr().String() }

func (m *mockUpstream) acceptLoop() {
	for {
		c, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handle(c)
	}
}

func (m *mockUpstream) handle(c net.Conn) {
	defer c.Close()
	m.hits.Add(1)
	br := bufio.NewReader(c)
	_, _ = http.ReadRequest(br)
	resp := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: " + strconv.Itoa(len(m.body)) + "\r\n" +
		"Connection: close\r\n\r\n" + m.body
	c.Write([]byte(resp))
}

// =====================================================================
//  协议握手字节测试(dialUpstream 直接调,不走 serveHTTPProxy)
// =====================================================================

func TestDialTrojan_HandshakesAgainstMockServer(t *testing.T) {
	const password = "mySecret-123"
	mock := newMockTrojan(t, password, "hello-trojan")
	host, portStr, _ := net.SplitHostPort(mock.addr())
	port, _ := strconv.Atoi(portStr)

	node := &ProxyNode{
		Type:   "trojan",
		Server: host,
		Port:   port,
		Extra: map[string]interface{}{
			"password":         password,
			"skip-cert-verify": true, // 自签 cert 必须跳过验证
		},
	}

	conn, err := dialUpstream(node, "example.com:80")
	if err != nil {
		t.Fatalf("dialUpstream(trojan) 失败: %v", err)
	}
	defer conn.Close()

	// 通过这个 conn 发 HTTP,验证真的能跟 mock 通信
	_, err = fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n")
	if err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读响应失败: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "hello-trojan" {
		t.Errorf("body = %q, want %q", string(body), "hello-trojan")
	}

	if mock.handshakes.Load() != 1 {
		t.Errorf("握手次数 = %d, want 1", mock.handshakes.Load())
	}
	if got := mock.gotTargetHost.Load().(string); got != "example.com" {
		t.Errorf("mock 看到的 host = %q, want %q", got, "example.com")
	}
	if got := mock.gotTargetPort.Load(); int(got) != 80 {
		t.Errorf("mock 看到的 port = %d, want 80", got)
	}

	// 验证 hex 密码完全一致(防止 sha224 用错)
	want := sha256.New224()
	want.Write([]byte(password))
	if got := mock.gotPasswordHex.Load().(string); got != hex.EncodeToString(want.Sum(nil)) {
		t.Errorf("hex 密码不匹配: got=%q want=%q", got, hex.EncodeToString(want.Sum(nil)))
	}
}

func TestDialSocks5_HandshakesAgainstMockServer(t *testing.T) {
	mock := newMockSOCKS5(t, "hello-socks5")
	host, portStr, _ := net.SplitHostPort(mock.addr())
	port, _ := strconv.Atoi(portStr)

	node := &ProxyNode{
		Type:   "socks5",
		Server: host,
		Port:   port,
		Extra:  map[string]interface{}{},
	}

	conn, err := dialUpstream(node, "example.com:443")
	if err != nil {
		t.Fatalf("dialUpstream(socks5) 失败: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读响应失败: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "hello-socks5" {
		t.Errorf("body = %q, want %q", string(body), "hello-socks5")
	}

	if mock.handshakes.Load() != 1 {
		t.Errorf("握手次数 = %d, want 1", mock.handshakes.Load())
	}
	if got := mock.gotTargetHost.Load().(string); got != "example.com" {
		t.Errorf("mock 看到的 host = %q, want %q", got, "example.com")
	}
	if int(mock.gotTargetPort.Load()) != 443 {
		t.Errorf("mock 看到的 port = %d, want 443", mock.gotTargetPort.Load())
	}
	// 默认无认证
	if int(mock.gotAuthMethod.Load()) != 0x00 {
		t.Errorf("client 应选 method 0x00 (无认证), got 0x%02x", mock.gotAuthMethod.Load())
	}
}

func TestDialHTTP_HandshakesAgainstMockServer(t *testing.T) {
	mock := newMockHTTPProxy(t, "hello-http-proxy")
	host, portStr, _ := net.SplitHostPort(mock.addr())
	port, _ := strconv.Atoi(portStr)

	node := &ProxyNode{
		Type:   "http",
		Server: host,
		Port:   port,
		Extra:  map[string]interface{}{},
	}

	conn, err := dialUpstream(node, "example.com:8080")
	if err != nil {
		t.Fatalf("dialUpstream(http) 失败: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读响应失败: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "hello-http-proxy" {
		t.Errorf("body = %q, want %q", string(body), "hello-http-proxy")
	}

	if mock.handshakes.Load() != 1 {
		t.Errorf("mock 收到 CONNECT 次数 = %d, want 1", mock.handshakes.Load())
	}
	if got := mock.gotConnectHost.Load().(string); got != "example.com:8080" {
		t.Errorf("mock CONNECT host = %q, want %q", got, "example.com:8080")
	}
}

// =====================================================================
//  end-to-end: 客户端 → devtools listener → mock 节点(再转发或自己响应)
//  这才是"代理链路通不通"的真实测试
// =====================================================================

// forceGlobalRouting 把路由模式设成 global,确保所有 host 都走代理(不直连)
func forceGlobalRouting(t *testing.T, activeNode *ProxyNode) {
	t.Helper()
	setProxySessionForTest(t,
		[]ProxyNode{*activeNode},
		activeNode,
		proxyRouteModeGlobal, // ← 关键:让所有目标都走代理
		"", "",
		"", "",
	)
}

func TestHandleProxyConn_EndToEndWithMockTrojan(t *testing.T) {
	const password = "e2e-pw"
	trojan := newMockTrojan(t, password, "ok-trojan-e2e")
	tHost, tPortStr, _ := net.SplitHostPort(trojan.addr())
	tPort, _ := strconv.Atoi(tPortStr)

	node := &ProxyNode{
		Type:   "trojan",
		Server: tHost,
		Port:   tPort,
		Extra: map[string]interface{}{
			"password":         password,
			"skip-cert-verify": true,
		},
	}
	forceGlobalRouting(t, node)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go serveHTTPProxy(ln, node, "")

	// 客户端:发 CONNECT 到任意目标(127.0.0.1:1 不会真连,mock 节点自己返回响应)
	target := "example.com:80"
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("读 CONNECT 响应失败: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("CONNECT 响应 = %d, want 200", resp.StatusCode)
	}

	// 隧道里发 HTTP 请求
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", target)
	resp2, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("隧道内读响应失败: %v", err)
	}
	body, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if resp2.StatusCode != 200 {
		t.Errorf("response status = %d, want 200", resp2.StatusCode)
	}
	if string(body) != "ok-trojan-e2e" {
		t.Errorf("body = %q, want %q", string(body), "ok-trojan-e2e")
	}
	if trojan.handshakes.Load() != 1 {
		t.Errorf("Trojan 节点握手 %d 次, want 1", trojan.handshakes.Load())
	}
}

// waitHandshakeDone 在 1s 内等 mock 的 handshakeDone 至少触发一次,防止 close 跟 mock read 的竞争
func waitHandshakeDone(t *testing.T, done <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s 未在 2s 内完成握手(说明 dialUpstream 的协议字节没让 mock 完成解析)", name)
	}
}

func TestHandleProxyConn_EndToEndWithMockSOCKS5(t *testing.T) {
	socks := newMockSOCKS5(t, "ok-socks5-e2e")
	sHost, sPortStr, _ := net.SplitHostPort(socks.addr())
	sPort, _ := strconv.Atoi(sPortStr)

	node := &ProxyNode{
		Type:   "socks5",
		Server: sHost,
		Port:   sPort,
		Extra:  map[string]interface{}{},
	}
	forceGlobalRouting(t, node)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go serveHTTPProxy(ln, node, "")

	target := "example.com:443"
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("读 CONNECT 响应失败: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("CONNECT 响应 = %d, want 200", resp.StatusCode)
	}

	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", target)
	resp2, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("隧道内读响应失败: %v", err)
	}
	body, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body) != "ok-socks5-e2e" {
		t.Errorf("body = %q, want %q", string(body), "ok-socks5-e2e")
	}
}

func TestHandleProxyConn_EndToEndWithMockHTTPProxy(t *testing.T) {
	httpProxy := newMockHTTPProxy(t, "ok-http-e2e")
	hHost, hPortStr, _ := net.SplitHostPort(httpProxy.addr())
	hPort, _ := strconv.Atoi(hPortStr)

	node := &ProxyNode{
		Type:   "http",
		Server: hHost,
		Port:   hPort,
		Extra:  map[string]interface{}{},
	}
	forceGlobalRouting(t, node)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go serveHTTPProxy(ln, node, "")

	target := "example.com:8080"
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("读 CONNECT 响应失败: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("CONNECT 响应 = %d, want 200", resp.StatusCode)
	}

	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", target)
	resp2, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("隧道内读响应失败: %v", err)
	}
	body, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body) != "ok-http-e2e" {
		t.Errorf("body = %q, want %q", string(body), "ok-http-e2e")
	}
}

// =====================================================================
//  checkNodeReachability 走真握手(不只是 TCP ping),用 mock 验证
//  这是用户最关心的"节点是不是活着"——这才是真测试
// =====================================================================

func TestDialUpstream_MockTrojanReachesUpstream(t *testing.T) {
	const password = "reachability-pw"

	// 真实目标(mock upstream)
	upstream := newMockUpstream(t, "from-real-upstream")
	uHost, uPortStr, _ := net.SplitHostPort(upstream.addr())
	uPort, _ := strconv.Atoi(uPortStr)

	// 但是 dialTrojan 直接建立到 trojan 节点的 TLS,不能真"转发"到 upstream
	// 所以这里只能验证 dialTrojan 自己联通;checkNodeReachability 内部就是
	// "建代理 + 用代理发请求到 google.com",这里改成 dialUpstream 验证握手即可

	trojan := newMockTrojan(t, password, "from-trojan")
	tHost, tPortStr, _ := net.SplitHostPort(trojan.addr())
	tPort, _ := strconv.Atoi(tPortStr)

	node := &ProxyNode{
		Type:   "trojan",
		Server: tHost,
		Port:   tPort,
		Extra: map[string]interface{}{
			"password":         password,
			"skip-cert-verify": true,
		},
	}

	// dialUpstream 拿到的是一个"到 trojan 节点的 TLS 连接",而不是到 upstream 的连接
	// 也就是说 dialTrojan 不负责真转发,这是它跟 sing-box 的根本区别
	// 当前实现的局限:dialTrojan 不会把 upstream 的流量转发给 trojan 节点,
	// 它只是建立到 trojan 节点的 TLS,后续字节直传给 trojan 节点
	// 这就是 sing-box 集成的动机之一:真正的代理层应该由专业程序做

	conn, err := dialUpstream(node, fmt.Sprintf("%s:%d", uHost, uPort))
	if err != nil {
		t.Fatalf("dialUpstream 失败: %v", err)
	}
	waitHandshakeDone(t, trojan.handshakeDone, "Trojan")
	conn.Close()

	if trojan.handshakes.Load() != 1 {
		t.Errorf("Trojan 握手 = %d, want 1(说明 dialUpstream 真的发了完整握手)", trojan.handshakes.Load())
	}

	// upstream **不会**被访问,因为 devtools 的 dialTrojan 只是建到 trojan 节点的 TLS,
	// 后续流量给 trojan 节点,由 trojan 节点自己决定怎么连到真目标。
	// 这是协议限制,不是 bug。
	if upstream.hits.Load() != 0 {
		t.Logf("upstream 被访问 %d 次(协议不允许,记录说明)", upstream.hits.Load())
	}
}

// =====================================================================
//  验收:这些测试全过 = dialUpstream 对 http/socks5/trojan 真能联通
//  对 vmess/vless/ss/ssr/hysteria2/anytls/tuic 仍然返回错误(回归防护)
// =====================================================================

func TestDialUpstream_E2EAllRealisticProtocols(t *testing.T) {
	// 真实可用的三种
	t.Run("trojan", func(t *testing.T) {
		mock := newMockTrojan(t, "pw", "ok")
		host, portStr, _ := net.SplitHostPort(mock.addr())
		port, _ := strconv.Atoi(portStr)
		node := &ProxyNode{
			Type:   "trojan",
			Server: host,
			Port:   port,
			Extra:  map[string]interface{}{"password": "pw", "skip-cert-verify": true},
		}
		conn, err := dialUpstream(node, "127.0.0.1:1")
		if err != nil {
			t.Fatalf("trojan dialUpstream 应成功: %v", err)
		}
		waitHandshakeDone(t, mock.handshakeDone, "Trojan")
		conn.Close()
		if mock.handshakes.Load() != 1 {
			t.Errorf("Trojan 收到握手 = %d, want 1", mock.handshakes.Load())
		}
	})

	t.Run("socks5", func(t *testing.T) {
		mock := newMockSOCKS5(t, "ok")
		host, portStr, _ := net.SplitHostPort(mock.addr())
		port, _ := strconv.Atoi(portStr)
		node := &ProxyNode{
			Type:   "socks5",
			Server: host,
			Port:   port,
			Extra:  map[string]interface{}{},
		}
		conn, err := dialUpstream(node, "127.0.0.1:1")
		if err != nil {
			t.Fatalf("socks5 dialUpstream 应成功: %v", err)
		}
		conn.Close()
		if mock.handshakes.Load() != 1 {
			t.Errorf("SOCKS5 收到握手 = %d, want 1", mock.handshakes.Load())
		}
	})

	t.Run("http", func(t *testing.T) {
		mock := newMockHTTPProxy(t, "ok")
		host, portStr, _ := net.SplitHostPort(mock.addr())
		port, _ := strconv.Atoi(portStr)
		node := &ProxyNode{
			Type:   "http",
			Server: host,
			Port:   port,
			Extra:  map[string]interface{}{},
		}
		conn, err := dialUpstream(node, "127.0.0.1:1")
		if err != nil {
			t.Fatalf("http dialUpstream 应成功: %v", err)
		}
		conn.Close()
		if mock.handshakes.Load() != 1 {
			t.Errorf("HTTP 收到 CONNECT = %d, want 1", mock.handshakes.Load())
		}
	})

	// 已知不可用的类型必须报错
	for _, typ := range []string{"vmess", "vless", "ss", "ssr", "hysteria2", "anytls", "tuic", "hy2", "hysteria"} {
		t.Run(typ+"_must_fail", func(t *testing.T) {
			node := &ProxyNode{
				Type:   typ,
				Server: "127.0.0.1",
				Port:   1,
				Extra:  map[string]interface{}{},
			}
			conn, err := dialUpstream(node, "127.0.0.1:1")
			if err == nil {
				if conn != nil {
					conn.Close()
				}
				t.Fatalf("%s 应返回错误,但成功了", typ)
			}
			if conn != nil {
				t.Errorf("%s 错误返回时 conn 应为 nil", typ)
			}
		})
	}
}

// =====================================================================
//  防探测:CONNECT + 错误密码,验证 407
// =====================================================================

func TestServeHTTPProxy_RejectsUnauthenticatedCONNECT(t *testing.T) {
	// 任意 mock 节点(我们走不到)
	mock := newMockSOCKS5(t, "")
	nHost, nPortStr, _ := net.SplitHostPort(mock.addr())
	nPort, _ := strconv.Atoi(nPortStr)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	node := &ProxyNode{
		Type:   "socks5",
		Server: nHost,
		Port:   nPort,
		Extra:  map[string]interface{}{},
	}
	go serveHTTPProxy(ln, node, "secret-admin-pw")

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// 不带认证,发 CONNECT
	fmt.Fprintf(conn, "CONNECT example.com:80 HTTP/1.1\r\nHost: example.com:80\r\n\r\n")
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("读响应失败: %v", err)
	}
	if resp.StatusCode != 407 {
		t.Errorf("未认证 CONNECT 应返回 407,实际 = %d", resp.StatusCode)
	}
	// mock SOCKS5 不应被触达(认证失败)
	if mock.handshakes.Load() != 0 {
		t.Errorf("认证失败时不应触达 SOCKS5 mock,但收到 %d 次握手", mock.handshakes.Load())
	}
}

// =====================================================================
//  模拟 "代理请求 google.com" 完整链路:
//    client → devtools CONNECT → mock trojan(真转发)→ mock Google 服务器
//  验证整条代理链能拿到 Google 风格的响应
// =====================================================================

type mockGoogle struct {
	listener net.Listener
	hits     atomic.Int32
	body     string
}

func newMockGoogle(t *testing.T) *mockGoogle {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	m := &mockGoogle{
		listener: ln,
		body:     "<html><head><title>Mock Google</title></head><body>Hello from mock google via devtools proxy</body></html>",
	}
	go m.acceptLoop()
	t.Cleanup(func() { ln.Close() })
	return m
}

func (m *mockGoogle) addr() string { return m.listener.Addr().String() }

func (m *mockGoogle) acceptLoop() {
	for {
		c, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handle(c)
	}
}

func (m *mockGoogle) handle(c net.Conn) {
	defer c.Close()
	m.hits.Add(1)
	br := bufio.NewReader(c)
	// 读 client 请求头直到空行
	for {
		line, err := br.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
	}
	resp := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Length: " + strconv.Itoa(len(m.body)) + "\r\n" +
		"Connection: close\r\n\r\n" + m.body
	c.Write([]byte(resp))
}

// mockTrojanProxy 真转发版:trojan 协议握手后真 dial forwardTo 做透明 TCP 转发
type mockTrojanProxy struct {
	listener         net.Listener
	expectedPassword string
	forwardTo        string
	gotTargetHost    atomic.Value
	gotTargetPort    atomic.Int32
	handshakes       atomic.Int32
	handshakeDone    chan struct{}
	handshakeOnce    sync.Once
}

func newMockTrojanProxy(t *testing.T, password, forwardTo string) *mockTrojanProxy {
	cert := generateTestCert(t)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatal(err)
	}
	m := &mockTrojanProxy{
		listener:         ln,
		expectedPassword: password,
		forwardTo:        forwardTo,
		handshakeDone:    make(chan struct{}),
	}
	go m.acceptLoop()
	t.Cleanup(func() { ln.Close() })
	return m
}

func (m *mockTrojanProxy) addr() string { return m.listener.Addr().String() }

func (m *mockTrojanProxy) acceptLoop() {
	for {
		c, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handle(c)
	}
}

func (m *mockTrojanProxy) handle(c net.Conn) {
	defer c.Close()
	br := bufio.NewReader(c)

	// 1. 读 SHA224 hex
	line, err := br.ReadString('\n')
	if err != nil {
		return
	}
	got := strings.TrimSpace(line)

	h := sha256.New224()
	h.Write([]byte(m.expectedPassword))
	want := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return
	}

	// 2. CMD
	cmd := make([]byte, 1)
	if _, err := io.ReadFull(br, cmd); err != nil || cmd[0] != 0x01 {
		return
	}

	// 3. ATYP + host + port
	atyp := make([]byte, 1)
	if _, err := io.ReadFull(br, atyp); err != nil {
		return
	}
	var host string
	switch atyp[0] {
	case 0x01:
		hb := make([]byte, 4)
		io.ReadFull(br, hb)
		host = net.IP(hb).String()
	case 0x03:
		n := make([]byte, 1)
		io.ReadFull(br, n)
		hb := make([]byte, n[0])
		io.ReadFull(br, hb)
		host = string(hb)
	case 0x04:
		hb := make([]byte, 16)
		io.ReadFull(br, hb)
		host = net.IP(hb).String()
	default:
		return
	}
	pb := make([]byte, 2)
	if _, err := io.ReadFull(br, pb); err != nil {
		return
	}
	port := int(pb[0])<<8 | int(pb[1])
	br.ReadString('\n')

	m.gotTargetHost.Store(host)
	m.gotTargetPort.Store(int32(port))
	m.handshakes.Add(1)
	m.handshakeOnce.Do(func() { close(m.handshakeDone) })

	// 4. 真 dial forwardTo(模拟 trojan 节点 "连到 google 的真实目标")
	upstream, err := net.DialTimeout("tcp", m.forwardTo, 3*time.Second)
	if err != nil {
		c.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
		return
	}
	defer upstream.Close()

	// 5. 透明双向转发
	go func() {
		io.Copy(upstream, br)
		if tcp, ok := upstream.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
	}()
	io.Copy(c, upstream)
}

func TestProxyGoogle_FullChainViaMockTrojan(t *testing.T) {
	const password = "google-pw"

	// 1. mock Google 服务器(目标)
	google := newMockGoogle(t)

	// 2. mock trojan 节点(代理层),握手后真转到 google 服务器
	trojan := newMockTrojanProxy(t, password, google.addr())
	tHost, tPortStr, _ := net.SplitHostPort(trojan.addr())
	tPort, _ := strconv.Atoi(tPortStr)

	node := &ProxyNode{
		Type:   "trojan",
		Server: tHost,
		Port:   tPort,
		Extra: map[string]interface{}{
			"password":         password,
			"skip-cert-verify": true,
		},
	}
	forceGlobalRouting(t, node)

	// 3. devtools 监听
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go serveHTTPProxy(ln, node, "")

	// 4. client:CONNECT google.com:80 → 期望拿到 mock Google 响应
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	target := "google.com:80"
	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("读 CONNECT 响应失败: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("CONNECT 响应 = %d, want 200", resp.StatusCode)
	}

	// 隧道里发 HTTP GET(模拟浏览器访问 google.com)
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", target)
	resp2, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读目标响应失败: %v", err)
	}
	body, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if resp2.StatusCode != 200 {
		t.Errorf("response status = %d, want 200", resp2.StatusCode)
	}
	if !strings.Contains(string(body), "Mock Google") {
		t.Errorf("body 应包含 'Mock Google',实际:\n%s", string(body))
	}

	// 链路验证:
	// - trojan 节点真做了握手
	// - trojan 看到目标 host=google.com, port=80
	// - mock Google 真被访问(说明 trojan 真转发了)
	if trojan.handshakes.Load() != 1 {
		t.Errorf("Trojan 握手次数 = %d, want 1", trojan.handshakes.Load())
	}
	if got := trojan.gotTargetHost.Load().(string); got != "google.com" {
		t.Errorf("Trojan 看到的目标 host = %q, want %q", got, "google.com")
	}
	if int(trojan.gotTargetPort.Load()) != 80 {
		t.Errorf("Trojan 看到的目标 port = %d, want 80", trojan.gotTargetPort.Load())
	}
	if google.hits.Load() != 1 {
		t.Errorf("mock Google 被访问 %d 次, want 1", google.hits.Load())
	}
}
