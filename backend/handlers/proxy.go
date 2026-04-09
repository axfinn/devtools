package handlers

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

// ProxyNode 代理节点
type ProxyNode struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Server  string                 `json:"server"`
	Port    int                    `json:"port"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
	Latency int64                  `json:"latency"` // ms，-1=失败
}

// proxyPersist 持久化到磁盘的数据
type proxyPersist struct {
	SourceURL   string      `json:"source_url,omitempty"`
	YAMLContent string      `json:"yaml_content,omitempty"`
	Nodes       []ProxyNode `json:"nodes"`
}

const proxyDataFile = "./data/proxy_config.json"

// proxySession 当前会话（内存单例）
type proxySession struct {
	mu          sync.RWMutex
	nodes       []ProxyNode
	sourceURL   string
	yamlContent string
	active      *ProxyNode
	listener    net.Listener // HTTP CONNECT 代理监听
}

var globalSession = &proxySession{}

// loadPersistedProxy 启动时从磁盘恢复节点配置
func loadPersistedProxy() {
	data, err := os.ReadFile(proxyDataFile)
	if err != nil {
		return
	}
	var p proxyPersist
	if err := json.Unmarshal(data, &p); err != nil {
		return
	}
	globalSession.mu.Lock()
	globalSession.nodes = p.Nodes
	globalSession.sourceURL = p.SourceURL
	globalSession.yamlContent = p.YAMLContent
	globalSession.mu.Unlock()
}

// savePersistedProxy 将节点配置写入磁盘
func savePersistedProxy() {
	globalSession.mu.RLock()
	p := proxyPersist{
		SourceURL:   globalSession.sourceURL,
		YAMLContent: globalSession.yamlContent,
		Nodes:       globalSession.nodes,
	}
	globalSession.mu.RUnlock()
	data, _ := json.Marshal(p)
	os.WriteFile(proxyDataFile, data, 0644)
}

// ProxyHandler 科学上网处理器
type ProxyHandler struct {
	adminPassword  string
	npcCfg         npcConfig
	npcTunnelPort  string // NPS 上配置的外网端口，用于前端展示
	npcCmd         *exec.Cmd
	npcMu          sync.Mutex
}

type npcConfig struct {
	serverAddr string // host:port
	vkey       string
	binPath    string // 留空则自动查找
}

func NewProxyHandler(cfg *config.Config) *ProxyHandler {
	loadPersistedProxy()
	h := &ProxyHandler{
		adminPassword: cfg.Proxy.AdminPassword,
		npcTunnelPort: cfg.Proxy.TunnelPort,
	}
	if cfg.NPS.VKey != "" {
		bridgePort := cfg.NPS.BridgePort
		if bridgePort == "" {
			bridgePort = "8024"
		}
		host := cfg.NPS.BridgeHost
		if host == "" {
			if u, err := url.Parse(cfg.NPS.ServerURL); err == nil && u.Hostname() != "" {
				host = u.Hostname()
			}
		}
		h.npcCfg = npcConfig{
			serverAddr: host + ":" + bridgePort,
			vkey:       cfg.NPS.VKey,
		}
	}
	return h
}

func (h *ProxyHandler) checkAdmin(password string) bool {
	return h.adminPassword != "" && password == h.adminPassword
}

// npcBin 查找 npc 可执行文件路径
func npcBin() string {
	if _, err := os.Stat("/usr/local/bin/npc"); err == nil {
		return "/usr/local/bin/npc"
	}
	if _, err := os.Stat("/app/data/npc"); err == nil {
		return "/app/data/npc"
	}
	return ""
}

// startNPC 启动 npc，代理启动时调用
func (h *ProxyHandler) startNPC() {
	if h.npcCfg.vkey == "" {
		return
	}
	bin := h.npcCfg.binPath
	if bin == "" {
		bin = npcBin()
	}
	if bin == "" {
		return
	}
	h.npcMu.Lock()
	if h.npcCmd != nil && h.npcCmd.Process != nil {
		h.npcMu.Unlock()
		return // 已在运行
	}
	cmd := exec.Command(bin, "-server="+h.npcCfg.serverAddr, "-vkey="+h.npcCfg.vkey, "-type=tcp")
	h.npcCmd = cmd
	h.npcMu.Unlock()
	cmd.Start()
	go func() {
		cmd.Wait()
		h.npcMu.Lock()
		if h.npcCmd == cmd {
			h.npcCmd = nil
		}
		h.npcMu.Unlock()
	}()
}

// stopNPC 停止 npc，代理停止时调用
func (h *ProxyHandler) stopNPC() {
	h.npcMu.Lock()
	defer h.npcMu.Unlock()
	if h.npcCmd != nil && h.npcCmd.Process != nil {
		h.npcCmd.Process.Kill()
		h.npcCmd = nil
	}
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
	globalSession.sourceURL = req.SourceURL
	globalSession.yamlContent = req.YAMLContent
	globalSession.mu.Unlock()

	go savePersistedProxy()

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

	go savePersistedProxy()

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

	go serveHTTPProxy(ln, target, h.adminPassword)
	h.startNPC()

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

	h.stopNPC()
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

	h.npcMu.Lock()
	npcRunning := h.npcCmd != nil && h.npcCmd.Process != nil
	h.npcMu.Unlock()

	resp := gin.H{
		"nodes":            globalSession.nodes,
		"source_url":       globalSession.sourceURL,
		"running":          false,
		"npc_running":      npcRunning,
		"npc_tunnel_port":  h.npcTunnelPort,
		"npc_server_addr":  h.npcCfg.serverAddr,
	}
	if globalSession.active != nil && globalSession.listener != nil {
		port := globalSession.listener.Addr().(*net.TCPAddr).Port
		resp["running"] = true
		resp["node"] = globalSession.active.Name
		resp["http_port"] = port
		resp["proxy_url"] = fmt.Sprintf("http://127.0.0.1:%d", port)
	}
	c.JSON(200, resp)
}

// makeProxyClient 构建走代理节点的 http.Client
func makeProxyClient() *http.Client {
	globalSession.mu.RLock()
	ln := globalSession.listener
	globalSession.mu.RUnlock()

	var transport *http.Transport
	if ln != nil {
		port := ln.Addr().(*net.TCPAddr).Port
		pu, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
		transport = &http.Transport{Proxy: http.ProxyURL(pu)}
	} else {
		transport = &http.Transport{}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// proxyGet 通过代理节点 GET 目标 URL，返回 body 和 Content-Type
func proxyGet(targetURL string) ([]byte, string, int, error) {
	client := makeProxyClient()
	req, err := http.NewRequestWithContext(context.Background(), "GET", targetURL, nil)
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	return body, resp.Header.Get("Content-Type"), resp.StatusCode, err
}

// rewriteURLToProxy 将一个 URL 改写为 /api/proxy/resource?url=xxx&p=PASSWORD
func rewriteURLToProxy(rawURL, baseURL, password string) string {
	if rawURL == "" || strings.HasPrefix(rawURL, "data:") ||
		strings.HasPrefix(rawURL, "javascript:") || strings.HasPrefix(rawURL, "#") ||
		strings.HasPrefix(rawURL, "mailto:") {
		return rawURL
	}
	// 解析为绝对 URL
	abs := rawURL
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		base, err := url.Parse(baseURL)
		if err != nil {
			return rawURL
		}
		ref, err := url.Parse(rawURL)
		if err != nil {
			return rawURL
		}
		abs = base.ResolveReference(ref).String()
	}
	return "/api/proxy/resource?p=" + url.QueryEscape(password) + "&url=" + url.QueryEscape(abs)
}

// rewriteHTML 将 HTML 中所有资源/链接 URL 改写为代理路径
func rewriteHTML(html, baseURL, password string) string {
	// 改写属性值：src= href= action= srcset= poster=
	for _, attr := range []string{"src", "href", "action", "poster"} {
		html = rewriteAttrAll(html, attr, baseURL, password)
	}
	// 改写 CSS url(...)
	html = rewriteCSSURLs(html, baseURL, password)
	// 注入 <base> 标签防止相对路径逃逸，并注入拦截脚本
	inject := fmt.Sprintf(`<base href="%s">
