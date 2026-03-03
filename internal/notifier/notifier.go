package notifier

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/CyberPigeon/internal/config"
)

// Message 通知消息
type Message struct {
	Modem     string
	From      string
	To        string
	Timestamp time.Time
	Text      string
	Incoming  bool
}

// String 返回纯文本格式
func (m Message) String() string {
	return fmt.Sprintf(
		"%s\n\n发送人: %s\n时间: %s",
		m.displayText(),
		m.From,
		m.formatTimestamp(),
	)
}

func (m Message) displayText() string {
	if m.Text == "" {
		return "(空消息)"
	}
	return m.Text
}

func (m Message) formatTimestamp() string {
	if m.Timestamp.IsZero() {
		return "未知"
	}
	return m.Timestamp.Format("2006-01-02 15:04:05")
}

// Notifier 通知器
type Notifier struct {
	channels []Channel
}

// New 创建通知器
func New(cfg *config.Config) (*Notifier, error) {
	channels := make([]Channel, 0, len(cfg.Channels))

	for _, chCfg := range cfg.Channels {
		if !chCfg.Enabled {
			slog.Info("跳过未启用的通道", "type", chCfg.Type)
			continue
		}
		ch, err := createChannel(chCfg)
		if err != nil {
			slog.Error("创建通道失败", "type", chCfg.Type, "error", err)
			continue
		}
		channels = append(channels, ch)
		slog.Info("已加载通道", "type", chCfg.Type)
	}

	return &Notifier{
		channels: channels,
	}, nil
}

// Send 并发发送通知到所有通道
func (n *Notifier) Send(msg Message) error {
	if len(n.channels) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errs := make([]error, len(n.channels))

	for i, ch := range n.channels {
		wg.Add(1)
		go func(idx int, c Channel) {
			defer wg.Done()
			if err := c.Send(msg); err != nil {
				slog.Error("通道发送失败", "type", c.Type(), "error", err)
				errs[idx] = err
			} else {
				slog.Info("通知已发送", "type", c.Type())
			}
		}(i, ch)
	}

	wg.Wait()

	// 返回最后一个错误
	var lastErr error
	for _, err := range errs {
		if err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// createChannel 根据配置创建通道
func createChannel(cfg config.ChannelConfig) (Channel, error) {
	switch cfg.Type {
	case "email":
		return NewEmailChannel(cfg)
	case "bark":
		return NewBarkChannel(cfg)
	case "gotify":
		return NewGotifyChannel(cfg)
	case "serverchan":
		return NewServerChanChannel(cfg)
	case "webhook":
		return NewWebhookChannel(cfg)
	case "wecom":
		return NewWeComChannel(cfg)
	default:
		return nil, fmt.Errorf("未知通道类型: %s", cfg.Type)
	}
}

// Channel 通知通道接口
type Channel interface {
	Type() string
	Send(msg Message) error
}
