package handlers

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
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
	SourceURL        string      `json:"source_url,omitempty"`
	SourceURLs       []string    `json:"source_urls,omitempty"`
	YAMLContent      string      `json:"yaml_content,omitempty"`
	RoutingMode      string      `json:"routing_mode,omitempty"`
	DefaultNodeName  string      `json:"default_node_name,omitempty"`
	DefaultNodeRegex string      `json:"default_node_regex,omitempty"`
	AINodeName       string      `json:"ai_node_name,omitempty"`
	AINodeRegex      string      `json:"ai_node_regex,omitempty"`
	Nodes            []ProxyNode `json:"nodes"`
}

const proxyDataFile = "./data/proxy_config.json"

// proxySession 当前会话（内存单例）
type proxySession struct {
	mu               sync.RWMutex
	nodes            []ProxyNode
	sourceURL        string
	sourceURLs       []string
	yamlContent      string
	routingMode      string
	defaultNodeName  string
	defaultNodeRegex string
	aiNodeName       string
	aiNodeRegex      string
	active           *ProxyNode
	listener         net.Listener // HTTP CONNECT 代理监听
}

var globalSession = &proxySession{}

const (
	proxyRouteModeSmart      = "smart"
	proxyRouteModeGlobal     = "global"
	proxyRouteModeAIPriority = "ai_priority"
	proxyProbeConcurrency    = 12
)

var aiDedicatedDomains = []string{
	"openai.com",
	"api.openai.com",
	"chatgpt.com",
	"chat.openai.com",
	"oaistatic.com",
	"oaiusercontent.com",
	"auth.openai.com",
	"claude.ai",
	"anthropic.com",
	"api.anthropic.com",
	"console.anthropic.com",
	"gemini.google.com",
	"generativelanguage.googleapis.com",
	"makersuite.google.com",
	"copilot.microsoft.com",
	"bing.com",
	"grok.com",
	"x.ai",
	"poe.com",
	"perplexity.ai",
}

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
	globalSession.sourceURLs = normalizeSourceURLs(p.SourceURLs, p.SourceURL)
	globalSession.yamlContent = p.YAMLContent
	globalSession.routingMode = normalizeProxyRouteMode(p.RoutingMode)
	globalSession.defaultNodeName = strings.TrimSpace(p.DefaultNodeName)
	globalSession.defaultNodeRegex = strings.TrimSpace(p.DefaultNodeRegex)
	globalSession.aiNodeName = strings.TrimSpace(p.AINodeName)
	globalSession.aiNodeRegex = strings.TrimSpace(p.AINodeRegex)
	globalSession.mu.Unlock()
}

// savePersistedProxy 将节点配置写入磁盘
func savePersistedProxy() {
	globalSession.mu.RLock()
	p := proxyPersist{
		SourceURL:        firstSourceURL(globalSession.sourceURLs, globalSession.sourceURL),
		SourceURLs:       append([]string(nil), globalSession.sourceURLs...),
		YAMLContent:      globalSession.yamlContent,
		RoutingMode:      normalizeProxyRouteMode(globalSession.routingMode),
		DefaultNodeName:  globalSession.defaultNodeName,
		DefaultNodeRegex: globalSession.defaultNodeRegex,
		AINodeName:       globalSession.aiNodeName,
		AINodeRegex:      globalSession.aiNodeRegex,
		Nodes:            globalSession.nodes,
	}
	globalSession.mu.RUnlock()
	data, _ := json.Marshal(p)
	os.WriteFile(proxyDataFile, data, 0644)
}

func normalizeProxyRouteMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", proxyRouteModeAIPriority:
		return proxyRouteModeAIPriority
	case proxyRouteModeSmart:
		return proxyRouteModeSmart
	case proxyRouteModeGlobal:
		return proxyRouteModeGlobal
	default:
		return proxyRouteModeAIPriority
	}
}

func firstSourceURL(sourceURLs []string, fallback string) string {
	if len(sourceURLs) > 0 {
		return sourceURLs[0]
	}
	return strings.TrimSpace(fallback)
}

func normalizeSourceURLs(sourceURLs []string, fallback string) []string {
	seen := make(map[string]struct{}, len(sourceURLs)+1)
	result := make([]string, 0, len(sourceURLs)+1)
	appendURL := func(raw string) {
		for _, line := range strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' }) {
			u := strings.TrimSpace(line)
			if u == "" {
				continue
			}
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			result = append(result, u)
		}
	}
	for _, raw := range sourceURLs {
		appendURL(raw)
	}
	appendURL(fallback)
	return result
}

func newDirectHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:               nil,
			DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:   true,
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
		Timeout: timeout,
	}
}

func newHTTPClient(timeout time.Duration, proxyURL string, withCookies bool) (*http.Client, error) {
	transport := &http.Transport{
		Proxy:               nil,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if strings.TrimSpace(proxyURL) != "" {
		pu, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(pu)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
	if withCookies {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		client.Jar = jar
	}
	return client, nil
}

func currentProxyURL() string {
	globalSession.mu.RLock()
	ln := globalSession.listener
	globalSession.mu.RUnlock()
	if ln == nil {
		return ""
	}

	pu, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port))
	if err != nil {
		return ""
	}
	if globalProxyHandler != nil && globalProxyHandler.adminPassword != "" {
		pu.User = url.UserPassword("proxy", globalProxyHandler.adminPassword)
	}
	return pu.String()
}

type managedRefreshClientCandidate struct {
	source string
	client *http.Client
}

type managedRefreshAttempt struct {
	Source       string `json:"source"`
	SiteURL      string `json:"site_url,omitempty"`
	SubscribeURL string `json:"subscribe_url,omitempty"`
	Stage        string `json:"stage"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	NodeCount    int    `json:"node_count,omitempty"`
	NodeHint     string `json:"node_hint,omitempty"`
	Changed      bool   `json:"changed"`
	Applied      bool   `json:"applied"`
}

func (h *ProxyHandler) sendManagedRefreshReport(subject, outcome string, attempts []managedRefreshAttempt, extra map[string]interface{}) {
	if h.alertEmail == "" || h.smtpHost == "" || h.smtpUser == "" {
		return
	}

	report := map[string]interface{}{
		"time":       time.Now().Format(time.RFC3339),
		"outcome":    outcome,
		"attempts":   attempts,
		"active_url": currentProxyURL(),
	}
	for k, v := range extra {
		report[k] = v
	}

	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		payload = []byte(fmt.Sprintf("{\"marshal_error\":%q}", err.Error()))
	}

	lines := []string{
		fmt.Sprintf("结果：%s", outcome),
		fmt.Sprintf("时间：%s", time.Now().Format("2006-01-02 15:04:05")),
		fmt.Sprintf("当前活动代理：%s", currentProxyURL()),
		"",
		"全链路详情：",
	}
	if len(attempts) == 0 {
		lines = append(lines, "- 无尝试记录")
	} else {
		for idx, attempt := range attempts {
			lines = append(lines,
				fmt.Sprintf("%d. source=%s stage=%s success=%t changed=%t applied=%t", idx+1, attempt.Source, attempt.Stage, attempt.Success, attempt.Changed, attempt.Applied),
				fmt.Sprintf("   site_url=%s", attempt.SiteURL),
				fmt.Sprintf("   subscribe_url=%s", attempt.SubscribeURL),
				fmt.Sprintf("   node_hint=%s node_count=%d", attempt.NodeHint, attempt.NodeCount),
			)
			if attempt.Error != "" {
				lines = append(lines, fmt.Sprintf("   error=%s", attempt.Error))
			}
		}
	}
	lines = append(lines, "", "JSON:", string(payload))

	go h.sendAlert(subject, strings.Join(lines, "\n"))
}

func (h *ProxyHandler) managedRefreshClients(timeout time.Duration) []managedRefreshClientCandidate {
	candidates := make([]managedRefreshClientCandidate, 0, 3)
	seen := make(map[string]struct{}, 3)
	appendClient := func(source, proxyURL string) {
		key := source + "|" + strings.TrimSpace(proxyURL)
		if _, ok := seen[key]; ok {
			return
		}
		client, err := newHTTPClient(timeout, proxyURL, true)
		if err != nil {
			log.Printf("proxy: 构建托管订阅 %s client 失败: %v", source, err)
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, managedRefreshClientCandidate{
			source: source,
			client: client,
		})
	}

	if proxyURL := currentProxyURL(); proxyURL != "" {
		appendClient("active_proxy", proxyURL)
	}
	if proxyURL := strings.TrimSpace(h.subRefresh.requestProxyURL); proxyURL != "" {
		appendClient("configured_proxy", proxyURL)
	}
	appendClient("direct", "")
	return candidates
}

func proxyNodeKey(node ProxyNode) string {
	return strings.ToLower(strings.TrimSpace(node.Type)) + "|" +
		strings.ToLower(strings.TrimSpace(node.Server)) + "|" +
		strconv.Itoa(node.Port) + "|" +
		strings.ToLower(strings.TrimSpace(node.Name))
}

func mergeProxyNodes(groups ...[]ProxyNode) []ProxyNode {
	seen := make(map[string]struct{})
	merged := make([]ProxyNode, 0)
	for _, group := range groups {
		for _, node := range group {
			key := proxyNodeKey(node)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, node)
		}
	}
	return merged
}

func fetchSubscriptions(sourceURLs []string) (string, []ProxyNode, error) {
	urls := normalizeSourceURLs(sourceURLs, "")
	if len(urls) == 0 {
		return "", nil, fmt.Errorf("未提供订阅 URL")
	}

	type fetchResult struct {
		index int
		url   string
		text  string
		nodes []ProxyNode
		err   error
	}

	results := make([]fetchResult, len(urls))
	var wg sync.WaitGroup
	sem := make(chan struct{}, minInt(proxyProbeConcurrency, len(urls)))

	for i, sourceURL := range urls {
		wg.Add(1)
		go func(idx int, rawURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			text, err := fetchSubscription(rawURL)
			if err != nil {
				results[idx] = fetchResult{index: idx, url: rawURL, err: err}
				return
			}
			nodes, err := parseClashYAML(text)
			results[idx] = fetchResult{index: idx, url: rawURL, text: text, nodes: nodes, err: err}
		}(i, sourceURL)
	}
	wg.Wait()

	yamls := make([]string, 0, len(results))
	merged := make([][]ProxyNode, 0, len(results))
	failures := make([]string, 0)
	successCount := 0
	for _, item := range results {
		if item.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", item.url, item.err))
			continue
		}
		if len(item.nodes) == 0 {
			failures = append(failures, fmt.Sprintf("%s: 未解析到节点", item.url))
			continue
		}
		successCount++
		yamls = append(yamls, item.text)
		merged = append(merged, item.nodes)
	}
	if successCount == 0 {
		return "", nil, fmt.Errorf("所有订阅均不可用: %s", strings.Join(failures, "; "))
	}

	nodes := mergeProxyNodes(merged...)
	joinedYAML := strings.Join(yamls, "\n\n# ---- merged subscription ----\n\n")
	if len(failures) > 0 {
		log.Printf("proxy: 部分订阅拉取失败，成功 %d/%d: %s", successCount, len(urls), strings.Join(failures, "; "))
	}
	return joinedYAML, nodes, nil
}

func joinURL(baseURL, path string) string {
	baseURL = strings.TrimSpace(baseURL)
	path = strings.TrimSpace(path)
	if baseURL == "" {
		return ""
	}
	if path == "" {
		return baseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(path)
	if err != nil {
		return ""
	}
	return u.ResolveReference(ref).String()
}

func maskURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	if len(u.Path) <= 12 {
		return u.Scheme + "://" + u.Host + u.Path
	}
	return u.Scheme + "://" + u.Host + u.Path[:12] + "..."
}

func extractLatestSiteURL(baseURL, html string) string {
	html = strings.TrimSpace(html)
	if html != "" {
		re := regexp.MustCompile(`https?://[^\s"'<>]+`)
		candidates := re.FindAllString(html, -1)
		seen := make(map[string]struct{}, len(candidates))
		for _, candidate := range candidates {
			u, err := url.Parse(candidate)
			if err != nil || u.Host == "" {
				continue
			}
			host := strings.ToLower(u.Hostname())
			if strings.Contains(host, "t.me") || strings.Contains(host, "telegram.me") || strings.Contains(host, "github.io") {
				continue
			}
			candidate = strings.TrimRight(candidate, "/")
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			return candidate
		}
	}
	if u, err := url.Parse(strings.TrimSpace(baseURL)); err == nil && u.Scheme != "" && u.Host != "" {
		return strings.TrimRight(u.Scheme+"://"+u.Host, "/")
	}
	return ""
}

