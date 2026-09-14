// Package notif 提供一个共享的 SMTP 邮件发送通道。
//
// 复用现有的 SMTP 客户端流程（与 handlers/proxy.go、handlers/planner_profile.go
// 同源:465 隐式 TLS,其他端口走 STARTTLS,PlainAuth,multipart attachment）。
// 新模块（如 trip）只需配置 SMTP 并调用 Send,不再各自维护一份连接逻辑。
//
// 故意不替换 proxy / planner 已有的发送路径,避免改动范围扩散;此处只暴露
// 给新模块使用,保持原有调用方的稳定性。
package notif

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// Attachment 邮件附件。Content 为空时跳过 multipart。
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Config 邮件发送所需的 SMTP 凭据。
type Config struct {
	Host     string // SMTP 服务器,留空视为未配置,Send 直接返回 nil 静默跳过
	Port     int    // SMTP 端口;0 = 465
	User     string // 发件人邮箱(同时作为 SMTP 登录用户名)
	Password string // SMTP 授权码 / 密码
	From     string // 邮件 From 头;为空则用 User
}

// Configured 报告 SMTP 是否配齐四项(Host/User/Password/From)。
// 任一缺失就视为未启用,Send 静默跳过。
func (c Config) Configured() bool {
	return strings.TrimSpace(c.Host) != "" &&
		strings.TrimSpace(c.User) != "" &&
		strings.TrimSpace(c.Password) != "" &&
		strings.TrimSpace(c.FromOrDefault()) != ""
}

// FromOrDefault 返回 From 头,缺省用 User。
func (c Config) FromOrDefault() string {
	if v := strings.TrimSpace(c.From); v != "" {
		return v
	}
	return strings.TrimSpace(c.User)
}

// Send 发一封纯文本邮件,可选附件。
//
// 未配置 / 收件人为空 / 任何网络错误都返回 error,调用方决定是否
// 落到日志还是升级到用户可见提示。本函数不 panic。
func Send(cfg Config, recipients []string, subject, body string, att *Attachment) error {
	if !cfg.Configured() {
		return fmt.Errorf("notif: smtp not configured")
	}
	cleaned := SplitRecipients(strings.Join(recipients, ","))
	if len(cleaned) == 0 {
		return fmt.Errorf("notif: no recipients")
	}
	port := cfg.Port
	if port == 0 {
		port = 465
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	msg := buildMessage(cfg.FromOrDefault(), cleaned, subject, body, att)

	client, conn, err := dialSMTP(cfg.Host, addr, port)
	if err != nil {
		return err
	}
	defer func() {
		if client != nil {
			_ = client.Close()
		}
		if conn != nil {
			_ = conn.Close()
		}
	}()

	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("notif: smtp auth: %w", err)
		}
	}
	if err = client.Mail(cfg.User); err != nil {
		return fmt.Errorf("notif: smtp MAIL FROM: %w", err)
	}
	for _, r := range cleaned {
		if err = client.Rcpt(r); err != nil {
			return fmt.Errorf("notif: smtp RCPT TO %s: %w", r, err)
		}
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("notif: smtp DATA: %w", err)
	}
	if _, err = io.WriteString(wc, msg); err != nil {
		_ = wc.Close()
		return fmt.Errorf("notif: smtp write: %w", err)
	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("notif: smtp commit: %w", err)
	}
	if err = client.Quit(); err != nil {
		// Quit 失败常见于服务端先 close,内容已发出,只记日志不计失败
		log.Printf("notif: smtp QUIT 提示失败(邮件已发送): %v", err)
	}
	return nil
}

// SendAndLog 是 Send 的便利包装:失败时记一行日志,返回是否成功。
// 适用于 fire-and-forget 的提醒场景(如行程前 N 天邮件)。
func SendAndLog(cfg Config, recipients []string, subject, body string, att *Attachment) bool {
	if err := Send(cfg, recipients, subject, body, att); err != nil {
		log.Printf("notif: send failed subject=%q err=%v", subject, err)
		return false
	}
	return true
}

