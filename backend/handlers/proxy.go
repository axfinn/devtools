package handlers

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// ProxyNode 代理节点
type ProxyNode struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Server   string                 `json:"server"`
	Port     int                    `json:"port"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
	Latency  int64                  `json:"latency"` // ms，-1=失败
}

// proxySession 当前会话（内存单例）
type proxySession struct {
	mu       sync.RWMutex
	nodes    []ProxyNode
	active   *ProxyNode
	listener net.Listener // HTTP CONNECT 代理监听
}

var globalSession = &proxySession{}

// ProxyHandler 科学上网处理器
type ProxyHandler struct {
	adminPassword string
}

func NewProxyHandler(cfg *config.Config) *ProxyHandler {
	return &ProxyHandler{adminPassword: cfg.Proxy.AdminPassword}
}

func (h *ProxyHandler) checkAdmin(password string) bool {
	return h.adminPassword != "" && password == h.adminPassword
}

// clashConfig Clash YAML 顶层结构（只取 proxies）
type clashConfig struct {
	Proxies []map[string]interface{} `yaml:"proxies"`
}

// parseClashYAML 解析 Clash YAML 文本，返回节点列表
func parseClashYAML(data string) ([]ProxyNode, error) {
	var cfg clashConfig
	if err := yaml.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, err
	}
	nodes := make([]ProxyNode, 0, len(cfg.Proxies))
	for _, p := range cfg.Proxies {
		node := ProxyNode{Extra: make(map[string]interface{})}
		if v, ok := p["name"].(string); ok {
			node.Name = v
		}
		if v, ok := p["type"].(string); ok {
			node.Type = strings.ToLower(v)
		}
		if v, ok := p["server"].(string); ok {
			node.Server = v
		}
		switch port := p["port"].(type) {
		case int:
			node.Port = port
		case float64:
			node.Port = int(port)
		}
		// 保存其余字段
		for k, v := range p {
			if k != "name" && k != "type" && k != "server" && k != "port" {
				node.Extra[k] = v
			}
		}
		node.Latency = -1
		if node.Name != "" && node.Server != "" && node.Port > 0 {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// fetchSubscription 下载订阅内容（支持 Base64 和 YAML）
func fetchSubscription(rawURL string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(body))
	// 尝试 Base64 解码
	if decoded, err := base64.StdEncoding.DecodeString(text); err == nil {
		return string(decoded), nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(text); err == nil {
		return string(decoded), nil
	}
	return text, nil
}

// LoadConfig POST /api/proxy/config
func (h *ProxyHandler) LoadConfig(c *gin.Context) {
	var req struct {
		AdminPassword string `json:"admin_password"`
		SourceURL     string `json:"source_url"`
		YAMLContent   string `json:"yaml_content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	if !h.checkAdmin(req.AdminPassword) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}

	var yamlText string
	if req.SourceURL != "" {
		text, err := fetchSubscription(req.SourceURL)
		if err != nil {
			c.JSON(500, gin.H{"error": "下载订阅失败: " + err.Error()})
			return
		}
		yamlText = text
	} else {
		yamlText = req.YAMLContent
	}

	nodes, err := parseClashYAML(yamlText)
	if err != nil {
		c.JSON(400, gin.H{"error": "解析配置失败: " + err.Error()})
		return
	}
	if len(nodes) == 0 {
		c.JSON(400, gin.H{"error": "未找到任何节点"})
		return
	}

	globalSession.mu.Lock()
	globalSession.nodes = nodes
	globalSession.mu.Unlock()

	c.JSON(200, gin.H{"nodes": nodes, "count": len(nodes)})
}