<script>
(function(){
  var _p = %q;
  var _base = %q;
  function toProxy(u){
    if(!u||u.startsWith('data:')||u.startsWith('javascript:')||u.startsWith('#')||u.startsWith('/api/proxy/'))return u;
    try{
      var abs = new URL(u, _base).href;
      return '/api/proxy/resource?p='+encodeURIComponent(_p)+'&url='+encodeURIComponent(abs);
    }catch(e){return u;}
  }
  // 拦截 fetch
  var _fetch = window.fetch;
  window.fetch = function(input, init){
    if(typeof input === 'string') input = toProxy(input);
    return _fetch.call(this, input, init);
  };
  // 拦截 XHR
  var _open = XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open = function(m, u){
    arguments[1] = toProxy(u);
    return _open.apply(this, arguments);
  };
  // 拦截链接点击，在同一 iframe 内导航
  document.addEventListener('click', function(e){
    var a = e.target.closest('a');
    if(!a) return;
    var href = a.getAttribute('href');
    if(!href||href.startsWith('#')||href.startsWith('javascript:')) return;
    e.preventDefault();
    var abs;
    try{ abs = new URL(href, _base).href; }catch(e){ return; }
    window.location.href = toProxy(abs);
  }, true);
})();
</script>`, baseURL, password, baseURL)
	if idx := strings.Index(html, "<head>"); idx >= 0 {
		html = html[:idx+6] + inject + html[idx+6:]
	} else if idx := strings.Index(html, "<html"); idx >= 0 {
		end := strings.Index(html[idx:], ">")
		if end >= 0 {
			pos := idx + end + 1
			html = html[:pos] + "<head>" + inject + "</head>" + html[pos:]
		}
	} else {
		html = "<head>" + inject + "</head>" + html
	}
	return html
}

func rewriteAttrAll(html, attr, baseURL, password string) string {
	var sb strings.Builder
	remaining := html
	// 匹配 attr="..." 和 attr='...'
	for _, quote := range []string{`"`, `'`} {
		prefix := attr + "=" + quote
		var out strings.Builder
		rem := remaining
		for {
			idx := strings.Index(rem, prefix)
			if idx < 0 {
				out.WriteString(rem)
				break
			}
			out.WriteString(rem[:idx+len(prefix)])
			rest := rem[idx+len(prefix):]
			end := strings.IndexByte(rest, quote[0])
			if end < 0 {
				out.WriteString(rest)
				break
			}
			link := rest[:end]
			link = rewriteURLToProxy(link, baseURL, password)
			out.WriteString(link)
			rem = rest[end:]
		}
		remaining = out.String()
	}
	sb.WriteString(remaining)
	return sb.String()
}

