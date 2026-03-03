package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/CyberPigeon/internal/config"
)

// WeComChannel 企业微信通道
type WeComChannel struct {
	cfg    config.ChannelConfig
	client *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewWeComChannel 创建企业微信通道
func NewWeComChannel(cfg config.ChannelConfig) (*WeComChannel, error) {
	if cfg.CorpID == "" {
		return nil, fmt.Errorf("企业微信 corp_id 未配置")
	}
	if cfg.CorpSecret == "" {
		return nil, fmt.Errorf("企业微信 corp_secret 未配置")
	}
	if cfg.AgentID == 0 {
		return nil, fmt.Errorf("企业微信 agent_id 未配置")
	}
	return &WeComChannel{cfg: cfg, client: newHTTPClient(cfg)}, nil
}

// Type 返回通道类型
func (w *WeComChannel) Type() string {
	return "wecom"
}

// Send 发送企业微信通知
func (w *WeComChannel) Send(msg Message) error {
	return w.doSend(msg, true)
}

// doSend 执行实际发送，allowRetry 控制 token 过期时是否自动重试一次
func (w *WeComChannel) doSend(msg Message, allowRetry bool) error {
	token, err := w.getAccessToken()
	if err != nil {
		return fmt.Errorf("获取 access_token 失败: %w", err)
	}

	toUser := w.cfg.ToUser
	if toUser == "" {
		toUser = "@all"
	}

	payload := map[string]interface{}{
		"touser":  toUser,
		"msgtype": "text",
		"agentid": w.cfg.AgentID,
		"text": map[string]string{
			"content": msg.String(),
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	sendURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	req, err := http.NewRequest(http.MethodPost, sendURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送企业微信消息失败: %w", err)
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
		// access_token 过期或无效时，清除缓存并自动重试一次
		if (result.ErrCode == 40014 || result.ErrCode == 42001) && allowRetry {
			w.mu.Lock()
			w.accessToken = ""
			w.tokenExpiry = time.Time{}
			w.mu.Unlock()
			return w.doSend(msg, false)
		}
		return fmt.Errorf("企业微信返回错误: %d %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// getAccessToken 获取 access_token，带缓存（提前 5 分钟刷新）
func (w *WeComChannel) getAccessToken() (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 缓存仍然有效
	if w.accessToken != "" && time.Now().Before(w.tokenExpiry) {
		return w.accessToken, nil
	}

	tokenURL := fmt.Sprintf(
		"https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		url.QueryEscape(w.cfg.CorpID), url.QueryEscape(w.cfg.CorpSecret),
	)

	resp, err := w.client.Get(tokenURL)
	if err != nil {
		return "", fmt.Errorf("请求 access_token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 token 响应失败: %w", err)
	}

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析 token 响应失败: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("获取 token 失败: %d %s", result.ErrCode, result.ErrMsg)
	}

	w.accessToken = result.AccessToken
	// 提前 5 分钟刷新，避免边界过期
	w.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn)*time.Second - 5*time.Minute)

	return w.accessToken, nil
}
