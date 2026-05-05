package notifier

import (
	"context"
	"encoding/json"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"github.com/HoshinoDesu/CyberPigeon/internal/config"
)

// FeishuChannel 飞书通道（使用飞书应用 SDK）
type FeishuChannel struct {
	cfg    config.ChannelConfig
	client *lark.Client
}

// NewFeishuChannel 创建飞书通道
func NewFeishuChannel(cfg config.ChannelConfig) (*FeishuChannel, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("飞书 app_id 未配置")
	}
	if cfg.AppSecret == "" {
		return nil, fmt.Errorf("飞书 app_secret 未配置")
	}
	if cfg.ReceiveID == "" {
		return nil, fmt.Errorf("飞书 receive_id 未配置")
	}
	if cfg.ReceiveIDType == "" {
		cfg.ReceiveIDType = "open_id"
	}

	client := lark.NewClient(cfg.AppID, cfg.AppSecret)

	return &FeishuChannel{cfg: cfg, client: client}, nil
}

// Type 返回通道类型
func (f *FeishuChannel) Type() string {
	return "feishu"
}

// Send 发送飞书通知
func (f *FeishuChannel) Send(msg Message) error {
	title := "短信通知"
	if f.cfg.Title != "" {
		title = f.cfg.Title
	}

	content, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf("[%s] %s", title, msg.String()),
	})

	ctx := context.Background()
	if timeout := requestTimeout(f.cfg); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(f.cfg.ReceiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(f.cfg.ReceiveID).
			MsgType("text").
			Content(string(content)).
			Build()).
		Build()

	resp, err := f.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("发送飞书消息失败: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("飞书返回错误: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return nil
}
