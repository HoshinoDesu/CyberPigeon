package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/HoshinoDesu/CyberPigeon/internal/config"
)

// TelegramChannel Telegram Bot 通道
type TelegramChannel struct {
	cfg    config.ChannelConfig
	client *http.Client
}

// NewTelegramChannel 创建 Telegram 通道
func NewTelegramChannel(cfg config.ChannelConfig) (*TelegramChannel, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("Telegram bot_token 未配置")
	}
	if cfg.ChatID == "" {
		return nil, fmt.Errorf("Telegram chat_id 未配置")
	}
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.telegram.org"
	}
	// 确保 API URL 没有以斜杠结尾
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")

	return &TelegramChannel{cfg: cfg, client: newHTTPClient(cfg)}, nil
}

// Type 返回通道类型
func (t *TelegramChannel) Type() string {
	return "telegram"
}

// Send 发送 Telegram 通知
func (t *TelegramChannel) Send(msg Message) error {
	api := fmt.Sprintf("%s/bot%s/sendMessage", t.cfg.APIURL, t.cfg.BotToken)

	payload := map[string]interface{}{
		"chat_id": t.cfg.ChatID,
		"text":    msg.String(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, api, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送 Telegram 消息失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Ok          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if !result.Ok {
		return fmt.Errorf("Telegram 返回错误: %d %s", result.ErrorCode, result.Description)
	}

	return nil
}
