package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/CyberPigeon/internal/config"
)

// GotifyChannel Gotify 通道
type GotifyChannel struct {
	cfg    config.ChannelConfig
	client *http.Client
}

// NewGotifyChannel 创建 Gotify 通道
func NewGotifyChannel(cfg config.ChannelConfig) (*GotifyChannel, error) {
	if cfg.Endpoint == "" || cfg.Token == "" {
		return nil, fmt.Errorf("Gotify 配置不完整")
	}
	if cfg.Priority == 0 {
		cfg.Priority = 5
	}
	return &GotifyChannel{cfg: cfg, client: newHTTPClient(cfg)}, nil
}

// Type 返回通道类型
func (g *GotifyChannel) Type() string {
	return "gotify"
}

// Send 发送 Gotify 通知
func (g *GotifyChannel) Send(msg Message) error {
	url := fmt.Sprintf("%s/message", g.cfg.Endpoint)
	title := msg.From
	if title == "" {
		title = "未知号码"
	}

	payload := map[string]interface{}{
		"title":    title,
		"message":  msg.String(),
		"priority": g.cfg.Priority,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", g.cfg.Token)

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Gotify 返回错误状态: %d", resp.StatusCode)
	}

	return nil
}
