package notifier

import (
	"fmt"
	"time"

	serverchan_sdk "github.com/easychen/serverchan-sdk-golang"
	"github.com/CyberPigeon/internal/config"
)

// ServerChanChannel ServerChan 通道
type ServerChanChannel struct {
	cfg     config.ChannelConfig
	timeout time.Duration
}

// NewServerChanChannel 创建 ServerChan 通道
func NewServerChanChannel(cfg config.ChannelConfig) (*ServerChanChannel, error) {
	if cfg.SendKey == "" {
		return nil, fmt.Errorf("ServerChan SendKey 未配置")
	}
	return &ServerChanChannel{
		cfg:     cfg,
		timeout: requestTimeout(cfg),
	}, nil
}

// Type 返回通道类型
func (s *ServerChanChannel) Type() string {
	return "serverchan"
}

// Send 发送 ServerChan 通知（带超时控制）
func (s *ServerChanChannel) Send(msg Message) error {
	title := msg.From
	if title == "" {
		title = "未知号码"
	}
	markdown := msg.String()

	type result struct {
		resp *serverchan_sdk.ScSendResponse
		err  error
	}

	ch := make(chan result, 1)
	go func() {
		resp, err := serverchan_sdk.ScSend(s.cfg.SendKey, title, markdown, nil)
		ch <- result{resp, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		if r.resp == nil {
			return fmt.Errorf("ServerChan 返回空响应")
		}
		if r.resp.Code != 0 {
			return fmt.Errorf("ServerChan 返回错误: %s", r.resp.Message)
		}
		return nil
	case <-time.After(s.timeout):
		return fmt.Errorf("ServerChan 请求超时 (%v)", s.timeout)
	}
}