func extractManagedSubscribeURLs(html string) []string {
	re := regexp.MustCompile(`https?://[^"'\s<>]+/link/[^"'\s<>]+`)
	matches := re.FindAllString(html, -1)
	seen := make(map[string]struct{}, len(matches))
	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		match = strings.TrimSpace(strings.TrimRight(match, `"'`))
		if match == "" {
			continue
		}
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		urls = append(urls, match)
	}
	return urls
}

func subscriptionURLScore(rawURL, preferredType string) int {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	query := u.Query()
	score := 0
	switch preferredType {
	case "clash":
		switch query.Get("clash") {
		case "1":
			score += 100
		case "2":
			score += 90
		}
	case "clashr":
		if query.Get("clash") == "2" {
			score += 100
		}
	case "ssr":
		if query.Get("sub") == "1" {
			score += 100
		}
	case "surfboard":
		if query.Get("surfboard") == "1" {
			score += 100
		}
	case "shadowrocket":
		if query.Get("list") == "shadowrocket" {
			score += 100
		}
	}
	if query.Get("clash") == "1" {
		score += 15
	}
	if query.Get("sub") == "1" {
		score += 10
	}
	if query.Get("surfboard") == "1" {
		score += 5
	}
	if preferredType != "" && strings.Contains(strings.ToLower(rawURL), preferredType) {
		score += 20
	}
	return score
}

func selectPreferredSubscriptionURL(urls []string, preferredType string) string {
	if len(urls) == 0 {
		return ""
	}
	sorted := append([]string(nil), urls...)
	sort.SliceStable(sorted, func(i, j int) bool {
		leftScore := subscriptionURLScore(sorted[i], preferredType)
		rightScore := subscriptionURLScore(sorted[j], preferredType)
		if leftScore == rightScore {
			return len(sorted[i]) < len(sorted[j])
		}
		return leftScore > rightScore
	})
	return sorted[0]
}

