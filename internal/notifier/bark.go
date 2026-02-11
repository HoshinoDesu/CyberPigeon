package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sms-forwarder/internal/config"
)

// BarkChannel Bark 通道
type BarkChannel struct {
	cfg config.ChannelConfig
}

// NewBarkChannel 创建 Bark 通道
func NewBarkChannel(cfg config.ChannelConfig) (*BarkChannel, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("Bark endpoint 未配置")
	}
	return &BarkChannel{cfg: cfg}, nil
}

// Type 返回通道类型
func (b *BarkChannel) Type() string {
	return "bark"
}

// Send 发送 Bark 通知
func (b *BarkChannel) Send(msg Message) error {
	title := msg.From
	if title == "" {
		title = "未知号码"
	}

	payload := map[string]interface{}{
		"title": title,
		"body":  msg.String(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(b.cfg.Endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bark 返回错误状态: %d", resp.StatusCode)
	}

	return nil
}
