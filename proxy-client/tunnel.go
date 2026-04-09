package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// dialTunnel 建立 WebSocket 隧道，成功后回复 SOCKS5 成功并双向转发
func dialTunnel(socks net.Conn, wsBaseURL, password, target string) error {
	wsURL := fmt.Sprintf("%s?p=%s&host=%s",
		wsBaseURL,
		url.QueryEscape(password),
		url.QueryEscape(target),
	)

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	}

	wsConn, resp, err := dialer.Dial(wsURL, http.Header{})
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return fmt.Errorf("WebSocket 连接失败: %v", err)
	}
	defer wsConn.Close()

	// 回复 SOCKS5 成功
	socks.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// 双向转发
	done := make(chan struct{}, 2)

	// socks → ws
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			n, err := socks.Read(buf)
			if n > 0 {
				if werr := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// ws → socks
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			_, msg, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
			if _, werr := socks.Write(msg); werr != nil {
				return
			}
		}
	}()

	<-done
	return nil
}