func managedSubscriptionChanged(currentURLs []string, currentYAML string, nextURL, nextYAML string) bool {
	currentURLs = normalizeSourceURLs(currentURLs, "")
	nextURLs := normalizeSourceURLs([]string{nextURL}, "")
	if len(currentURLs) != len(nextURLs) {
		return true
	}
	for i := range currentURLs {
		if currentURLs[i] != nextURLs[i] {
			return true
		}
	}
	return strings.TrimSpace(currentYAML) != strings.TrimSpace(nextYAML)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ProxyHandler 科学上网处理器
type ProxyHandler struct {
	adminPassword string
	localPort     string // 本地节点代理固定端口，空则随机
	npcCfg        npcConfig
	npcTunnelPort string // NPS 上配置的外网端口，用于前端展示
	npcCmd        *exec.Cmd
	npcMu         sync.Mutex
	npsHandler    *NPSHandler // 用于一键创建 NPS 端口映射
	alertEmail    string
	smtpHost      string
	smtpPort      int
	smtpUser      string
	smtpPass      string
	subRefresh    proxySubscriptionRefreshRuntime
	subRefreshMu  sync.Mutex
	subStateMu    sync.RWMutex
	subState      proxySubscriptionRefreshState
	checkMu       sync.Mutex
	checkStateMu  sync.RWMutex
	checkState    proxyReachabilityCheckState
}

type proxySubscriptionRefreshRuntime struct {
	enabled          bool
	interval         time.Duration
	landingURL       string
	siteURL          string
	requestProxyURL  string
	loginEmail       string
	loginPassword    string
	loginPath        string
	userPath         string
	preferredSubType string
}

type proxySubscriptionRefreshState struct {
	Enabled             bool
	ResolvedSiteURL     string
	LastSubscribeURL    string
	LastRefreshStatus   string
	LastRefreshAt       string
	LastRefreshChanged  bool
	LastRefreshApplied  bool
	LastRefreshSource   string
	LastRefreshNodeHint string
}

type proxyReachabilityResult struct {
	Name      string `json:"name"`
	Reachable bool   `json:"reachable"`
	Latency   int64  `json:"latency"`
}

type proxyReachabilityCheckState struct {
	Running         bool
	LastCheckStatus string
	LastCheckAt     string
	Results         []proxyReachabilityResult
}

type npcConfig struct {
	serverAddr string // host:port
	vkey       string
	binPath    string // 留空则自动查找
}

func NewProxyHandler(cfg *config.Config, npsHandler *NPSHandler) *ProxyHandler {
	loadPersistedProxy()
	refreshCfg := cfg.Proxy.SubscriptionRefresh
	refreshInterval := time.Duration(refreshCfg.IntervalHours) * time.Hour
	if refreshInterval <= 0 {
		refreshInterval = 24 * time.Hour
	}
	loginPath := strings.TrimSpace(refreshCfg.LoginPath)
	if loginPath == "" {
		loginPath = "/auth/login"
	}
	userPath := strings.TrimSpace(refreshCfg.UserPath)
	if userPath == "" {
		userPath = "/user"
	}
	preferredSubType := strings.TrimSpace(strings.ToLower(refreshCfg.PreferredSubType))
	if preferredSubType == "" {
		preferredSubType = "clash"
	}
	h := &ProxyHandler{
		adminPassword: cfg.Proxy.AdminPassword,
		localPort:     cfg.Proxy.LocalPort,
		npcTunnelPort: cfg.Proxy.TunnelPort,
		npsHandler:    npsHandler,
		alertEmail:    cfg.Proxy.AlertEmail,
		smtpHost:      cfg.Proxy.SMTPHost,
		smtpPort:      cfg.Proxy.SMTPPort,
		smtpUser:      cfg.Proxy.SMTPUser,
		smtpPass:      cfg.Proxy.SMTPPass,
		subRefresh: proxySubscriptionRefreshRuntime{
			enabled:          refreshCfg.Enabled,
			interval:         refreshInterval,
			landingURL:       strings.TrimSpace(refreshCfg.LandingURL),
			siteURL:          strings.TrimSpace(refreshCfg.SiteURL),
			requestProxyURL:  strings.TrimSpace(refreshCfg.RequestProxyURL),
			loginEmail:       strings.TrimSpace(refreshCfg.LoginEmail),
			loginPassword:    strings.TrimSpace(refreshCfg.LoginPassword),
			loginPath:        loginPath,
			userPath:         userPath,
			preferredSubType: preferredSubType,
		},
		subState: proxySubscriptionRefreshState{
			Enabled:           refreshCfg.Enabled,
			LastRefreshStatus: "未执行",
		},
		checkState: proxyReachabilityCheckState{
			LastCheckStatus: "未执行",
		},
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
	globalProxyHandler = h
	return h
}

func (h *ProxyHandler) updateSubscriptionState(update func(*proxySubscriptionRefreshState)) {
	h.subStateMu.Lock()
	defer h.subStateMu.Unlock()
	update(&h.subState)
}

func (h *ProxyHandler) snapshotSubscriptionState() proxySubscriptionRefreshState {
	h.subStateMu.RLock()
	defer h.subStateMu.RUnlock()
	return h.subState
}

func (h *ProxyHandler) markSubscriptionState(status, siteURL, subscribeURL, source, nodeHint string, changed, applied bool) {
	h.updateSubscriptionState(func(state *proxySubscriptionRefreshState) {
		state.Enabled = h.subRefresh.enabled
		if siteURL != "" {
			state.ResolvedSiteURL = siteURL
		}
		if subscribeURL != "" {
			state.LastSubscribeURL = subscribeURL
		}
		state.LastRefreshStatus = status
		state.LastRefreshAt = time.Now().Format(time.RFC3339)
		state.LastRefreshChanged = changed
		state.LastRefreshApplied = applied
		state.LastRefreshSource = source
		state.LastRefreshNodeHint = nodeHint
	})
}

func (h *ProxyHandler) updateCheckState(update func(*proxyReachabilityCheckState)) {
	h.checkStateMu.Lock()
	defer h.checkStateMu.Unlock()
	update(&h.checkState)
}

func (h *ProxyHandler) snapshotCheckState() proxyReachabilityCheckState {
	h.checkStateMu.RLock()
	defer h.checkStateMu.RUnlock()
	state := h.checkState
	state.Results = append([]proxyReachabilityResult(nil), h.checkState.Results...)
	return state
}

func (h *ProxyHandler) markCheckState(status string, running bool, results []proxyReachabilityResult) {
	h.updateCheckState(func(state *proxyReachabilityCheckState) {
		state.Running = running
		state.LastCheckStatus = status
		state.LastCheckAt = time.Now().Format(time.RFC3339)
		if results != nil {
			state.Results = append([]proxyReachabilityResult(nil), results...)
		}
	})
}

func (h *ProxyHandler) resolveManagedSiteURL(client *http.Client) (string, string, error) {
	if strings.TrimSpace(h.subRefresh.siteURL) != "" {
		siteURL := strings.TrimRight(strings.TrimSpace(h.subRefresh.siteURL), "/")
		return siteURL, siteURL, nil
	}
	if strings.TrimSpace(h.subRefresh.landingURL) == "" {
		return "", "", errors.New("未配置 landing_url 或 site_url")
	}
	resp, err := client.Get(h.subRefresh.landingURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", "", err
	}
	siteURL := extractLatestSiteURL(resp.Request.URL.String(), string(body))
	if siteURL == "" {
		return "", "", errors.New("未解析到最新面板域名")
	}
	return siteURL, string(body), nil
}

func (h *ProxyHandler) loginManagedSubscription(client *http.Client, siteURL string) error {
	loginURL := joinURL(siteURL, h.subRefresh.loginPath)
	if loginURL == "" {
		return errors.New("登录地址无效")
	}
	if _, err := client.Get(loginURL); err != nil {
		return err
	}

	form := url.Values{}
	form.Set("email", h.subRefresh.loginEmail)
	form.Set("passwd", h.subRefresh.loginPassword)
	form.Set("remember_me", "1")
	form.Set("code", "")

	req, err := http.NewRequest(http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", loginURL)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return err
	}
	var payload struct {
		Ret    int    `json:"ret"`
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("登录响应解析失败: %w", err)
	}
	if payload.Ret == 1 || payload.Status == 1 {
		return nil
	}
	msg := strings.TrimSpace(payload.Msg)
	if msg == "" {
		msg = "登录失败"
	}
	return errors.New(msg)
}

func (h *ProxyHandler) fetchManagedSubscriptionLink(client *http.Client, siteURL string) (string, string, error) {
	userURL := joinURL(siteURL, h.subRefresh.userPath)
	if userURL == "" {
		return "", "", errors.New("用户中心地址无效")
	}
	resp, err := client.Get(userURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return "", "", err
	}
	html := string(body)
	urls := extractManagedSubscribeURLs(html)
	subscribeURL := selectPreferredSubscriptionURL(urls, h.subRefresh.preferredSubType)
	if subscribeURL == "" {
		return "", html, errors.New("未在用户中心解析到订阅链接")
	}
	if strings.HasPrefix(subscribeURL, "http://") {
		if httpsURL := "https://" + strings.TrimPrefix(subscribeURL, "http://"); strings.Contains(httpsURL, "/link/") {
			subscribeURL = httpsURL
		}
	}
	return subscribeURL, html, nil
}

func chooseReachableNode(nodes []ProxyNode) (*ProxyNode, int64) {
	reachable := collectReachableNodes(nodes, proxyProbeConcurrency, func(n ProxyNode) int64 {
		return checkNodeReachability(&n, probeURL)
	})
	if len(reachable) == 0 {
		return nil, -1
	}
	best := reachable[0]
	for _, item := range reachable[1:] {
		if item.latency < best.latency {
			best = item
		}
	}
	best.node.Latency = best.latency
	return &best.node, best.latency
}

func (h *ProxyHandler) applyManagedSubscription(sourceURL, yamlText string, nodes []ProxyNode) (bool, error) {
	globalSession.mu.RLock()
	currentURLs := append([]string(nil), globalSession.sourceURLs...)
	currentYAML := globalSession.yamlContent
	active := cloneNode(globalSession.active)
	globalSession.mu.RUnlock()

	if !managedSubscriptionChanged(currentURLs, currentYAML, sourceURL, yamlText) {
		return false, nil
	}

	nextNode := func() *ProxyNode {
		if active == nil {
			return nil
		}
		candidate := findNodeByName(nodes, active.Name)
		if candidate == nil {
			return nil
		}
		if latency := checkNodeReachability(candidate, probeURL); latency >= 0 {
			candidate.Latency = latency
			return candidate
		}
		return nil
	}()

	nodeHint := ""
	if nextNode == nil {
		bestNode, _ := chooseReachableNode(nodes)
		if bestNode == nil {
			return false, errors.New("新订阅测试失败：没有可用节点")
		}
		nextNode = bestNode
	}
	nodeHint = nextNode.Name

	globalSession.mu.Lock()
	globalSession.nodes = nodes
	globalSession.sourceURL = sourceURL
	globalSession.sourceURLs = normalizeSourceURLs([]string{sourceURL}, "")
	globalSession.yamlContent = yamlText
	globalSession.mu.Unlock()
	go savePersistedProxy()

	if _, err := startProxyListener(nextNode, h.adminPassword, h.localPort); err != nil {
		return false, err
	}
	h.startNPC()
	log.Printf("proxy: 托管订阅已接管，订阅 %s，线路 %s", maskURL(sourceURL), nodeHint)
	return true, nil
}

func (h *ProxyHandler) refreshManagedSubscriptionLocked() {
	attempts := make([]managedRefreshAttempt, 0, 4)
	recordAttempt := func(attempt managedRefreshAttempt) {
		attempts = append(attempts, attempt)
	}

	if !h.subRefresh.enabled {
		return
	}
	if strings.TrimSpace(h.subRefresh.loginEmail) == "" || strings.TrimSpace(h.subRefresh.loginPassword) == "" {
		status := "托管订阅未执行：缺少登录账号或密码"
		recordAttempt(managedRefreshAttempt{
			Source:  "managed",
			Stage:   "precheck",
			Success: false,
			Error:   status,
		})
		h.markSubscriptionState(status, "", "", "managed", "", false, false)
		h.sendManagedRefreshReport("代理订阅刷新失败", status, attempts, map[string]interface{}{
			"enabled":          h.subRefresh.enabled,
			"landing_url":      h.subRefresh.landingURL,
			"configured_proxy": h.subRefresh.requestProxyURL,
		})
		return
	}

	var lastErr error
	for _, candidate := range h.managedRefreshClients(20 * time.Second) {
		attempt := managedRefreshAttempt{
			Source: candidate.source,
			Stage:  "resolve_site",
		}

		siteURL, _, err := h.resolveManagedSiteURL(candidate.client)
		if err != nil {
			lastErr = fmt.Errorf("%s: 解析站点失败: %w", candidate.source, err)
			log.Printf("proxy: 托管订阅解析站点失败(%s): %v", candidate.source, err)
			attempt.Error = err.Error()
			recordAttempt(attempt)
			continue
		}
		attempt.SiteURL = siteURL

		if err := h.loginManagedSubscription(candidate.client, siteURL); err != nil {
			lastErr = fmt.Errorf("%s: 登录失败: %w", candidate.source, err)
			log.Printf("proxy: 托管订阅登录失败(%s): %v", candidate.source, err)
			attempt.Stage = "login"
			attempt.Error = err.Error()
			recordAttempt(attempt)
			continue
		}

		subscribeURL, _, err := h.fetchManagedSubscriptionLink(candidate.client, siteURL)
		if err != nil {
			lastErr = fmt.Errorf("%s: 提取订阅链接失败: %w", candidate.source, err)
			log.Printf("proxy: 托管订阅提取订阅链接失败(%s): %v", candidate.source, err)
			attempt.Stage = "fetch_subscription_link"
			attempt.Error = err.Error()
			recordAttempt(attempt)
			continue
		}
		attempt.SubscribeURL = subscribeURL

		yamlText, err := fetchSubscriptionWithClient(candidate.client, subscribeURL)
		if err != nil {
			lastErr = fmt.Errorf("%s: 下载新订阅失败: %w", candidate.source, err)
			log.Printf("proxy: 托管订阅下载失败(%s): %v", candidate.source, err)
			attempt.Stage = "download_subscription"
			attempt.Error = err.Error()
			recordAttempt(attempt)
			continue
		}
		nodes, err := parseClashYAML(yamlText)
		if err != nil {
			lastErr = fmt.Errorf("%s: 解析新订阅失败: %w", candidate.source, err)
			log.Printf("proxy: 托管订阅解析失败(%s): %v", candidate.source, err)
			attempt.Stage = "parse_subscription"
			attempt.Error = err.Error()
			recordAttempt(attempt)
			continue
		}
		if len(nodes) == 0 {
			lastErr = fmt.Errorf("%s: 新订阅节点为空", candidate.source)
			attempt.Stage = "parse_subscription"
			attempt.Error = lastErr.Error()
			recordAttempt(attempt)
			continue
		}
		attempt.NodeCount = len(nodes)

		applied, err := h.applyManagedSubscription(subscribeURL, yamlText, nodes)
		if err != nil {
			status := "托管订阅已发现更新，但新订阅测试失败"
			attempt.Stage = "apply_subscription"
			attempt.Error = err.Error()
			attempt.Changed = true
			recordAttempt(attempt)
			h.markSubscriptionState(status, siteURL, subscribeURL, candidate.source, "", true, false)
			log.Printf("proxy: 托管订阅接管失败(%s): %v", candidate.source, err)
			h.sendManagedRefreshReport("代理订阅刷新失败", status, attempts, map[string]interface{}{
				"resolved_site_url": siteURL,
				"subscribe_url":     subscribeURL,
				"refresh_source":    candidate.source,
				"node_count":        len(nodes),
			})
			return
		}
		if !applied {
			status := "托管订阅检查完成：无变化，保持当前线路"
			attempt.Stage = "apply_subscription"
			attempt.Success = true
			recordAttempt(attempt)
			h.markSubscriptionState(status, siteURL, subscribeURL, candidate.source, "", false, false)
			h.sendManagedRefreshReport("代理订阅刷新成功", status, attempts, map[string]interface{}{
				"resolved_site_url": siteURL,
				"subscribe_url":     subscribeURL,
				"refresh_source":    candidate.source,
				"node_count":        len(nodes),
			})
			return
		}
		activeName := ""
		globalSession.mu.RLock()
		if globalSession.active != nil {
			activeName = globalSession.active.Name
		}
		globalSession.mu.RUnlock()
		status := "托管订阅已更新并接管"
		attempt.Stage = "apply_subscription"
		attempt.Success = true
		attempt.Changed = true
		attempt.Applied = true
		attempt.NodeHint = activeName
		recordAttempt(attempt)
		h.markSubscriptionState(status, siteURL, subscribeURL, candidate.source, activeName, true, true)
		h.sendManagedRefreshReport("代理订阅刷新成功", status, attempts, map[string]interface{}{
			"resolved_site_url": siteURL,
			"subscribe_url":     subscribeURL,
			"refresh_source":    candidate.source,
			"active_node":       activeName,
			"node_count":        len(nodes),
		})
		return
	}

	status := "托管订阅未执行：全部刷新链路失败"
	if lastErr != nil {
		status = "托管订阅未执行：" + lastErr.Error()
	}
	h.markSubscriptionState(status, "", "", "managed", "", false, false)
	h.sendManagedRefreshReport("代理订阅刷新失败", status, attempts, map[string]interface{}{
		"landing_url":       h.subRefresh.landingURL,
		"site_url":          h.subRefresh.siteURL,
		"configured_proxy":  h.subRefresh.requestProxyURL,
		"preferred_subtype": h.subRefresh.preferredSubType,
	})
}

func (h *ProxyHandler) refreshManagedSubscription() {
	h.subRefreshMu.Lock()
	defer h.subRefreshMu.Unlock()
	h.refreshManagedSubscriptionLocked()
}

// TriggerSubscriptionRefresh POST /api/proxy/subscription-refresh
func (h *ProxyHandler) TriggerSubscriptionRefresh(c *gin.Context) {
	var req struct {
		AdminPassword string `json:"admin_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}
	if !h.checkAdmin(req.AdminPassword) {
		c.JSON(http.StatusForbidden, gin.H{"error": "密码错误"})
		return
	}
	if !h.subRefresh.enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "托管订阅自动刷新未启用"})
		return
	}

	if !h.subRefreshMu.TryLock() {
		subState := h.snapshotSubscriptionState()
		c.JSON(http.StatusConflict, gin.H{
			"error": "托管订阅刷新正在执行中",
			"subscription_refresh": gin.H{
				"enabled":                subState.Enabled,
				"resolved_site_url":      subState.ResolvedSiteURL,
				"last_subscribe_url":     subState.LastSubscribeURL,
				"last_refresh_status":    subState.LastRefreshStatus,
				"last_refresh_at":        subState.LastRefreshAt,
				"last_refresh_changed":   subState.LastRefreshChanged,
				"last_refresh_applied":   subState.LastRefreshApplied,
				"last_refresh_source":    subState.LastRefreshSource,
				"last_refresh_node_hint": subState.LastRefreshNodeHint,
			},
		})
		return
	}
	h.markSubscriptionState("手动刷新已启动，正在后台执行", "", "", "manual", "", false, false)
	go func() {
		defer h.subRefreshMu.Unlock()
		h.refreshManagedSubscriptionLocked()
	}()

	globalSession.mu.RLock()
	activeName := ""
	if globalSession.active != nil {
		activeName = globalSession.active.Name
	}
	globalSession.mu.RUnlock()
	defaultNode, aiNode := resolveConfiguredNodes()

	subState := h.snapshotSubscriptionState()
	resp := gin.H{
		"ok":      true,
		"started": true,
		"node":    activeName,
		"subscription_refresh": gin.H{
			"enabled":                subState.Enabled,
			"resolved_site_url":      subState.ResolvedSiteURL,
			"last_subscribe_url":     subState.LastSubscribeURL,
			"last_refresh_status":    subState.LastRefreshStatus,
			"last_refresh_at":        subState.LastRefreshAt,
			"last_refresh_changed":   subState.LastRefreshChanged,
			"last_refresh_applied":   subState.LastRefreshApplied,
			"last_refresh_source":    subState.LastRefreshSource,
			"last_refresh_node_hint": subState.LastRefreshNodeHint,
		},
	}
	if defaultNode != nil {
		resp["default_node"] = defaultNode.Name
	}
	if aiNode != nil {
		resp["effective_ai_node"] = aiNode.Name
	}
	c.JSON(http.StatusAccepted, resp)
}

// AutoSelectOnStartup 启动时若有持久化节点但无活跃节点，后台自动选最优节点
func (h *ProxyHandler) AutoSelectOnStartup() {
	globalSession.mu.RLock()
	active := globalSession.active
	nodeCount := len(globalSession.nodes)
	globalSession.mu.RUnlock()

	if active != nil || nodeCount == 0 {
		return
	}

	go func() {
		globalSession.mu.RLock()
		nodes := make([]ProxyNode, len(globalSession.nodes))
		copy(nodes, globalSession.nodes)
		globalSession.mu.RUnlock()

		reachable := collectReachableNodes(nodes, proxyProbeConcurrency, func(n ProxyNode) int64 {
			return tcpPing(n.Server, n.Port)
		})
		if len(reachable) == 0 {
			return
		}

		best := reachable[0]
		for _, item := range reachable[1:] {
			if item.latency < best.latency {
				best = item
			}
		}
		best.node.Latency = best.latency
		startProxyListener(&best.node, h.adminPassword, h.localPort) //nolint
		h.startNPC()
		log.Printf("proxy: 启动自动选节点 %s (延迟 %dms)", best.node.Name, best.latency)
	}()
}

func (h *ProxyHandler) checkAdmin(password string) bool {
	return h.adminPassword != "" && password == h.adminPassword
}

const probeURL = "https://www.google.com"

type proxyNodeReachability struct {
	node    ProxyNode
	latency int64
}

func collectReachableNodes(nodes []ProxyNode, limit int, checker func(ProxyNode) int64) []proxyNodeReachability {
	if len(nodes) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = proxyProbeConcurrency
	}

	results := make([]proxyNodeReachability, len(nodes))
	var wg sync.WaitGroup
	sem := make(chan struct{}, minInt(limit, len(nodes)))

	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, n ProxyNode) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = proxyNodeReachability{
				node:    n,
				latency: checker(n),
			}
		}(i, node)
	}
	wg.Wait()

	reachable := make([]proxyNodeReachability, 0, len(results))
	for _, item := range results {
		if item.latency >= 0 {
			reachable = append(reachable, item)
		}
	}
	return reachable
}

// probeActive 检测当前活跃节点是否真实可用，返回 true=可用
func probeActive() bool {
	globalSession.mu.RLock()
	node := globalSession.active
	globalSession.mu.RUnlock()
	if node == nil {
		return false
	}
	return checkNodeReachability(node, probeURL) >= 0
}

// switchToBestNode 测速所有节点，选延迟最低且真实可用的节点启动代理
func (h *ProxyHandler) switchToBestNode() {
	globalSession.mu.RLock()
	nodes := make([]ProxyNode, len(globalSession.nodes))
	copy(nodes, globalSession.nodes)
	globalSession.mu.RUnlock()

	if len(nodes) == 0 {
		return
	}

	reachable := collectReachableNodes(nodes, proxyProbeConcurrency, func(n ProxyNode) int64 {
		return checkNodeReachability(&n, probeURL)
	})
	if len(reachable) == 0 {
		log.Printf("proxy: 自动切换失败，所有节点均不可用")
		go h.sendAlert("代理节点全部不可用", fmt.Sprintf("时间：%s\n节点总数：%d\n所有节点均无法连接，请检查订阅或节点状态。", time.Now().Format("2006-01-02 15:04:05"), len(nodes)))
		return
	}
	best := reachable[0]
	for _, item := range reachable[1:] {
		if item.latency < best.latency {
			best = item
		}
	}
	best.node.Latency = best.latency
	if _, err := startProxyListener(&best.node, h.adminPassword, h.localPort); err != nil {
		log.Printf("proxy: 切换节点失败: %v", err)
		return
	}
	h.startNPC()
	log.Printf("proxy: 已切换到节点 %s（延迟 %dms）", best.node.Name, best.latency)
}

// refreshSubscription 重新拉取订阅 URL，更新节点列表
func (h *ProxyHandler) refreshSubscription() {
	globalSession.mu.RLock()
	sourceURLs := append([]string(nil), globalSession.sourceURLs...)
	globalSession.mu.RUnlock()

	if len(sourceURLs) == 0 {
		return
	}
	text, nodes, err := fetchSubscriptions(sourceURLs)
	if err != nil {
		log.Printf("proxy: 订阅更新失败: %v", err)
		return
	}
	if len(nodes) == 0 {
		log.Printf("proxy: 订阅解析失败或节点为空")
		return
	}
	globalSession.mu.Lock()
	globalSession.nodes = nodes
	globalSession.yamlContent = text
	globalSession.mu.Unlock()
	go savePersistedProxy()
	log.Printf("proxy: 订阅已更新，共 %d 个节点，来源 %d 个 URL", len(nodes), len(sourceURLs))
}

// StartAutoMaintenance 启动后台维护：定期更新订阅 + 定期探测切换
func (h *ProxyHandler) StartAutoMaintenance() {
	go func() {
		subInterval := 6 * time.Hour
		if h.subRefresh.enabled {
			subInterval = h.subRefresh.interval
		}
		subTicker := time.NewTicker(subInterval)
		// 节点探测：每 10 分钟
		probeTicker := time.NewTicker(10 * time.Minute)
		defer subTicker.Stop()
		defer probeTicker.Stop()

		if h.subRefresh.enabled {
			h.refreshManagedSubscription()
		}

		for {
			select {
			case <-subTicker.C:
				if h.subRefresh.enabled {
					h.refreshManagedSubscription()
				} else {
					h.refreshSubscription()
				}
				// 更新订阅后立即探测，若当前节点不可用则切换
				if !probeActive() {
					log.Printf("proxy: 订阅更新后探测失败，触发切换")
					h.switchToBestNode()
				}
			case <-probeTicker.C:
				globalSession.mu.RLock()
				active := globalSession.active
				globalSession.mu.RUnlock()
				if active == nil {
					continue
				}
				if !probeActive() {
					log.Printf("proxy: 节点 %s 探测失败，触发自动切换", active.Name)
					h.switchToBestNode()
				}
			}
		}
	}()
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
	return fetchSubscriptionWithClient(newDirectHTTPClient(15*time.Second), rawURL)
}

func fetchSubscriptionWithClient(client *http.Client, rawURL string) (string, error) {
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
		AdminPassword    string   `json:"admin_password"`
		SourceURL        string   `json:"source_url"`
		SourceURLs       []string `json:"source_urls"`
		YAMLContent      string   `json:"yaml_content"`
		RoutingMode      string   `json:"routing_mode"`
		DefaultNodeName  string   `json:"default_node_name"`
		DefaultNodeRegex string   `json:"default_node_regex"`
		AINodeName       string   `json:"ai_node_name"`
		AINodeRegex      string   `json:"ai_node_regex"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	if !h.checkAdmin(req.AdminPassword) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}

	sourceURLs := normalizeSourceURLs(req.SourceURLs, req.SourceURL)
	routingMode := normalizeProxyRouteMode(req.RoutingMode)
	defaultNodeName := strings.TrimSpace(req.DefaultNodeName)
	defaultNodeRegex := strings.TrimSpace(req.DefaultNodeRegex)
	aiNodeName := strings.TrimSpace(req.AINodeName)
	aiNodeRegex := strings.TrimSpace(req.AINodeRegex)

	for _, pattern := range []struct {
		label string
		value string
	}{
		{label: "默认线路正则", value: defaultNodeRegex},
		{label: "AI 专线正则", value: aiNodeRegex},
	} {
		if pattern.value == "" {
			continue
		}
		if _, err := regexp.Compile(pattern.value); err != nil {
			c.JSON(400, gin.H{"error": pattern.label + "非法: " + err.Error()})
			return
		}
	}

	var (
		yamlText string
		nodes    []ProxyNode
		err      error
	)
	if len(sourceURLs) > 0 {
		yamlText, nodes, err = fetchSubscriptions(sourceURLs)
		if err != nil {
			c.JSON(500, gin.H{"error": "下载订阅失败: " + err.Error()})
			return
		}
	} else {
		yamlText = req.YAMLContent
		nodes, err = parseClashYAML(yamlText)
		if err != nil {
			c.JSON(400, gin.H{"error": "解析配置失败: " + err.Error()})
			return
		}
	}
	if len(nodes) == 0 {
		c.JSON(400, gin.H{"error": "未找到任何节点"})
		return
	}
	if defaultNodeName != "" {
		found := false
		for i := range nodes {
			if nodes[i].Name == defaultNodeName {
				found = true
				break
			}
		}
		if !found {
			c.JSON(400, gin.H{"error": "手动默认线路不存在"})
			return
		}
	}
	if aiNodeName != "" {
		found := false
		for i := range nodes {
			if nodes[i].Name == aiNodeName {
				found = true
				break
			}
		}
		if !found {
			c.JSON(400, gin.H{"error": "手动 AI 线路不存在"})
			return
		}
	}

	globalSession.mu.Lock()
	globalSession.nodes = nodes
	globalSession.sourceURL = firstSourceURL(sourceURLs, req.SourceURL)
	globalSession.sourceURLs = sourceURLs
	globalSession.yamlContent = yamlText
	globalSession.routingMode = routingMode
	globalSession.defaultNodeName = defaultNodeName
	globalSession.defaultNodeRegex = defaultNodeRegex
	globalSession.aiNodeName = aiNodeName
	globalSession.aiNodeRegex = aiNodeRegex
	globalSession.mu.Unlock()

	go savePersistedProxy()
	configuredDefaultNode, configuredAINode := resolveConfiguredNodes()

	c.JSON(200, gin.H{
		"nodes":              nodes,
		"count":              len(nodes),
		"source_url":         firstSourceURL(sourceURLs, req.SourceURL),
		"source_urls":        sourceURLs,
		"routing_mode":       routingMode,
		"default_node_name":  defaultNodeName,
		"default_node_regex": defaultNodeRegex,
		"ai_node_name":       aiNodeName,
		"ai_node_regex":      aiNodeRegex,
		"default_node": func() string {
			if configuredDefaultNode != nil {
				return configuredDefaultNode.Name
			}
			return ""
		}(),
		"effective_ai_node": func() string {
			if configuredAINode != nil {
				return configuredAINode.Name
			}
			return ""
		}(),
	})
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

	results := make([]ProxyNode, len(nodes))
	reachable := collectReachableNodes(nodes, proxyProbeConcurrency, func(n ProxyNode) int64 {
		return tcpPing(n.Server, n.Port)
	})
	latencyMap := make(map[string]int64, len(reachable))
	for _, item := range reachable {
		latencyMap[proxyNodeKey(item.node)] = item.latency
	}
	for i, node := range nodes {
		node.Latency = -1
		if latency, ok := latencyMap[proxyNodeKey(node)]; ok {
			node.Latency = latency
		}
		results[i] = node
	}

	globalSession.mu.Lock()
	globalSession.nodes = results
	globalSession.mu.Unlock()

	go savePersistedProxy()

	c.JSON(200, gin.H{"results": results})
}

// tcpPing 测量 TCP 连接延迟（ms），失败返回 -1
func tcpPing(host string, port int) int64 {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return -1
	}
	conn.Close()
	return time.Since(start).Milliseconds()
}

// checkNodeReachability 通过节点实际发 HTTP 请求到 testURL，验证代理真实可用性
// 返回延迟 ms，失败返回 -1
func checkNodeReachability(node *ProxyNode, testURL string) int64 {
	// 临时启动一个无密码代理监听
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return -1
	}
	defer ln.Close()

	go serveHTTPProxy(ln, node, "") // 无密码，内部调用，持续 accept

	port := ln.Addr().(*net.TCPAddr).Port
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	resp, err := client.Get(testURL)
	if err != nil {
		return -1
	}
	resp.Body.Close()
	return time.Since(start).Milliseconds()
}

