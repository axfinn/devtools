package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	server, password, listen := loadConfig()

	if server == "" || password == "" {
		fmt.Println("=== Proxy Client ===")
		fmt.Println("未找到配置，请输入：")
		if server == "" {
			fmt.Print("服务器地址（如 yourserver.com）: ")
			fmt.Scanln(&server)
		}
		if password == "" {
			fmt.Print("密码: ")
			fmt.Scanln(&password)
		}
		// 保存到 config.txt 方便下次直接双击
		saveConfig(server, password, listen)
		fmt.Println("已保存到 config.txt，下次双击直接启动")
	}

	wsURL := normalizeServerURL(server)
	fmt.Printf("\n服务器: %s\n", wsURL)
	fmt.Printf("本地 SOCKS5: %s\n", listen)
	fmt.Println("\n浏览器/系统代理配置: SOCKS5 127.0.0.1:1080")
	fmt.Println("按 Ctrl+C 停止\n")

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("监听失败: %v\n请检查端口 1080 是否被占用", err)
	}
	log.Printf("SOCKS5 代理已启动")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleSocks5(conn, wsURL, password)
	}
}

// loadConfig 从同目录 config.txt 读取配置
func loadConfig() (server, password, listen string) {
	listen = "127.0.0.1:1080"

	exe, err := os.Executable()
	if err != nil {
		return
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "config.txt")
	f, err := os.Open(cfgPath)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "server":
			server = strings.TrimSpace(v)
		case "password":
			password = strings.TrimSpace(v)
		case "listen":
			listen = strings.TrimSpace(v)
		}
	}
	return
}

func saveConfig(server, password, listen string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "config.txt")
	content := fmt.Sprintf("server=%s\npassword=%s\nlisten=%s\n", server, password, listen)
	os.WriteFile(cfgPath, []byte(content), 0600)
}

// normalizeServerURL 把各种格式的服务器地址统一成 ws(s)://host/api/proxy/ws-tunnel
func normalizeServerURL(server string) string {
	if strings.HasPrefix(server, "ws://") || strings.HasPrefix(server, "wss://") {
		if !strings.Contains(server, "/api/proxy/ws-tunnel") {
			return strings.TrimRight(server, "/") + "/api/proxy/ws-tunnel"
		}
		return server
	}
	if strings.HasPrefix(server, "https://") {
		host := strings.TrimPrefix(server, "https://")
		return "wss://" + strings.TrimRight(host, "/") + "/api/proxy/ws-tunnel"
	}
	if strings.HasPrefix(server, "http://") {
		host := strings.TrimPrefix(server, "http://")
		return "ws://" + strings.TrimRight(host, "/") + "/api/proxy/ws-tunnel"
	}
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