// SplitRecipients 把 "a@x; b@x, c@x" 之类的字符串拆成去重去空白的列表。
// 同时支持传入已切好的 slice(只做去重去空白)。
func SplitRecipients(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		item := strings.TrimSpace(p)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// dialSMTP 处理 465 隐式 TLS 与其他端口 STARTTLS 的分支,
// 复用 proxy.go / planner_profile.go 的连接策略。
func dialSMTP(host, addr string, port int) (*smtp.Client, net.Conn, error) {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	tlsCfg := &tls.Config{ServerName: host}
	var (
		conn   net.Conn
		client *smtp.Client
		err    error
	)
	if port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
		if err != nil {
			return nil, nil, fmt.Errorf("notif: smtp ssl dial: %w", err)
		}
		client, err = smtp.NewClient(conn, host)
	} else {
		conn, err = dialer.Dial("tcp", addr)
		if err != nil {
			return nil, nil, fmt.Errorf("notif: smtp dial: %w", err)
		}
		client, err = smtp.NewClient(conn, host)
		if err == nil {
			if ok, _ := client.Extension("STARTTLS"); ok {
				if err = client.StartTLS(tlsCfg); err != nil {
					_ = client.Close()
					_ = conn.Close()
					return nil, nil, fmt.Errorf("notif: smtp starttls: %w", err)
				}
			}
		}
	}
	if err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, nil, fmt.Errorf("notif: smtp client: %w", err)
	}
	return client, conn, nil
}

// buildMessage 拼一封带 From/To/Subject/Date/Message-ID 的 RFC822 邮件,
// 有附件时切 multipart/mixed。逻辑与 proxy.go:buildSMTPMessage 一致,
// 移到这里是为了让新模块不依赖 handlers 包。
func buildMessage(sender string, recipients []string, subject, body string, att *Attachment) string {
	messageIDHost := "devtools.local"
	if h, err := os.Hostname(); err == nil && strings.TrimSpace(h) != "" {
		messageIDHost = h
	}
	headers := []string{
		fmt.Sprintf("From: %s", sender),
		fmt.Sprintf("To: %s", strings.Join(recipients, ", ")),
		fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("UTF-8", "[DevTools] "+subject)),
		fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)),
		fmt.Sprintf("Message-ID: <%d.%s@%s>", time.Now().UnixNano(), randomHex(8), messageIDHost),
		"MIME-Version: 1.0",
	}
	if att == nil || len(att.Content) == 0 {
		headers = append(headers,
			"Content-Type: text/plain; charset=UTF-8",
			"Content-Transfer-Encoding: 8bit",
		)
		return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
	}

	filename := strings.TrimSpace(att.Filename)
	if filename == "" {
		filename = "devtools-attachment.txt"
	}
	contentType := strings.TrimSpace(att.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	boundary := "devtools-boundary-" + randomHex(16)
	headers = append(headers, fmt.Sprintf("Content-Type: multipart/mixed; boundary=%q", boundary))

	var msg strings.Builder
	msg.WriteString(strings.Join(headers, "\r\n"))
	msg.WriteString("\r\n\r\n")
	msg.WriteString("--" + boundary + "\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	msg.WriteString(body)
	msg.WriteString("\r\n")
	msg.WriteString("--" + boundary + "\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: %s; name=%q\r\n", contentType, filename))
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%q\r\n\r\n", filename))
	writeBase64Lines(&msg, att.Content)
	msg.WriteString("--" + boundary + "--\r\n")
	return msg.String()
}

func writeBase64Lines(dst *strings.Builder, data []byte) {
	if len(data) == 0 {
		dst.WriteString("\r\n")
		return
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		dst.WriteString(encoded[:76])
		dst.WriteString("\r\n")
		encoded = encoded[76:]
	}
	dst.WriteString(encoded)
	dst.WriteString("\r\n")
}

func randomHex(length int) string {
	if length <= 0 {
		length = 16
	}
	buf := make([]byte, (length+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)[:length]
}