// CheckNodes POST /api/proxy/check
// 对节点列表做真实可用性检测（通过节点实际访问 google.com）
func (h *ProxyHandler) CheckNodes(c *gin.Context) {
	var req struct {
		AdminPassword string   `json:"admin_password"`
		NodeNames     []string `json:"node_names"` // 空则检测所有节点
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
	allNodes := make([]ProxyNode, len(globalSession.nodes))
	copy(allNodes, globalSession.nodes)
	globalSession.mu.RUnlock()

	if len(allNodes) == 0 {
		c.JSON(400, gin.H{"error": "请先加载节点配置"})
		return
	}

	nameSet := make(map[string]bool, len(req.NodeNames))
	for _, n := range req.NodeNames {
		nameSet[n] = true
	}
	targets := allNodes
	if len(nameSet) > 0 {
		targets = targets[:0]
		for _, n := range allNodes {
			if nameSet[n.Name] {
				targets = append(targets, n)
			}
		}
	}
	if len(targets) == 0 {
		c.JSON(400, gin.H{"error": "未匹配到待检测节点"})
		return
	}

	if !h.checkMu.TryLock() {
		state := h.snapshotCheckState()
		c.JSON(http.StatusConflict, gin.H{
			"error": "节点可用性检测正在执行中",
			"check_reachability": gin.H{
				"running":           state.Running,
				"last_check_status": state.LastCheckStatus,
				"last_check_at":     state.LastCheckAt,
				"results":           state.Results,
			},
		})
		return
	}

	h.markCheckState(fmt.Sprintf("节点可用性检测已启动，目标 %d 个", len(targets)), true, nil)
	go func(targets []ProxyNode) {
		defer h.checkMu.Unlock()
		h.runCheckNodesLocked(targets)
	}(append([]ProxyNode(nil), targets...))

	state := h.snapshotCheckState()
	c.JSON(http.StatusAccepted, gin.H{
		"ok":      true,
		"started": true,
		"check_reachability": gin.H{
			"running":           state.Running,
			"last_check_status": state.LastCheckStatus,
			"last_check_at":     state.LastCheckAt,
			"results":           state.Results,
		},
	})
}

func (h *ProxyHandler) runCheckNodesLocked(targets []ProxyNode) {
	if len(targets) == 0 {
		h.markCheckState("节点可用性检测未执行：没有待检测节点", false, []proxyReachabilityResult{})
		return
	}

	const testURL = "https://www.google.com"
	results := make([]proxyReachabilityResult, len(targets))
	reachable := collectReachableNodes(targets, proxyProbeConcurrency, func(n ProxyNode) int64 {
		return checkNodeReachability(&n, testURL)
	})
	latencyMap := make(map[string]int64, len(reachable))
	for _, item := range reachable {
		latencyMap[proxyNodeKey(item.node)] = item.latency
	}
	okCount := 0
	for i, node := range targets {
		latency := int64(-1)
		if v, ok := latencyMap[proxyNodeKey(node)]; ok {
			latency = v
		}
		results[i] = proxyReachabilityResult{
			Name:      node.Name,
			Reachable: latency >= 0,
			Latency:   latency,
		}
		if latency >= 0 {
			okCount++
		}
	}
	h.markCheckState(fmt.Sprintf("节点可用性检测完成：%d/%d 可用", okCount, len(targets)), false, results)
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

	// 停止旧代理（startProxyListener 内部会处理，这里保留以便提前释放）
	ln, err := startProxyListener(target, h.adminPassword, h.localPort)
	if err != nil {
		c.JSON(500, gin.H{"error": "启动失败: " + err.Error()})
		return
	}

	port := ln.Addr().(*net.TCPAddr).Port
	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	c.JSON(200, gin.H{
		"http_port": port,
		"proxy_url": proxyURL,
		"node":      target.Name,
	})
	h.startNPC()
}

// triggerAutoSelect 若当前无活跃节点且有节点列表，后台异步测速并选最优节点
// 不阻塞调用方，当前请求继续直连
var autoSelectMu sync.Mutex

func (h *ProxyHandler) triggerAutoSelect() {
	globalSession.mu.RLock()
	active := globalSession.active
	nodeCount := len(globalSession.nodes)
	globalSession.mu.RUnlock()

	if active != nil || nodeCount == 0 {
		return
	}

	autoSelectMu.Lock()
	defer autoSelectMu.Unlock()

	// 再次检查，避免并发重复触发
	globalSession.mu.RLock()
	active = globalSession.active
	globalSession.mu.RUnlock()
	if active != nil {
		return
	}

	go func() {
		globalSession.mu.RLock()
		nodes := make([]ProxyNode, len(globalSession.nodes))
		copy(nodes, globalSession.nodes)
		globalSession.mu.RUnlock()

		reachable := collectReachableNodes(nodes, proxyProbeConcurrency, func(n ProxyNode) int64 {
			return tcpPing(n.Server, n.Port)
		})
		if len(reachable) == 0 {
			return
		}
		best := reachable[0]
		for _, item := range reachable[1:] {
			if item.latency < best.latency {
				best = item
			}
		}
		best.node.Latency = best.latency
		startProxyListener(&best.node, h.adminPassword, h.localPort) //nolint
		h.startNPC()
	}()
}

// startProxyListener 启动 HTTP CONNECT 代理监听，返回 listener
// localPort 非空时固定监听 127.0.0.1:localPort，否则随机端口
func startProxyListener(target *ProxyNode, adminPassword, localPort string) (net.Listener, error) {
	addr := "127.0.0.1:0"
	if localPort != "" {
		addr = "127.0.0.1:" + localPort
	}
	// 关闭旧 listener（固定端口时必须先释放）
	globalSession.mu.Lock()
	if globalSession.listener != nil {
		globalSession.listener.Close()
		globalSession.listener = nil
	}
	globalSession.mu.Unlock()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	globalSession.mu.Lock()
	globalSession.listener = ln
	globalSession.active = target
	globalSession.mu.Unlock()
	go serveHTTPProxy(ln, target, adminPassword)
	return ln, nil
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

// CreateNPSTunnel POST /api/proxy/nps-tunnel
// 用 proxy 密码一键将 tunnel_port 映射到 NPS 公网，自动分配公网端口
func (h *ProxyHandler) CreateNPSTunnel(c *gin.Context) {
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
	if h.npsHandler == nil || h.npsHandler.cfg.ServerURL == "" {
		c.JSON(400, gin.H{"error": "NPS 未配置"})
		return
	}
	if h.npcTunnelPort == "" {
		c.JSON(400, gin.H{"error": "proxy.tunnel_port 未配置"})
		return
	}

	clientID, err := h.npsHandler.getClientID()
	if err != nil {
		c.JSON(500, gin.H{"error": "获取 NPS 客户端失败: " + err.Error()})
		return
	}

	params := url.Values{}
	params.Set("type", "tcp")
	params.Set("port", h.npcTunnelPort)                // 公网端口 = tunnel_port（如 18080）
	params.Set("target", "127.0.0.1:"+h.npcTunnelPort) // target = 本地代理端口
	params.Set("client_id", strconv.Itoa(clientID))
	params.Set("remark", "代理端口（防探测）")
	result, err := h.npsHandler.npsPost("/index/add/", params)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if status, _ := result["status"].(float64); status != 1 {
		msg, _ := result["msg"].(string)
		c.JSON(400, gin.H{"error": msg})
		return
	}
	tunnelPort, _ := strconv.Atoi(h.npcTunnelPort)
	c.JSON(200, gin.H{"port": tunnelPort})
}

// AutoStart POST /api/proxy/auto-start
// 测速后自动选延迟最低的可用节点启动代理
func (h *ProxyHandler) AutoStart(c *gin.Context) {
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
		c.JSON(400, gin.H{"error": "请先加载节点配置"})
		return
	}

	results := make([]ProxyNode, len(nodes))
	reachable := collectReachableNodes(nodes, proxyProbeConcurrency, func(n ProxyNode) int64 {
		return checkNodeReachability(&n, probeURL)
	})
	latencyMap := make(map[string]int64, len(reachable))
	for _, item := range reachable {
		latencyMap[proxyNodeKey(item.node)] = item.latency
	}
	for i, node := range nodes {
		node.Latency = -1
		if latency, ok := latencyMap[proxyNodeKey(node)]; ok {
			node.Latency = latency
		}
		results[i] = node
	}

	// 保存测速结果
	globalSession.mu.Lock()
	globalSession.nodes = results
	globalSession.mu.Unlock()
	go savePersistedProxy()

	// 选延迟最低的可用节点
	var best *ProxyNode
	for i := range results {
		if results[i].Latency < 0 {
			continue
		}
		if best == nil || results[i].Latency < best.Latency {
			best = &results[i]
		}
	}
	if best == nil {
		c.JSON(400, gin.H{"error": "所有节点均不可用", "results": results})
		return
	}

	// 启动代理
	ln, err := startProxyListener(best, h.adminPassword, h.localPort)
	if err != nil {
		c.JSON(500, gin.H{"error": "启动失败: " + err.Error()})
		return
	}
	port := ln.Addr().(*net.TCPAddr).Port
	c.JSON(200, gin.H{
		"node":      best.Name,
		"latency":   best.Latency,
		"http_port": port,
		"proxy_url": fmt.Sprintf("http://127.0.0.1:%d", port),
		"results":   results,
	})
	h.startNPC()
}

// ListCustomDomains GET /api/proxy/custom-domains
func (h *ProxyHandler) ListCustomDomains(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(200, gin.H{"domains": ListCustomDomains()})
}

// AddCustomDomain POST /api/proxy/custom-domains  body: {"domain":"example.com"}
func (h *ProxyHandler) AddCustomDomain(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Domain == "" {
		c.JSON(400, gin.H{"error": "domain required"})
		return
	}
	if err := AddCustomDomain(req.Domain); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// RemoveCustomDomain DELETE /api/proxy/custom-domains  body: {"domain":"example.com"}
func (h *ProxyHandler) RemoveCustomDomain(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Domain == "" {
		c.JSON(400, gin.H{"error": "domain required"})
		return
	}
	if err := RemoveCustomDomain(req.Domain); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
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
	subState := h.snapshotSubscriptionState()
	checkState := h.snapshotCheckState()

	resp := gin.H{
		"nodes":              globalSession.nodes,
		"source_url":         firstSourceURL(globalSession.sourceURLs, globalSession.sourceURL),
		"source_urls":        append([]string(nil), globalSession.sourceURLs...),
		"yaml_content":       globalSession.yamlContent,
		"routing_mode":       normalizeProxyRouteMode(globalSession.routingMode),
		"default_node_name":  globalSession.defaultNodeName,
		"default_node_regex": globalSession.defaultNodeRegex,
		"ai_node_name":       globalSession.aiNodeName,
		"ai_node_regex":      globalSession.aiNodeRegex,
		"running":            false,
		"npc_running":        npcRunning,
		"npc_tunnel_port":    h.npcTunnelPort,
		"npc_server_addr":    h.npcCfg.serverAddr,
		"subscription_refresh": gin.H{
			"enabled":                subState.Enabled,
			"resolved_site_url":      subState.ResolvedSiteURL,
			"last_subscribe_url":     subState.LastSubscribeURL,
			"last_refresh_status":    subState.LastRefreshStatus,
			"last_refresh_at":        subState.LastRefreshAt,
			"last_refresh_changed":   subState.LastRefreshChanged,
			"last_refresh_applied":   subState.LastRefreshApplied,
			"last_refresh_source":    subState.LastRefreshSource,
			"last_refresh_node_hint": subState.LastRefreshNodeHint,
		},
		"check_reachability": gin.H{
			"running":           checkState.Running,
			"last_check_status": checkState.LastCheckStatus,
			"last_check_at":     checkState.LastCheckAt,
			"results":           checkState.Results,
		},
	}
	defaultNode, aiNode := resolveConfiguredNodes()
	if defaultNode != nil {
		resp["default_node"] = defaultNode.Name
	}
	if aiNode != nil {
		resp["effective_ai_node"] = aiNode.Name
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
		// 带上代理认证，避免被自己的密码保护拦截
		if globalProxyHandler != nil && globalProxyHandler.adminPassword != "" {
			pu.User = url.UserPassword("proxy", globalProxyHandler.adminPassword)
		}
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

	// 防探测：验证 Proxy-Authorization，失败时根据请求类型响应
	if adminPassword != "" {
		authHeader := req.Header.Get("Proxy-Authorization")
		if !checkProxyAuthHeader(authHeader, adminPassword) {
			if req.Method == http.MethodConnect {
				// CONNECT 必须返回 407，系统/浏览器才会重试带认证头
				clientConn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"proxy\"\r\n\r\n"))
			} else {
				// 非 CONNECT（端口扫描/探测）返回 200 伪装
				clientConn.Write([]byte(fakeWebPage))
			}
			return
		}
	}

	clientConn.SetDeadline(time.Time{}) // 认证通过后取消超时

	if req.Method == http.MethodConnect {
		// HTTPS CONNECT 隧道
		host := req.Host
		upstream, err := dialWithGFW(node, host)
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
		upstream, err := dialWithGFW(node, host)
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
	nodeAddr := net.JoinHostPort(node.Server, strconv.Itoa(node.Port))

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

	case "trojan":
		return dialTrojan(nodeAddr, targetHost, node)

	default:
		// ss/vmess/trojan 等：直连目标（降级，仅测速可用）
		return net.DialTimeout("tcp", targetHost, 10*time.Second)
	}
}

func stripPort(targetHost string) string {
	host := strings.TrimSpace(targetHost)
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]") {
		trimmed := strings.TrimPrefix(host, "[")
		if idx := strings.Index(trimmed, "]"); idx >= 0 {
			return strings.ToLower(strings.TrimSuffix(trimmed[:idx], "."))
		}
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(strings.TrimSuffix(h, "."))
	}
	if idx := strings.LastIndex(host, ":"); idx > 0 && !strings.Contains(host[:idx], ":") {
		host = host[:idx]
	}
	return strings.ToLower(strings.TrimSuffix(host, "."))
}

