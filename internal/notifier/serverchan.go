package notifier

import (
	"fmt"

	serverchan_sdk "github.com/easychen/serverchan-sdk-golang"
	"github.com/sms-forwarder/internal/config"
)

// ServerChanChannel ServerChan 通道
type ServerChanChannel struct {
	cfg config.ChannelConfig
}

// NewServerChanChannel 创建 ServerChan 通道
func NewServerChanChannel(cfg config.ChannelConfig) (*ServerChanChannel, error) {
	if cfg.SendKey == "" {
		return nil, fmt.Errorf("ServerChan SendKey 未配置")
	}
	return &ServerChanChannel{cfg: cfg}, nil
}

// Type 返回通道类型
func (s *ServerChanChannel) Type() string {
	return "serverchan"
}

// Send 发送 ServerChan 通知
func (s *ServerChanChannel) Send(msg Message) error {
	title := msg.From
	if title == "" {
		title = "未知号码"
	}
	markdown := msg.String()

	resp, err := serverchan_sdk.ScSend(s.cfg.SendKey, title, markdown, nil)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("ServerChan 返回空响应")
	}
	if resp.Code != 0 {
		return fmt.Errorf("ServerChan 返回错误: %s", resp.Message)
	}

	return nil
}