// SpeedTest POST /api/proxy/speedtest
func (h *ProxyHandler) SpeedTest(c *gin.Context) {
	var req struct {
		AdminPassword string `json:"admin_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	if !h.checkAdmin(req.AdminPassword) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}

	globalSession.mu.RLock()
	nodes := make([]ProxyNode, len(globalSession.nodes))
	copy(nodes, globalSession.nodes)
	globalSession.mu.RUnlock()

	if len(nodes) == 0 {
		c.JSON(400, gin.H{"error": "请先加载配置"})
		return
	}

	var wg sync.WaitGroup
	results := make([]ProxyNode, len(nodes))
	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, n ProxyNode) {
			defer wg.Done()
			n.Latency = tcpPing(n.Server, n.Port)
			results[idx] = n
		}(i, node)
	}
	wg.Wait()

	globalSession.mu.Lock()
	globalSession.nodes = results
	globalSession.mu.Unlock()

	c.JSON(200, gin.H{"results": results})
}

// tcpPing 测量 TCP 连接延迟（ms），失败返回 -1
func tcpPing(host string, port int) int64 {
	addr := fmt.Sprintf("%s:%d", host, port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return -1
	}
	conn.Close()
	return time.Since(start).Milliseconds()
}

// Start POST /api/proxy/start
func (h *ProxyHandler) Start(c *gin.Context) {
	var req struct {
		AdminPassword string `json:"admin_password"`
		NodeName      string `json:"node_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	if !h.checkAdmin(req.AdminPassword) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}

	globalSession.mu.RLock()
	var target *ProxyNode
	for i := range globalSession.nodes {
		if globalSession.nodes[i].Name == req.NodeName {
			n := globalSession.nodes[i]
			target = &n
			break
		}
	}
	globalSession.mu.RUnlock()

	if target == nil {
		c.JSON(404, gin.H{"error": "节点不存在"})
		return
	}

	// 停止旧代理
	globalSession.mu.Lock()
	if globalSession.listener != nil {
		globalSession.listener.Close()
		globalSession.listener = nil
	}
	globalSession.mu.Unlock()

	// 启动 HTTP CONNECT 代理
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		c.JSON(500, gin.H{"error": "监听失败: " + err.Error()})
		return
	}

	globalSession.mu.Lock()
	globalSession.listener = ln
	globalSession.active = target
	globalSession.mu.Unlock()

	go serveHTTPProxy(ln, target)

	port := ln.Addr().(*net.TCPAddr).Port
	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	c.JSON(200, gin.H{
		"http_port": port,
		"proxy_url": proxyURL,
		"node":      target.Name,
	})
}

// Stop POST /api/proxy/stop
func (h *ProxyHandler) Stop(c *gin.Context) {
	var req struct {
		AdminPassword string `json:"admin_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	if !h.checkAdmin(req.AdminPassword) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}

	globalSession.mu.Lock()
	if globalSession.listener != nil {
		globalSession.listener.Close()
		globalSession.listener = nil
	}
	globalSession.active = nil
	globalSession.mu.Unlock()

	c.JSON(200, gin.H{"ok": true})
}

// Status GET /api/proxy/status
func (h *ProxyHandler) Status(c *gin.Context) {
	password := c.Query("admin_password")
	if !h.checkAdmin(password) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}

	globalSession.mu.RLock()
	defer globalSession.mu.RUnlock()

	if globalSession.active == nil || globalSession.listener == nil {
		c.JSON(200, gin.H{"running": false})
		return
	}
	port := globalSession.listener.Addr().(*net.TCPAddr).Port
	c.JSON(200, gin.H{
		"running":   true,
		"node":      globalSession.active.Name,
		"http_port": port,
		"proxy_url": fmt.Sprintf("http://127.0.0.1:%d", port),
	})
}

// Fetch GET /api/proxy/fetch?url=xxx&admin_password=xxx
// 通过当前代理抓取 URL，返回 HTML（替换相对路径）
func (h *ProxyHandler) Fetch(c *gin.Context) {
	password := c.Query("admin_password")
	if !h.checkAdmin(password) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}

	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(400, gin.H{"error": "缺少 url 参数"})
		return
	}
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	globalSession.mu.RLock()
	ln := globalSession.listener
	globalSession.mu.RUnlock()

	var transport http.RoundTripper
	if ln != nil {
		port := ln.Addr().(*net.TCPAddr).Port
		proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
		transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	} else {
		transport = http.DefaultTransport
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", targetURL, nil)
	if err != nil {
		c.JSON(400, gin.H{"error": "URL 无效: " + err.Error()})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DevTools/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(502, gin.H{"error": "请求失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		c.JSON(502, gin.H{"error": "读取响应失败"})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		// 替换相对路径为绝对路径
		html := rewriteHTML(string(body), targetURL)
		c.Data(200, "text/html; charset=utf-8", []byte(html))
	} else {
		c.Data(resp.StatusCode, contentType, body)
	}
}

// rewriteHTML 将 HTML 中的相对路径替换为绝对路径
func rewriteHTML(html, baseURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return html
	}
	// 简单替换 href="/ src="/ action="/
	for _, attr := range []string{`href="`, `src="`, `action="`} {
		html = rewriteAttr(html, attr, base)
	}
	return html
}