func isLocalTargetHost(targetHost string) bool {
	host := stripPort(targetHost)
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	return false
}

func matchesDomainSuffix(host, suffix string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	suffix = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(suffix), "."))
	return host == suffix || strings.HasSuffix(host, "."+suffix)
}

func isAIDedicatedHost(targetHost string) bool {
	host := stripPort(targetHost)
	for _, domain := range aiDedicatedDomains {
		if matchesDomainSuffix(host, domain) {
			return true
		}
	}
	return false
}

func cloneNode(node *ProxyNode) *ProxyNode {
	if node == nil {
		return nil
	}
	cp := *node
	return &cp
}

func findNodeByName(nodes []ProxyNode, name string) *ProxyNode {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for i := range nodes {
		if nodes[i].Name == name {
			return cloneNode(&nodes[i])
		}
	}
	return nil
}

func matchNodeByRegex(nodes []ProxyNode, pattern string, preferred *ProxyNode) *ProxyNode {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return cloneNode(preferred)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return cloneNode(preferred)
	}
	if preferred != nil && re.MatchString(preferred.Name) {
		return cloneNode(preferred)
	}
	for i := range nodes {
		if re.MatchString(nodes[i].Name) {
			return cloneNode(&nodes[i])
		}
	}
	return cloneNode(preferred)
}

