package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sms-forwarder/internal/config"
)

// WebhookChannel Webhook 通道
type WebhookChannel struct {
	cfg config.ChannelConfig
}

// NewWebhookChannel 创建 Webhook 通道
func NewWebhookChannel(cfg config.ChannelConfig) (*WebhookChannel, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("Webhook URL 未配置")
	}
	if cfg.Method == "" {
		cfg.Method = "POST"
	}
	return &WebhookChannel{cfg: cfg}, nil
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
	var err error

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

		req, err = http.NewRequest("GET", fullURL, nil)
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

	if err != nil {
		return err
	}

	// 添加自定义头
	for k, v := range w.cfg.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Webhook 返回错误状态: %d", resp.StatusCode)
	}

	return nil
}