func rewriteAttr(html, attr string, base *url.URL) string {
	var sb strings.Builder
	remaining := html
	for {
		idx := strings.Index(remaining, attr)
		if idx < 0 {
			sb.WriteString(remaining)
			break
		}
		sb.WriteString(remaining[:idx+len(attr)])
		rest := remaining[idx+len(attr):]
		end := strings.IndexByte(rest, '"')
		if end < 0 {
			sb.WriteString(rest)
			break
		}
		link := rest[:end]
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") && !strings.HasPrefix(link, "//") && !strings.HasPrefix(link, "javascript:") && !strings.HasPrefix(link, "data:") && !strings.HasPrefix(link, "#") {
			ref, err := url.Parse(link)
			if err == nil {
				link = base.ResolveReference(ref).String()
			}
		}
		sb.WriteString(link)
		remaining = rest[end:]
	}
	return sb.String()
}

// serveHTTPProxy 运行一个简单的 HTTP CONNECT 代理，转发到目标节点
// 支持 http/socks5 类型节点的上游代理，其他类型直连
func serveHTTPProxy(ln net.Listener, node *ProxyNode) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleProxyConn(conn, node)
	}
}

func handleProxyConn(clientConn net.Conn, node *ProxyNode) {
	defer clientConn.Close()

	br := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	if req.Method == http.MethodConnect {
		// HTTPS CONNECT 隧道
		host := req.Host
		upstream, err := dialUpstream(node, host)
		if err != nil {
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}
		clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		go io.Copy(upstream, br)
		io.Copy(clientConn, upstream)
		upstream.Close()
	} else {
		// HTTP 普通请求
		host := req.Host
		if !strings.Contains(host, ":") {
			host += ":80"
		}
		upstream, err := dialUpstream(node, host)
		if err != nil {
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}
		defer upstream.Close()
		req.Write(upstream)
		resp, err := http.ReadResponse(bufio.NewReader(upstream), req)
		if err != nil {
			return
		}
		resp.Write(clientConn)
	}
}

// dialUpstream 根据节点类型建立到目标的连接
func dialUpstream(node *ProxyNode, targetHost string) (net.Conn, error) {
	nodeAddr := fmt.Sprintf("%s:%d", node.Server, node.Port)

	switch node.Type {
	case "http":
		// 通过 HTTP 代理节点
		conn, err := net.DialTimeout("tcp", nodeAddr, 10*time.Second)
		if err != nil {
			return nil, err
		}
		// 发送 CONNECT 到上游 HTTP 代理
		fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", targetHost, targetHost)
		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, nil)
		if err != nil || resp.StatusCode != 200 {
			conn.Close()
			return nil, fmt.Errorf("上游代理拒绝: %v", err)
		}
		return conn, nil

	case "socks5":
		// 通过 SOCKS5 节点
		return dialSocks5(nodeAddr, targetHost, node)

	default:
		// ss/vmess/trojan 等：直连目标（降级，仅测速可用）
		return net.DialTimeout("tcp", targetHost, 10*time.Second)
	}
}

// dialSocks5 简单 SOCKS5 握手
func dialSocks5(proxyAddr, targetHost string, node *ProxyNode) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", proxyAddr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	username, _ := node.Extra["username"].(string)
	password, _ := node.Extra["password"].(string)

	// 握手
	authMethod := byte(0x00) // no auth
	if username != "" {
		authMethod = 0x02 // username/password
	}
	conn.Write([]byte{0x05, 0x01, authMethod})
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		conn.Close()
		return nil, err
	}
	if buf[0] != 0x05 {
		conn.Close()
		return nil, fmt.Errorf("socks5 握手失败")
	}
	if buf[1] == 0x02 && username != "" {
		// 用户名密码认证
		auth := []byte{0x01, byte(len(username))}
		auth = append(auth, []byte(username)...)
		auth = append(auth, byte(len(password)))
		auth = append(auth, []byte(password)...)
		conn.Write(auth)
		if _, err := io.ReadFull(conn, buf); err != nil || buf[1] != 0x00 {
			conn.Close()
			return nil, fmt.Errorf("socks5 认证失败")
		}
	}

	// 发送 CONNECT 请求
	host, portStr, err := net.SplitHostPort(targetHost)
	if err != nil {
		conn.Close()
		return nil, err
	}
	var portNum int
	fmt.Sscanf(portStr, "%d", &portNum)

	req := []byte{0x05, 0x01, 0x00, 0x03, byte(len(host))}
	req = append(req, []byte(host)...)
	req = append(req, byte(portNum>>8), byte(portNum))
	conn.Write(req)

	resp := make([]byte, 10)
	if _, err := io.ReadFull(conn, resp); err != nil || resp[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("socks5 连接失败")
	}
	return conn, nil
}
