package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	server := flag.String("server", "", "服务器地址，如 yourserver.com 或 wss://yourserver.com")
	password := flag.String("password", "", "管理员密码")
	listen := flag.String("listen", "127.0.0.1:1080", "本地 SOCKS5 监听地址")
	flag.Parse()

	if *server == "" || *password == "" {
		fmt.Println("用法: proxy-client -server yourserver.com -password xxx [-listen 127.0.0.1:1080]")
		fmt.Println("")
		fmt.Println("参数:")
		fmt.Println("  -server    服务器地址（如 yourserver.com 或 wss://yourserver.com）")
		fmt.Println("  -password  管理员密码")
		fmt.Println("  -listen    本地 SOCKS5 监听地址（默认 127.0.0.1:1080）")
		fmt.Println("")
		fmt.Println("启动后浏览器配置 SOCKS5 代理: 127.0.0.1:1080")
		flag.Usage()
		return
	}

	// 规范化服务器地址
	wsURL := normalizeServerURL(*server)
	log.Printf("服务器: %s", wsURL)
	log.Printf("本地监听: %s", *listen)

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("监听失败: %v", err)
	}
	log.Printf("SOCKS5 代理已启动，浏览器配置: SOCKS5 %s", *listen)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleSocks5(conn, wsURL, *password)
	}
}

// normalizeServerURL 把各种格式的服务器地址统一成 ws(s)://host/api/proxy/ws-tunnel
func normalizeServerURL(server string) string {
	// 已经是完整 URL
	if strings.HasPrefix(server, "ws://") || strings.HasPrefix(server, "wss://") {
		if !strings.Contains(server, "/api/proxy/ws-tunnel") {
			return strings.TrimRight(server, "/") + "/api/proxy/ws-tunnel"
		}
		return server
	}
	// http/https 协议
	if strings.HasPrefix(server, "https://") {
		host := strings.TrimPrefix(server, "https://")
		return "wss://" + strings.TrimRight(host, "/") + "/api/proxy/ws-tunnel"
	}
	if strings.HasPrefix(server, "http://") {
		host := strings.TrimPrefix(server, "http://")
		return "ws://" + strings.TrimRight(host, "/") + "/api/proxy/ws-tunnel"
	}
	// 裸域名或 IP，根据是否有端口判断协议
	// 有非 443/80 端口或 localhost/IP 用 ws，否则用 wss
	host := strings.TrimRight(server, "/")
	if strings.Contains(host, ":") {
		_, port, _ := net.SplitHostPort(host)
		if port == "443" {
			return "wss://" + host + "/api/proxy/ws-tunnel"
		}
		return "ws://" + host + "/api/proxy/ws-tunnel"
	}
	return "wss://" + host + "/api/proxy/ws-tunnel"
}
