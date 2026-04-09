package handlers

import (
	"bufio"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	gfwlistURL       = "https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt"
	gfwlistCacheFile = "/tmp/gfwlist_cache.txt"
	gfwlistTTL       = 24 * time.Hour
)

type gfwlistMatcher struct {
	mu        sync.RWMutex
	domains   map[string]bool // 精确域名及其父域名
	lastFetch time.Time
}

var globalGFW = &gfwlistMatcher{domains: map[string]bool{}}

// InitGFWList 启动时加载，后台定期刷新
func InitGFWList() {
	if err := globalGFW.load(); err != nil {
		log.Printf("gfwlist 初始加载失败: %v，使用空列表", err)
	}
	go func() {
		for range time.Tick(gfwlistTTL) {
			if err := globalGFW.load(); err != nil {
				log.Printf("gfwlist 刷新失败: %v", err)
			}
		}
	}()
}

// ShouldProxy 判断 host（可含端口）是否需要走代理
func ShouldProxy(host string) bool {
	domain := host
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		// 排除 IPv6 的情况
		if !strings.Contains(host[:idx], ":") {
			domain = host[:idx]
		}
	}
	return globalGFW.match(domain)
}

func (g *gfwlistMatcher) match(domain string) bool {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	g.mu.RLock()
	defer g.mu.RUnlock()
	// 逐级匹配：sub.example.com → example.com → com
	for {
		if g.domains[domain] {
			return true
		}
		idx := strings.Index(domain, ".")
		if idx < 0 {
			return false
		}
		domain = domain[idx+1:]
	}
}

func (g *gfwlistMatcher) load() error {
	raw, err := fetchGFWList()
	if err != nil {
		// 尝试读本地缓存
		raw, err = os.ReadFile(gfwlistCacheFile)
		if err != nil {
			return err
		}
		log.Printf("gfwlist 使用本地缓存")
	}

	domains := parseGFWList(raw)
	g.mu.Lock()
	g.domains = domains
	g.lastFetch = time.Now()
	g.mu.Unlock()
	log.Printf("gfwlist 加载完成，共 %d 条规则", len(domains))
	return nil
}

func fetchGFWList() ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(gfwlistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// gfwlist 是 base64 编码的
	dec := base64.NewDecoder(base64.StdEncoding, resp.Body)
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := dec.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}

	// 写缓存
	os.WriteFile(gfwlistCacheFile, buf, 0644)
	return buf, nil
}

// parseGFWList 解析 ABP 格式规则，提取域名
func parseGFWList(data []byte) map[string]bool {
	domains := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue
		}
		// 白名单规则跳过
		if strings.HasPrefix(line, "@@") {
			continue
		}

		domain := extractDomain(line)
		if domain != "" {
			domains[strings.ToLower(domain)] = true
		}
	}
	return domains
}

func extractDomain(rule string) string {
	// ||example.com^  →  example.com
	if strings.HasPrefix(rule, "||") {
		rule = strings.TrimPrefix(rule, "||")
		rule = strings.TrimSuffix(rule, "^")
		rule = strings.TrimSuffix(rule, "/")
		if isValidDomain(rule) {
			return rule
		}
		return ""
	}
	// |https://example.com  →  example.com
	if strings.HasPrefix(rule, "|") {
		rule = strings.TrimPrefix(rule, "|")
		rule = strings.TrimPrefix(rule, "https://")
		rule = strings.TrimPrefix(rule, "http://")
		rule = strings.SplitN(rule, "/", 2)[0]
		if isValidDomain(rule) {
			return rule
		}
		return ""
	}
	// .example.com 或 example.com（无通配符）
	rule = strings.TrimPrefix(rule, ".")
	if !strings.Contains(rule, "*") && !strings.Contains(rule, "/") && isValidDomain(rule) {
		return rule
	}
	return ""
}

func isValidDomain(s string) bool {
	if s == "" || strings.Contains(s, " ") || strings.Contains(s, "*") {
		return false
	}
	return strings.Contains(s, ".")
}
