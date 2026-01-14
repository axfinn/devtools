package handlers

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

type DNSHandler struct{}

func NewDNSHandler() *DNSHandler {
	return &DNSHandler{}
}

type MXRecord struct {
	Host     string `json:"host"`
	Priority uint16 `json:"priority"`
}

type DNSResult struct {
	A     []string    `json:"a,omitempty"`
	AAAA  []string    `json:"aaaa,omitempty"`
	CNAME []string    `json:"cname,omitempty"`
	MX    []MXRecord  `json:"mx,omitempty"`
	NS    []string    `json:"ns,omitempty"`
	TXT   []string    `json:"txt,omitempty"`
}

// GetIP 返回客户端 IP 地址
func (h *DNSHandler) GetIP(c *gin.Context) {
	ip := c.ClientIP()

	// 尝试从常见的代理头获取真实 IP
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		ip = realIP
	} else if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For 可能包含多个 IP，取第一个
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			ip = strings.TrimSpace(ips[0])
		}
	}

	c.JSON(200, gin.H{"ip": ip})
}

// Lookup 执行 DNS 查询
func (h *DNSHandler) Lookup(c *gin.Context) {
	domain := strings.TrimSpace(c.Query("domain"))
	recordType := strings.ToUpper(c.Query("type"))

	if domain == "" {
		c.JSON(400, gin.H{"error": "域名不能为空"})
		return
	}

	// 移除可能的协议前缀
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, "/")

	// 提取主机名（去掉路径）
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	result := &DNSResult{}

	// 根据请求的记录类型查询
	switch recordType {
	case "A":
		result.A = lookupA(domain)
	case "AAAA":
		result.AAAA = lookupAAAA(domain)
	case "CNAME":
		result.CNAME = lookupCNAME(domain)
	case "MX":
		result.MX = lookupMX(domain)
	case "NS":
		result.NS = lookupNS(domain)
	case "TXT":
		result.TXT = lookupTXT(domain)
	default:
		// 查询所有类型
		result.A = lookupA(domain)
		result.AAAA = lookupAAAA(domain)
		result.CNAME = lookupCNAME(domain)
		result.MX = lookupMX(domain)
		result.NS = lookupNS(domain)
		result.TXT = lookupTXT(domain)
	}

	c.JSON(200, result)
}

func lookupA(domain string) []string {
	var result []string
	ips, err := net.LookupIP(domain)
	if err != nil {
		return result
	}
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			result = append(result, ipv4.String())
		}
	}
	return result
}

func lookupAAAA(domain string) []string {
	var result []string
	ips, err := net.LookupIP(domain)
	if err != nil {
		return result
	}
	for _, ip := range ips {
		if ip.To4() == nil {
			result = append(result, ip.String())
		}
	}
	return result
}

func lookupCNAME(domain string) []string {
	var result []string
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		return result
	}
	// 去掉末尾的点
	cname = strings.TrimSuffix(cname, ".")
	if cname != "" && cname != domain {
		result = append(result, cname)
	}
	return result
}

func lookupMX(domain string) []MXRecord {
	var result []MXRecord
	mxs, err := net.LookupMX(domain)
	if err != nil {
		return result
	}
	for _, mx := range mxs {
		result = append(result, MXRecord{
			Host:     strings.TrimSuffix(mx.Host, "."),
			Priority: mx.Pref,
		})
	}
	return result
}

func lookupNS(domain string) []string {
	var result []string
	nss, err := net.LookupNS(domain)
	if err != nil {
		return result
	}
	for _, ns := range nss {
		result = append(result, strings.TrimSuffix(ns.Host, "."))
	}
	return result
}

func lookupTXT(domain string) []string {
	var result []string
	txts, err := net.LookupTXT(domain)
	if err != nil {
		return result
	}
	result = append(result, txts...)
	return result
}
