package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/CyberPigeon/internal/config"
)

// WebhookChannel Webhook 通道
type WebhookChannel struct {
	cfg    config.ChannelConfig
	client *http.Client
}

// NewWebhookChannel 创建 Webhook 通道
func NewWebhookChannel(cfg config.ChannelConfig) (*WebhookChannel, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("Webhook URL 未配置")
	}
	u, err := url.Parse(cfg.URL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("Webhook URL 无效")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("Webhook URL 仅支持 http/https")
	}
	if !cfg.AllowPrivateNetwork && isPrivateOrLocalHost(u.Hostname()) {
		return nil, fmt.Errorf("Webhook URL 不允许私网或本地地址")
	}
	if cfg.Method == "" {
		cfg.Method = "POST"
	}
	return &WebhookChannel{cfg: cfg, client: newHTTPClient(cfg)}, nil
}

func isPrivateOrLocalHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return true
		}
		for _, v := range ips {
			if isPrivateIP(v) {
				return true
			}
		}
		return false
	}

	return isPrivateIP(ip)
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	return false
}

// Type 返回通道类型
func (w *WebhookChannel) Type() string {
	return "webhook"
}

// Send 发送 Webhook 通知
func (w *WebhookChannel) Send(msg Message) error {
	formattedTime := "未知"
	if !msg.Timestamp.IsZero() {
		formattedTime = msg.Timestamp.Format("2006-01-02 15:04:05")
	}

	payload := map[string]interface{}{
		"time": formattedTime,
		"from": msg.From,
		"text": msg.Text,
	}

	var req *http.Request

	if strings.ToUpper(w.cfg.Method) == "GET" {
		// GET 请求
		params := url.Values{}
		params.Set("time", formattedTime)
		params.Set("from", msg.From)
		params.Set("text", msg.Text)

		fullURL := w.cfg.URL
		if strings.Contains(fullURL, "?") {
			fullURL += "&" + params.Encode()
		} else {
			fullURL += "?" + params.Encode()
		}

		var err error
		req, err = http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return err
		}
	} else {
		// POST 请求
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		req, err = http.NewRequest("POST", w.cfg.URL, bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
	}

	// 添加自定义头
	for k, v := range w.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Webhook 返回错误状态: %d", resp.StatusCode)
	}

	return nil
}