func nodeMatchesRegex(node ProxyNode, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(node.Name)
}

func aiHostFamily(targetHost string) string {
	host := stripPort(targetHost)
	switch {
	case matchesDomainSuffix(host, "openai.com"),
		matchesDomainSuffix(host, "api.openai.com"),
		matchesDomainSuffix(host, "chatgpt.com"),
		matchesDomainSuffix(host, "chat.openai.com"),
		matchesDomainSuffix(host, "oaistatic.com"),
		matchesDomainSuffix(host, "oaiusercontent.com"),
		matchesDomainSuffix(host, "auth.openai.com"):
		return "openai"
	case matchesDomainSuffix(host, "claude.ai"),
		matchesDomainSuffix(host, "anthropic.com"),
		matchesDomainSuffix(host, "api.anthropic.com"),
		matchesDomainSuffix(host, "console.anthropic.com"):
		return "claude"
	case matchesDomainSuffix(host, "gemini.google.com"),
		matchesDomainSuffix(host, "generativelanguage.googleapis.com"),
		matchesDomainSuffix(host, "makersuite.google.com"):
		return "gemini"
	default:
		return ""
	}
}

func nodeFamilyScore(name, family string) int {
	lowerName := strings.ToLower(name)
	score := 0

	switch family {
	case "openai":
		if strings.Contains(lowerName, "chatgpt") || strings.Contains(lowerName, "openai") {
			score += 120
		} else if strings.Contains(lowerName, "gpt") {
			score += 90
		}
		if strings.Contains(lowerName, "美国") || strings.Contains(lowerName, "us") || strings.Contains(lowerName, "usa") {
			score += 30
		}
	case "claude":
		if strings.Contains(lowerName, "claude") || strings.Contains(lowerName, "anthropic") {
			score += 120
		}
	case "gemini":
		if strings.Contains(lowerName, "gemini") || strings.Contains(lowerName, "google") {
			score += 120
		}
	}

	if strings.Contains(lowerName, "netflix") || strings.Contains(lowerName, "流媒体") {
		score -= 40
	}

	return score
}

func latencyScore(latency int64) int {
	if latency < 0 {
		return 0
	}
	score := 50 - int(latency/5)
	if score < 0 {
		return 0
	}
	return score
}

