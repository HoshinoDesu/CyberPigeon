package notifier

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"

	"github.com/CyberPigeon/internal/config"
)

// EmailChannel Email 通道
type EmailChannel struct {
	cfg config.ChannelConfig
}

// NewEmailChannel 创建 Email 通道
func NewEmailChannel(cfg config.ChannelConfig) (*EmailChannel, error) {
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" || len(cfg.To) == 0 {
		return nil, fmt.Errorf("Email 配置不完整")
	}
	// 默认启用 TLS（用户未显式配置时 bool 零值为 false，这里默认设为 true）
	// 当 Port 为 587 或 465 时，自动视为加密连接
	if !cfg.UseTLS && (cfg.Port == 587 || cfg.Port == 465) {
		cfg.UseTLS = true
	}
	return &EmailChannel{cfg: cfg}, nil
}

// Type 返回通道类型
func (e *EmailChannel) Type() string {
	return "email"
}

// Send 发送邮件
func (e *EmailChannel) Send(msg Message) error {
	subject := msg.From
	if subject == "" {
		subject = "未知号码"
	}
	// RFC 2047 编码 Subject，防止非 ASCII 字符乱码
	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
	body := msg.String()

	// 构建邮件内容（使用有序写入，避免 map 遍历顺序不确定）
	var message strings.Builder
	message.WriteString(fmt.Sprintf("From: %s\r\n", e.cfg.From))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.cfg.To, ",")))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("\r\n")
	message.WriteString(body)

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", e.cfg.Host, e.cfg.Port)

	var auth smtp.Auth
	if e.cfg.Username != "" && e.cfg.Password != "" {
		auth = smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.Host)
	}

	if !e.cfg.UseTLS {
		// 不使用 TLS
		return smtp.SendMail(addr, auth, e.cfg.From, e.cfg.To, []byte(message.String()))
	}

	// 端口 465：隐式 TLS（直接 TLS 连接）
	if e.cfg.Port == 465 {
		return e.sendWithImplicitTLS(addr, auth, message.String())
	}

	// 其他端口（如 587）：STARTTLS（先明文连接，再升级为 TLS）
	return e.sendWithSTARTTLS(addr, auth, message.String())
}

// sendWithImplicitTLS 使用隐式 TLS（端口 465）
func (e *EmailChannel) sendWithImplicitTLS(addr string, auth smtp.Auth, message string) error {
	tlsConfig := &tls.Config{
		ServerName: e.cfg.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS 连接失败: %w", err)
	}
	// 不 defer conn.Close()，由 sendViaClient 内 client.Quit() 负责关闭

	return e.sendViaClient(conn, auth, message)
}

// sendWithSTARTTLS 使用 STARTTLS（端口 587 等）
func (e *EmailChannel) sendWithSTARTTLS(addr string, auth smtp.Auth, message string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	// conn 的生命周期由 smtp.Client 接管，Quit() 会关闭底层连接

	client, err := smtp.NewClient(conn, e.cfg.Host)
	if err != nil {
		conn.Close() // NewClient 失败时需手动关闭
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Quit()

	// 发送 STARTTLS
	tlsConfig := &tls.Config{
		ServerName: e.cfg.Host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS 失败: %w", err)
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("认证失败: %w", err)
		}
	}

	if err := client.Mail(e.cfg.From); err != nil {
		return err
	}
	for _, to := range e.cfg.To {
		if err := client.Rcpt(to); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write([]byte(message)); err != nil {
		return err
	}
	return w.Close()
}

// sendViaClient 通过已建立的连接发送邮件
func (e *EmailChannel) sendViaClient(conn net.Conn, auth smtp.Auth, message string) error {
	client, err := smtp.NewClient(conn, e.cfg.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Quit()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("认证失败: %w", err)
		}
	}

	if err := client.Mail(e.cfg.From); err != nil {
		return err
	}
	for _, to := range e.cfg.To {
		if err := client.Rcpt(to); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write([]byte(message)); err != nil {
		return err
	}
	return w.Close()
}