func rewriteCSSURLs(html, baseURL, password string) string {
	// 替换 url("...") url('...') url(...)
	var sb strings.Builder
	rem := html
	prefix := "url("
	for {
		idx := strings.Index(rem, prefix)
		if idx < 0 {
			sb.WriteString(rem)
			break
		}
		sb.WriteString(rem[:idx+len(prefix)])
		rest := rem[idx+len(prefix):]
		var link, tail string
		if strings.HasPrefix(rest, `"`) {
			end := strings.Index(rest[1:], `"`)
			if end < 0 {
				sb.WriteString(rest)
				break
			}
			link = rest[1 : end+1]
			tail = rest[end+2:]
			link = rewriteURLToProxy(link, baseURL, password)
			sb.WriteString(`"` + link + `"`)
		} else if strings.HasPrefix(rest, `'`) {
			end := strings.Index(rest[1:], `'`)
			if end < 0 {
				sb.WriteString(rest)
				break
			}
			link = rest[1 : end+1]
			tail = rest[end+2:]
			link = rewriteURLToProxy(link, baseURL, password)
			sb.WriteString(`'` + link + `'`)
		} else {
			end := strings.IndexByte(rest, ')')
			if end < 0 {
				sb.WriteString(rest)
				break
			}
			link = strings.TrimSpace(rest[:end])
			tail = rest[end:]
			link = rewriteURLToProxy(link, baseURL, password)
			sb.WriteString(link)
		}
		rem = tail
	}
	return sb.String()
}