func selectBestMatchingNode(nodes []ProxyNode, pattern, targetHost string, preferred *ProxyNode) *ProxyNode {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return cloneNode(preferred)
	}

	family := aiHostFamily(targetHost)
	var best *ProxyNode
	bestScore := -1
	bestLatency := int64(1<<62 - 1)

	for i := range nodes {
		if !nodeMatchesRegex(nodes[i], pattern) {
			continue
		}

		score := nodeFamilyScore(nodes[i].Name, family) + latencyScore(nodes[i].Latency)
		if preferred != nil && proxyNodeKey(nodes[i]) == proxyNodeKey(*preferred) {
			score += 1000
		}

		latency := nodes[i].Latency
		if latency < 0 {
			latency = 1<<62 - 1
		}

		if best == nil || score > bestScore || (score == bestScore && latency < bestLatency) {
			cp := nodes[i]
			best = &cp
			bestScore = score
			bestLatency = latency
		}
	}

	if best != nil {
		return best
	}
	return cloneNode(preferred)
}

func resolveRouteNode(defaultNode *ProxyNode, targetHost string) (*ProxyNode, bool, *ProxyNode) {
	globalSession.mu.RLock()
	mode := normalizeProxyRouteMode(globalSession.routingMode)
	nodes := make([]ProxyNode, len(globalSession.nodes))
	copy(nodes, globalSession.nodes)
	active := cloneNode(globalSession.active)
	defaultNodeName := globalSession.defaultNodeName
	defaultRegex := globalSession.defaultNodeRegex
	aiNodeName := globalSession.aiNodeName
	aiRegex := globalSession.aiNodeRegex
	globalSession.mu.RUnlock()

	if isLocalTargetHost(targetHost) {
		return nil, false, nil
	}

	baseNode := cloneNode(defaultNode)
	manualDefaultNode := findNodeByName(nodes, defaultNodeName)
	if manualDefaultNode != nil {
		baseNode = manualDefaultNode
	}
	if baseNode == nil {
		baseNode = active
	}
	if baseNode == nil && strings.TrimSpace(defaultRegex) != "" {
		baseNode = selectBestMatchingNode(nodes, defaultRegex, targetHost, nil)
	}

	switch mode {
	case proxyRouteModeGlobal:
		return baseNode, true, baseNode
	case proxyRouteModeSmart:
		if ShouldProxy(targetHost) {
			return baseNode, true, baseNode
		}
		return nil, false, baseNode
	default:
		if isAIDedicatedHost(targetHost) {
			if baseNode != nil && nodeMatchesRegex(*baseNode, aiRegex) {
				return baseNode, true, baseNode
			}
			manualAINode := findNodeByName(nodes, aiNodeName)
			if manualAINode != nil {
				return manualAINode, true, baseNode
			}
			aiNode := selectBestMatchingNode(nodes, aiRegex, targetHost, nil)
			if aiNode != nil {
				return aiNode, true, baseNode
			}
			return baseNode, true, baseNode
		}
		if ShouldProxy(targetHost) {
			return baseNode, true, baseNode
		}
		return nil, false, baseNode
	}
}

func resolveConfiguredNodes() (*ProxyNode, *ProxyNode) {
	globalSession.mu.RLock()
	nodes := make([]ProxyNode, len(globalSession.nodes))
	copy(nodes, globalSession.nodes)
	active := cloneNode(globalSession.active)
	defaultNodeName := globalSession.defaultNodeName
	defaultRegex := globalSession.defaultNodeRegex
	aiNodeName := globalSession.aiNodeName
	aiRegex := globalSession.aiNodeRegex
	globalSession.mu.RUnlock()

	defaultNode := findNodeByName(nodes, defaultNodeName)
	if defaultNode == nil {
		defaultNode = active
	}
	if defaultNode == nil && strings.TrimSpace(defaultRegex) != "" {
		defaultNode = selectBestMatchingNode(nodes, defaultRegex, "", nil)
	}

	aiNode := findNodeByName(nodes, aiNodeName)
	if aiNode == nil && defaultNode != nil && nodeMatchesRegex(*defaultNode, aiRegex) {
		aiNode = cloneNode(defaultNode)
	}
	if aiNode == nil {
		aiNode = selectBestMatchingNode(nodes, aiRegex, "chat.openai.com:443", nil)
	}

	return defaultNode, aiNode
}

// dialWithGFW 分流：默认保持当前线路，AI 域名可按正则命中专线，仅在不可用时切换
func dialWithGFW(node *ProxyNode, targetHost string) (net.Conn, error) {
	selectedNode, shouldProxy, fallbackNode := resolveRouteNode(node, targetHost)
	if !shouldProxy || selectedNode == nil {
		return net.DialTimeout("tcp", targetHost, 10*time.Second)
	}

	conn, err := dialUpstream(selectedNode, targetHost)
	if err == nil {
		return conn, nil
	}

	if fallbackNode != nil && proxyNodeKey(*fallbackNode) != proxyNodeKey(*selectedNode) {
		if fallbackConn, fallbackErr := dialUpstream(fallbackNode, targetHost); fallbackErr == nil {
			return fallbackConn, nil
		}
	}

	// 节点连接失败，异步触发切换（不阻塞当前请求）
	go triggerFailover()
	return nil, err
}

var failoverMu sync.Mutex
var lastFailover time.Time

// triggerFailover 请求失败时触发节点切换，60s 内只触发一次
func triggerFailover() {
	failoverMu.Lock()
	if time.Since(lastFailover) < 60*time.Second {
		failoverMu.Unlock()
		return
	}
	lastFailover = time.Now()
	failoverMu.Unlock()

	log.Printf("proxy: 请求异常，触发自动切换节点")
	globalProxyHandler.switchToBestNode()
}

// globalProxyHandler 供 triggerFailover 调用
var globalProxyHandler *ProxyHandler

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

// dialTrojan 通过 Trojan 节点建立到目标的连接
// Trojan 协议：TLS 连接 + SHA224(password) + CRLF + SOCKS5 地址格式 + CRLF
func dialTrojan(nodeAddr, targetHost string, node *ProxyNode) (net.Conn, error) {
	password, _ := node.Extra["password"].(string)
	if password == "" {
		password, _ = node.Extra["Password"].(string)
	}
	sni := node.Server
	if v, ok := node.Extra["sni"].(string); ok && v != "" {
		sni = v
	}
	skipVerify := false
	if v, ok := node.Extra["skip-cert-verify"].(bool); ok {
		skipVerify = v
	}

	tlsCfg := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: skipVerify,
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", nodeAddr, tlsCfg)
	if err != nil {
		return nil, err
	}

	// SHA224(password) hex
	h := sha256.New224()
	h.Write([]byte(password))
	hexPass := hex.EncodeToString(h.Sum(nil))

	// 解析目标地址
	host, portStr, err := net.SplitHostPort(targetHost)
	if err != nil {
		conn.Close()
		return nil, err
	}
	var portNum int
	fmt.Sscanf(portStr, "%d", &portNum)

	// Trojan 请求头：hex(sha224(pass)) CRLF 0x01(TCP) ATYP host port CRLF
	var req []byte
	req = append(req, []byte(hexPass)...)
	req = append(req, '\r', '\n')
	req = append(req, 0x01) // CMD: TCP connect
	// ATYP: 0x03 = domain
	req = append(req, 0x03, byte(len(host)))
	req = append(req, []byte(host)...)
	req = append(req, byte(portNum>>8), byte(portNum))
	req = append(req, '\r', '\n')

	if _, err := conn.Write(req); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// sendAlert 发送告警邮件（SMTP SSL，465 端口）
func (h *ProxyHandler) sendAlert(subject, body string) {
	if h.alertEmail == "" || h.smtpHost == "" || h.smtpUser == "" || h.smtpPass == "" {
		return
	}

	port := h.smtpPort
	if port == 0 {
		port = 465
	}
	addr := net.JoinHostPort(h.smtpHost, strconv.Itoa(port))
	recipients := splitAlertRecipients(h.alertEmail)
	if len(recipients) == 0 {
		return
	}

	messageIDHost := "devtools.local"
	if host, err := os.Hostname(); err == nil && strings.TrimSpace(host) != "" {
		messageIDHost = host
	}
	headers := []string{
		fmt.Sprintf("From: %s", h.smtpUser),
		fmt.Sprintf("To: %s", strings.Join(recipients, ", ")),
		fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("UTF-8", "[DevTools] "+subject)),
		fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)),
		fmt.Sprintf("Message-ID: <%d.%s@%s>", time.Now().UnixNano(), randomHex(8), messageIDHost),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	msg := strings.Join(headers, "\r\n") + "\r\n\r\n" + body

	var (
		client *smtp.Client
		conn   net.Conn
		err    error
	)

	dialer := &net.Dialer{Timeout: 15 * time.Second}
	tlsCfg := &tls.Config{ServerName: h.smtpHost}
	if port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
		if err != nil {
			log.Printf("proxy alert: SMTP SSL 连接失败: %v", err)
			return
		}
		client, err = smtp.NewClient(conn, h.smtpHost)
	} else {
		conn, err = dialer.Dial("tcp", addr)
		if err != nil {
			log.Printf("proxy alert: SMTP 连接失败: %v", err)
			return
		}
		client, err = smtp.NewClient(conn, h.smtpHost)
		if err == nil {
			if ok, _ := client.Extension("STARTTLS"); ok {
				if err = client.StartTLS(tlsCfg); err != nil {
					log.Printf("proxy alert: SMTP STARTTLS 失败: %v", err)
					client.Close()
					return
				}
			}
		}
	}
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		log.Printf("proxy alert: SMTP client 初始化失败: %v", err)
		return
	}
	defer client.Close()

	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", h.smtpUser, h.smtpPass, h.smtpHost)
		if err = client.Auth(auth); err != nil {
			log.Printf("proxy alert: SMTP 认证失败: %v", err)
			return
		}
	}

	if err = client.Mail(h.smtpUser); err != nil {
		log.Printf("proxy alert: SMTP MAIL FROM 失败: %v", err)
		return
	}
	for _, recipient := range recipients {
		if err = client.Rcpt(recipient); err != nil {
			log.Printf("proxy alert: SMTP RCPT TO 失败 (%s): %v", recipient, err)
			return
		}
	}

	wc, err := client.Data()
	if err != nil {
		log.Printf("proxy alert: SMTP DATA 失败: %v", err)
		return
	}
	if _, err = io.WriteString(wc, msg); err != nil {
		wc.Close()
		log.Printf("proxy alert: SMTP 写入邮件正文失败: %v", err)
		return
	}
	if err = wc.Close(); err != nil {
		log.Printf("proxy alert: SMTP 提交邮件失败: %v", err)
		return
	}
	if err = client.Quit(); err != nil {
		log.Printf("proxy alert: SMTP QUIT 失败: %v", err)
		return
	}

	log.Printf("proxy alert: 告警邮件已发送至 %s，主题=%s", strings.Join(recipients, ","), subject)
}

func splitAlertRecipients(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	recipients := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		recipients = append(recipients, item)
	}
	return recipients
}

