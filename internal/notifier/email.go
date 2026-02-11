package notifier

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/sms-forwarder/internal/config"
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
	if cfg.UseTLS {
		cfg.UseTLS = true // 默认使用 TLS
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
	body := msg.String()

	// 构建邮件内容
	headers := make(map[string]string)
	headers["From"] = e.cfg.From
	headers["To"] = strings.Join(e.cfg.To, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", e.cfg.Host, e.cfg.Port)

	var auth smtp.Auth
	if e.cfg.Username != "" && e.cfg.Password != "" {
		auth = smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.Host)
	}

	if e.cfg.UseTLS {
		// 使用 TLS
		tlsConfig := &tls.Config{
			ServerName: e.cfg.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS 连接失败: %w", err)
		}
		defer conn.Close()

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
		_, err = w.Write([]byte(message.String()))
		if err != nil {
			return err
		}
		return w.Close()
	}

	// 不使用 TLS
	return smtp.SendMail(addr, auth, e.cfg.From, e.cfg.To, []byte(message.String()))
}