// Fetch GET /api/proxy/fetch?url=xxx&admin_password=xxx
// 抓取目标 HTML，将所有资源改写为代理路径，注入拦截脚本
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

	body, ct, _, err := proxyGet(targetURL)
	if err != nil {
		c.JSON(502, gin.H{"error": "请求失败: " + err.Error()})
		return
	}

	if strings.Contains(ct, "text/html") || ct == "" {
		html := rewriteHTML(string(body), targetURL, password)
		c.Header("X-Frame-Options", "")
		c.Header("Content-Security-Policy", "")
		c.Data(200, "text/html; charset=utf-8", []byte(html))
	} else {
		c.Data(200, ct, body)
	}
}

// Resource GET /api/proxy/resource?url=xxx&p=PASSWORD
// 代理任意资源（CSS/JS/图片等），CSS 内容也做 URL 改写
func (h *ProxyHandler) Resource(c *gin.Context) {
	password := c.Query("p")
	if !h.checkAdmin(password) {
		c.Data(403, "text/plain", []byte("forbidden"))
		return
	}

	targetURL := c.Query("url")
	if targetURL == "" {
		c.Data(400, "text/plain", []byte("missing url"))
		return
	}

	body, ct, status, err := proxyGet(targetURL)
	if err != nil {
		c.Data(502, "text/plain", []byte("proxy error: "+err.Error()))
		return
	}

	// CSS 内也做 URL 改写
	if strings.Contains(ct, "text/css") {
		rewritten := rewriteCSSURLs(string(body), targetURL, password)
		c.Data(status, ct, []byte(rewritten))
		return
	}
	// HTML（跳转到的子页面）
	if strings.Contains(ct, "text/html") {
		html := rewriteHTML(string(body), targetURL, password)
		c.Header("X-Frame-Options", "")
		c.Header("Content-Security-Policy", "")
		c.Data(status, "text/html; charset=utf-8", []byte(html))
		return
	}

	// 其他资源直接透传，去掉 CSP 头
	c.Header("Content-Security-Policy", "")
	c.Header("X-Frame-Options", "")
	c.Data(status, ct, body)
}

// fakeWebPage 伪装成普通 nginx 页面，防止 GFW 主动探测识别代理
const fakeWebPage = "HTTP/1.1 200 OK\r\nServer: nginx/1.24.0\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: 55\r\nConnection: close\r\n\r\n<html><body><h1>Welcome to nginx!</h1></body></html>"

// serveHTTPProxy 运行一个简单的 HTTP CONNECT 代理，转发到目标节点
// 支持 http/socks5 类型节点的上游代理，其他类型直连
func serveHTTPProxy(ln net.Listener, node *ProxyNode, adminPassword string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleProxyConn(conn, node, adminPassword)
	}
}