func randomHex(length int) string {
	if length <= 0 {
		return ""
	}
	buf := make([]byte, (length+1)/2)
	if _, err := rand.Read(buf); err != nil {
		fallback := strconv.FormatInt(time.Now().UnixNano(), 16)
		if len(fallback) > length {
			return fallback[:length]
		}
		return fallback
	}
	value := hex.EncodeToString(buf)
	if len(value) > length {
		return value[:length]
	}
	return value
}

// Tunnel 处理 HTTP CONNECT 方法，让 DevTools 自身端口充当 HTTP 代理
// 浏览器/系统代理配置：http://yourserver:PORT
// 密码通过 Proxy-Authorization: Basic base64(user:password) 传递
func (h *ProxyHandler) Tunnel(w http.ResponseWriter, r *http.Request) {
	if !h.checkProxyAuth(r.Header.Get("Proxy-Authorization")) {
		if r.Method == http.MethodConnect {
			// CONNECT 请求必须返回 407，浏览器才会重试带认证头
			w.Header().Set("Proxy-Authenticate", `Basic realm="proxy"`)
			w.WriteHeader(407)
		} else {
			// 非 CONNECT（端口扫描/探测）返回 200 伪装
			w.Header().Set("Server", "nginx/1.24.0")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte("<html><body><h1>Welcome to nginx!</h1></body></html>"))
		}
		return
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	h.triggerAutoSelect()

	globalSession.mu.RLock()
	node := globalSession.active
	globalSession.mu.RUnlock()

	var upstream net.Conn
	var err error
	if node != nil {
		upstream, err = dialWithGFW(node, host)
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

// TunnelDirect 与 Tunnel 相同，但认证失败返回 407（供 NPS tunnel_port 使用）
// NPS 已有自身认证，不需要防探测伪装
func (h *ProxyHandler) TunnelDirect(w http.ResponseWriter, r *http.Request) {
	if !h.checkProxyAuth(r.Header.Get("Proxy-Authorization")) {
		w.Header().Set("Proxy-Authenticate", `Basic realm="proxy"`)
		w.WriteHeader(407)
		return
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	h.triggerAutoSelect()

	globalSession.mu.RLock()
	node := globalSession.active
	globalSession.mu.RUnlock()

	var upstream net.Conn
	var err error
	if node != nil {
		upstream, err = dialWithGFW(node, host)
	} else {
		upstream, err = net.DialTimeout("tcp", host, 10*time.Second)
	}
	if err != nil {
		http.Error(w, "Bad Gateway", 502)
		return
	}

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
	// 去掉 host 里可能带的端口，换成 tunnel_port（nginx 不支持 CONNECT，必须直连 tunnel_port）
	tunnelHost := proxyHost
	if h.npcTunnelPort != "" {
		hostname := proxyHost
		if hn, _, err := net.SplitHostPort(proxyHost); err == nil {
			hostname = hn
		}
		tunnelHost = hostname + ":" + h.npcTunnelPort
	}

	// MV3 manifest — 使用 PAC 脚本代理 + 主动注入认证头
	manifest := `{
  "manifest_version": 3,
  "name": "DevTools Proxy",
  "version": "3.1",
  "description": "通过服务器 HTTP 代理科学上网，支持自动认证",
  "permissions": [
    "storage",
    "proxy",
    "webRequest",
    "webRequestAuthProvider",
    "alarms"
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

	// background.js：PAC 脚本代理 + 主动注入认证头
	bgJS := fmt.Sprintf(`
const DEFAULT_SERVER = %q;
const DEFAULT_PASS   = %q;

function buildPac(server, mode) {
  // 被墙域名列表（命中则走代理）
  const blocked = [
    'google.com','googleapis.com','googleusercontent.com','gstatic.com','gmail.com',
    'youtube.com','youtu.be','ytimg.com','ggpht.com',
    'twitter.com','x.com','t.co','twimg.com',
    'facebook.com','fbcdn.net','instagram.com','whatsapp.com',
    'telegram.org','t.me',
    'github.com','githubusercontent.com','githubassets.com','ghcr.io',
    'openai.com','chatgpt.com','claude.ai','anthropic.com',
    'notion.so','notionusercontent.com',
    'medium.com','substack.com',
    'reddit.com','redd.it','redditmedia.com','redditstatic.com',
    'wikipedia.org','wikimedia.org',
    'dropbox.com','box.com','onedrive.live.com',
    'spotify.com','netflix.com','twitch.tv',
    'discord.com','discordapp.com','discordapp.net',
    'slack.com','zoom.us',
    'apple.com','icloud.com',
    'amazon.com','amazonaws.com',
    'microsoft.com','live.com','bing.com','msn.com',
    'pixiv.net','fanbox.cc',
    'dl.google.com','storage.googleapis.com',
    'cloudflare.com','cdn.cloudflare.net',
    'jsdelivr.net','unpkg.com','npmjs.com',
    'docker.com','hub.docker.com',
    'stackoverflow.com','stackexchange.com',
    'v2ex.com',
  ];

  // 全部走代理模式
  if (mode === 'global') {
    var blockedStr2 = JSON.stringify(blocked);
    return (
      'function FindProxyForURL(url,host){' +
        'if(isPlainHostName(host)||host==="127.0.0.1"||host==="localhost"||' +
          'isInNet(host,"10.0.0.0","255.0.0.0")||isInNet(host,"172.16.0.0","255.240.0.0")||' +
          'isInNet(host,"192.168.0.0","255.255.0.0"))return "DIRECT";' +
        'return "PROXY ' + server + '";' +
      '}'
    );
  }

  // 智能分流模式（默认）：
  // 被墙域名 → PROXY（代理不通再直连）
  // 其余 → DIRECT; PROXY（直连不通再走代理）
  var blockedStr = JSON.stringify(blocked);
  return (
    'var BLOCKED=' + blockedStr + ';' +
    'function FindProxyForURL(url,host){' +
      'if(isPlainHostName(host)||host==="127.0.0.1"||host==="localhost"||' +
        'isInNet(host,"10.0.0.0","255.0.0.0")||isInNet(host,"172.16.0.0","255.240.0.0")||' +
        'isInNet(host,"192.168.0.0","255.255.0.0"))return "DIRECT";' +
      'for(var i=0;i<BLOCKED.length;i++){var d=BLOCKED[i];if(host===d||host.slice(-(d.length+1))==="."+d)return "PROXY ' + server + '; DIRECT";}' +
      'return "DIRECT; PROXY ' + server + '";' +
    '}'
  );
}

function applyProxy(server, mode) {
  const pac = buildPac(server, mode || 'ai_priority');
  chrome.proxy.settings.set({
    value: { mode: 'pac_script', pacScript: { data: pac } },
    scope: 'regular'
  });
}

function disableProxy() {
  chrome.proxy.settings.clear({ scope: 'regular' });
}

function loadAndApply() {
  chrome.storage.local.get(['proxyEnabled', 'server', 'mode'], (s) => {
    if (s.proxyEnabled !== false) {
      applyProxy(s.server || DEFAULT_SERVER, s.mode || 'ai_priority');
    } else {
      disableProxy();
    }
  });
}

// 启动时立即用默认值同步设置代理，避免 storage 异步导致新标签页第一个请求走直连
applyProxy(DEFAULT_SERVER, 'ai_priority');

chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.local.set({ server: DEFAULT_SERVER, pass: DEFAULT_PASS, proxyEnabled: true, mode: 'ai_priority' });
  loadAndApply();
});
chrome.runtime.onStartup.addListener(loadAndApply);
loadAndApply();

// MV3 service worker 会被 Chrome 休眠，用 alarm 每25秒唤醒一次保持代理设置
chrome.alarms.create('keepAlive', { periodInMinutes: 0.4 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'keepAlive') loadAndApply();
});

// 自动填充代理认证
chrome.webRequest.onAuthRequired.addListener(
  (details) => {
    if (!details.isProxy) return {};
    return new Promise((resolve) => {
      chrome.storage.local.get(['pass'], (s) => {
        resolve({ authCredentials: { username: 'proxy', password: s.pass || DEFAULT_PASS } });
      });
    });
  },
  { urls: ['<all_urls>'] }
);

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.action === 'update') {
    chrome.storage.local.set({ server: msg.server, pass: msg.pass, mode: msg.mode, proxyEnabled: true }, () => {
      applyProxy(msg.server, msg.mode);
    });
  } else if (msg.action === 'disable') {
    chrome.storage.local.set({ proxyEnabled: false });
    disableProxy();
  } else if (msg.action === 'enable') {
    chrome.storage.local.get(['server', 'mode'], (s) => {
      chrome.storage.local.set({ proxyEnabled: true });
      applyProxy(s.server || DEFAULT_SERVER, s.mode || 'ai_priority');
    });
  }
  sendResponse({});
  return true;
});
`, tunnelHost, password)

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
<label>模式</label>
<select id="mode" style="width:100%;padding:5px 8px;border:1px solid #ddd;border-radius:4px;font-size:13px;">
  <option value="ai_priority">AI 优先分流（默认，GPT/Claude/Gemini 走专线）</option>
  <option value="smart">智能分流（国内直连，被墙走代理，不通自动切换）</option>
  <option value="global">全部走代理</option>
</select>
<div class="hint">HTTP 代理，流量经 NPS 隧道转发。安装后可在弹窗中开启/关闭代理。</div>
<div class="hint" style="margin-top:8px;padding:6px 8px;background:#fff7e6;border-radius:4px;color:#b45309;">⚠️ Chrome 内置翻译、拼写检查等功能不走扩展代理。如需翻译，请在系统设置中配置 HTTP 代理：<span id="sysProxy" style="font-family:monospace;user-select:all;"></span></div>
<div class="row">
  <span class="status" id="status">检测中...</span>
  <button class="btn-save" id="saveBtn">保存并启用</button>
  <button id="toggleBtn">-</button>
</div>
<script src="popup.js"></script>
</body></html>`

	popupJS := `
var enabled = true;

chrome.storage.local.get(['server','pass','proxyEnabled','mode'], function(s) {
  document.getElementById('server').value = s.server || '';
  document.getElementById('pass').value = s.pass || '';
  document.getElementById('mode').value = s.mode || 'ai_priority';
  enabled = s.proxyEnabled !== false;
  var srv = s.server || '';
  if (srv) { document.getElementById('sysProxy').textContent = srv; }
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
  var mode = document.getElementById('mode').value;
  if (!server || !pass) { alert('请填写服务器地址和密码'); return; }
  chrome.runtime.sendMessage({ action: 'update', server: server, pass: pass, mode: mode });
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

	h.triggerAutoSelect()

	globalSession.mu.RLock()
	node := globalSession.active
	globalSession.mu.RUnlock()

	var upstream net.Conn
	if node != nil {
		upstream, err = dialWithGFW(node, host)
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
