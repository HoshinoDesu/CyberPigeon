package notifier

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/HoshinoDesu/CyberPigeon/internal/config"
)

// DingtalkChannel 钉钉机器人通道
type DingtalkChannel struct {
	cfg    config.ChannelConfig
	client *http.Client
}

// NewDingtalkChannel 创建钉钉通道
func NewDingtalkChannel(cfg config.ChannelConfig) (*DingtalkChannel, error) {
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("钉钉 webhook_url 未配置")
	}
	return &DingtalkChannel{cfg: cfg, client: newHTTPClient(cfg)}, nil
}

// Type 返回通道类型
func (d *DingtalkChannel) Type() string {
	return "dingtalk"
}

// Send 发送钉钉通知
func (d *DingtalkChannel) Send(msg Message) error {
	title := "短信通知"
	if d.cfg.Title != "" {
		title = d.cfg.Title
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("[%s] %s", title, msg.String()),
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	webhookURL := d.cfg.WebhookURL

	// 如果配置了签名密钥，计算签名并附加到 URL
	if d.cfg.SignSecret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		sign := dingtalkSign(timestamp, d.cfg.SignSecret)
		webhookURL = fmt.Sprintf("%s&timestamp=%s&sign=%s", webhookURL, timestamp, sign)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送钉钉消息失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("钉钉返回错误: errcode=%d, errmsg=%s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// dingtalkSign 计算钉钉自定义机器人签名
func dingtalkSign(timestamp, secret string) string {
	stringToSign := timestamp + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	signData := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(signData)
}