func handleProxyConn(clientConn net.Conn, node *ProxyNode, adminPassword string) {
	defer clientConn.Close()
	clientConn.SetDeadline(time.Now().Add(30 * time.Second))

	br := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(br)
	if err != nil {
		// 非 HTTP 协议（如二进制探测）：静默关闭
		return
	}

	// 防探测：验证 Proxy-Authorization，失败时伪装成普通网站
	if adminPassword != "" {
		authHeader := req.Header.Get("Proxy-Authorization")
		if !checkProxyAuthHeader(authHeader, adminPassword) {
			clientConn.Write([]byte(fakeWebPage))
			return
		}
	}

	clientConn.SetDeadline(time.Time{}) // 认证通过后取消超时

	if req.Method == http.MethodConnect {
		// HTTPS CONNECT 隧道
		host := req.Host
		upstream, err := dialUpstream(node, host)
		if err != nil {
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nServer: nginx/1.24.0\r\n\r\n"))
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
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nServer: nginx/1.24.0\r\n\r\n"))
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

// Tunnel 处理 HTTP CONNECT 方法，让 DevTools 自身端口充当 HTTP 代理
// 浏览器/系统代理配置：http://yourserver:PORT
// 密码通过 Proxy-Authorization: Basic base64(user:password) 传递
func (h *ProxyHandler) Tunnel(w http.ResponseWriter, r *http.Request) {
	// 防探测：无认证或认证失败时伪装成普通网站，不暴露 407
	if !h.checkProxyAuth(r.Header.Get("Proxy-Authorization")) {
		w.Header().Set("Server", "nginx/1.24.0")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("<html><body><h1>Welcome to nginx!</h1></body></html>"))
		return
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	globalSession.mu.RLock()
	node := globalSession.active
	globalSession.mu.RUnlock()

	var (
		upstream net.Conn
		err      error
	)
	if node != nil {
		upstream, err = dialUpstream(node, host)
	} else {
		upstream, err = net.DialTimeout("tcp", host, 10*time.Second)
	}
	if err != nil {
		http.Error(w, "Bad Gateway", 502)
		return
	}

	// 劫持底层 TCP 连接
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		upstream.Close()
		http.Error(w, "Hijacking not supported", 500)
		return
	}
	clientConn, brw, err := hijacker.Hijack()
	if err != nil {
		upstream.Close()
		return
	}

	brw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
	brw.Flush()

	go io.Copy(upstream, brw)
	io.Copy(clientConn, upstream)
	upstream.Close()
	clientConn.Close()
}

// checkProxyAuth 验证 Proxy-Authorization Basic 头
func (h *ProxyHandler) checkProxyAuth(header string) bool {
	if h.adminPassword == "" {
		return false
	}
	if !strings.HasPrefix(header, "Basic ") {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return false
	}
	// 格式 user:password，user 随意，password 必须匹配
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}
	return parts[1] == h.adminPassword
}

// checkProxyAuthHeader 独立函数版本，供 handleProxyConn 使用
func checkProxyAuthHeader(header, adminPassword string) bool {
	if adminPassword == "" {
		return false
	}
	if !strings.HasPrefix(header, "Basic ") {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}
	return parts[1] == adminPassword
}

// DownloadExtension GET /api/proxy/extension?admin_password=xxx&host=xxx
// 生成并下载 Chrome 扩展 zip，导入后自动配置代理
func (h *ProxyHandler) DownloadExtension(c *gin.Context) {
	password := c.Query("admin_password")
	if !h.checkAdmin(password) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}

	// 优先用请求头里的 Host，也支持前端传 host 参数
	proxyHost := c.Query("host")
	if proxyHost == "" {
		proxyHost = c.Request.Host
	}

	// MV3 manifest — 使用 PAC 脚本代理 + onAuthRequired 自动填充认证
	manifest := `{
  "manifest_version": 3,
  "name": "DevTools Proxy",
  "version": "3.0",
  "description": "通过服务器 HTTP 代理科学上网，支持自动认证",
  "permissions": [
    "storage",
    "proxy",
    "webRequest"
  ],
  "host_permissions": ["<all_urls>"],
  "background": {
    "service_worker": "background.js"
  },
  "action": {
    "default_popup": "popup.html",
    "default_title": "DevTools Proxy"
  }
}`

	// background.js：PAC 脚本代理 + onAuthRequired 自动填充认证
	bgJS := fmt.Sprintf(`
const DEFAULT_SERVER = %q;
const DEFAULT_PASS   = %q;

function buildPac(server) {
  // PAC 脚本：所有流量走服务器 HTTP 代理
  // HTTP 流量直接代理；HTTPS 通过 CONNECT 隧道（需服务器直连，不经 nginx）
  return 'function FindProxyForURL(url, host) {' +
    'if (isPlainHostName(host) || host === "127.0.0.1" || host === "localhost") return "DIRECT";' +
    'return "PROXY ' + server + '";' +
    '}';
}

function applyProxy(server) {
  const pac = buildPac(server);
  chrome.proxy.settings.set({
    value: { mode: 'pac_script', pacScript: { data: pac } },
    scope: 'regular'
  });
}

function disableProxy() {
  chrome.proxy.settings.clear({ scope: 'regular' });
}

function loadAndApply() {
  chrome.storage.local.get(['proxyEnabled', 'server'], (s) => {
    if (s.proxyEnabled !== false) {
      applyProxy(s.server || DEFAULT_SERVER);
    } else {
      disableProxy();
    }
  });
}

chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.local.set({ server: DEFAULT_SERVER, pass: DEFAULT_PASS, proxyEnabled: true });
  loadAndApply();
});
chrome.runtime.onStartup.addListener(loadAndApply);
loadAndApply();

// 自动填充代理认证（Proxy-Authorization）
chrome.webRequest.onAuthRequired.addListener(
  (details, callback) => {
    if (details.isProxy) {
      chrome.storage.local.get(['pass'], (s) => {
        callback({ authCredentials: { username: 'proxy', password: s.pass || DEFAULT_PASS } });
      });
      return true; // 异步
    }
    callback({});
  },
  { urls: ['<all_urls>'] },
  ['asyncBlocking']
);

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.action === 'update') {
    chrome.storage.local.set({ server: msg.server, pass: msg.pass, proxyEnabled: true }, () => {
      applyProxy(msg.server);
    });
  } else if (msg.action === 'disable') {
    chrome.storage.local.set({ proxyEnabled: false });
    disableProxy();
  } else if (msg.action === 'enable') {
    chrome.storage.local.get(['server'], (s) => {
      chrome.storage.local.set({ proxyEnabled: true });
      applyProxy(s.server || DEFAULT_SERVER);
    });
  }
  sendResponse({});
  return true;
});
`, proxyHost, password)

	popupHTML := `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><style>
*{box-sizing:border-box;}
body{font-family:sans-serif;padding:14px;width:320px;font-size:13px;margin:0;}
label{display:block;color:#666;margin-bottom:3px;margin-top:10px;}
input{width:100%;padding:5px 8px;border:1px solid #ddd;border-radius:4px;font-size:13px;}
.row{display:flex;align-items:center;gap:8px;margin-top:12px;}
.status{flex:1;font-weight:bold;}
.on{color:#67c23a;} .off{color:#f56c6c;}
button{padding:6px 14px;border-radius:4px;border:none;cursor:pointer;font-size:13px;}
.btn-save{background:#409eff;color:#fff;}
.btn-on{background:#67c23a;color:#fff;}
.btn-off{background:#f56c6c;color:#fff;}
.hint{font-size:11px;color:#999;margin-top:6px;line-height:1.5;}
</style></head>
<body>
<label>服务器地址（host:port 或 域名）</label>
<input id="server" placeholder="example.com 或 1.2.3.4:8082">
<label>密码</label>
<input id="pass" type="password" placeholder="管理员密码">
<div class="hint">HTTP 代理，流量经 NPS 隧道转发。安装后可在弹窗中开启/关闭代理。</div>
<div class="row">
  <span class="status" id="status">检测中...</span>
  <button class="btn-save" id="saveBtn">保存并启用</button>
  <button id="toggleBtn">-</button>
</div>
<script src="popup.js"></script>
</body></html>`

	popupJS := `
var enabled = true;

chrome.storage.local.get(['server','pass','proxyEnabled'], function(s) {
  document.getElementById('server').value = s.server || '';
  document.getElementById('pass').value = s.pass || '';
  enabled = s.proxyEnabled !== false;
  updateUI();
});

function updateUI() {
  document.getElementById('status').textContent = enabled ? '代理已启用' : '代理已关闭';
  document.getElementById('status').className = 'status ' + (enabled ? 'on' : 'off');
  document.getElementById('toggleBtn').textContent = enabled ? '关闭' : '开启';
  document.getElementById('toggleBtn').className = enabled ? 'btn-off' : 'btn-on';
}

document.getElementById('saveBtn').addEventListener('click', function() {
  var server = document.getElementById('server').value.trim();
  var pass = document.getElementById('pass').value.trim();
  if (!server || !pass) { alert('请填写服务器地址和密码'); return; }
  chrome.runtime.sendMessage({ action: 'update', server: server, pass: pass });
  enabled = true;
  updateUI();
});

document.getElementById('toggleBtn').addEventListener('click', function() {
  enabled = !enabled;
  chrome.runtime.sendMessage({ action: enabled ? 'enable' : 'disable' });
  updateUI();
});
`

	// 打包成 zip
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"manifest.json": manifest,
		"background.js": bgJS,
		"popup.html":    popupHTML,
		"popup.js":      popupJS,
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			c.JSON(500, gin.H{"error": "打包失败"})
			return
		}
		w.Write([]byte(content))
	}
	zw.Close()

	c.Header("Content-Disposition", `attachment; filename="devtools-proxy.zip"`)
	c.Data(200, "application/zip", buf.Bytes())
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WsTunnel GET /api/proxy/ws-tunnel?p=PASSWORD&host=example.com:443
// 通过 WebSocket 建立 TCP 隧道，绕过 nginx 对 CONNECT 的限制
// 客户端用 wstunnel: wstunnel client -L 'socks5://127.0.0.1:1080' wss://yourserver.com/api/proxy/ws-tunnel?p=PASS
// DownloadClient GET /api/proxy/client/download?os=darwin&arch=arm64&admin_password=xxx
// 下载对应平台的 proxy-client 二进制
func (h *ProxyHandler) DownloadClient(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}
	goos := c.Query("os")
	arch := c.Query("arch")
	if goos == "" || arch == "" {
		c.JSON(400, gin.H{"error": "缺少 os 或 arch 参数"})
		return
	}

	var filename string
	if goos == "windows" {
		filename = fmt.Sprintf("proxy-client-%s-%s.exe", goos, arch)
	} else {
		filename = fmt.Sprintf("proxy-client-%s-%s", goos, arch)
	}

	filePath := "./proxy-client-bins/" + filename
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "该平台的客户端不存在，请联系管理员重新构建"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}

func (h *ProxyHandler) WsTunnel(c *gin.Context) {
	if !h.checkAdmin(c.Query("p")) {
		c.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	host := c.Query("host")
	if host == "" {
		c.JSON(400, gin.H{"error": "missing host"})
		return
	}

	wsConn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer wsConn.Close()

	globalSession.mu.RLock()
	node := globalSession.active
	globalSession.mu.RUnlock()

	var upstream net.Conn
	if node != nil {
		upstream, err = dialUpstream(node, host)
	} else {
		upstream, err = net.DialTimeout("tcp", host, 10*time.Second)
	}
	if err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error()))
		return
	}
	defer upstream.Close()

	// ws → tcp
	go func() {
		for {
			_, msg, err := wsConn.ReadMessage()
			if err != nil {
				upstream.Close()
				return
			}
			upstream.Write(msg)
		}
	}()

	// tcp → ws
	buf := make([]byte, 32*1024)
	for {
		n, err := upstream.Read(buf)
		if n > 0 {
			if werr := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
