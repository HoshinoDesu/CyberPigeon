package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config 主配置结构
type Config struct {
	Storage  StorageConfig   `toml:"storage"`
	Server   ServerConfig    `toml:"server"`
	Channels []ChannelConfig `toml:"channels"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Enabled        bool     `toml:"enabled"`         // 是否启用 Web 服务器
	Listen         string   `toml:"listen"`          // 监听地址，如 ":8080"
	AllowedOrigins []string `toml:"allowed_origins"` // 允许的 WebSocket Origin 列表，空则仅允许同源
}

// ChannelConfig 通道配置
type ChannelConfig struct {
	Type    string `toml:"type" json:"type"`       // email, bark, gotify, serverchan, pushdeer, webhook
	Enabled bool   `toml:"enabled" json:"enabled"` // 是否启用，默认 false

	// Common
	RequestTimeoutSec   int  `toml:"request_timeout_sec" json:"request_timeout_sec"`     // HTTP 请求超时秒数，默认 10
	AllowPrivateNetwork bool `toml:"allow_private_network" json:"allow_private_network"` // 是否允许访问私网地址（仅 webhook）

	// Email
	Host     string   `toml:"host" json:"host"`
	Port     int      `toml:"port" json:"port"`
	Username string   `toml:"username" json:"username"`
	Password string   `toml:"password" json:"password"`
	From     string   `toml:"from" json:"from"`
	To       []string `toml:"to" json:"to"`
	UseTLS   bool     `toml:"use_tls" json:"use_tls"`

	// Bark
	Endpoint string `toml:"endpoint" json:"endpoint"`
	Title    string `toml:"title" json:"title"`

	// Gotify
	Token    string `toml:"token" json:"token"`
	Priority int    `toml:"priority" json:"priority"`

	// ServerChan
	SendKey string `toml:"send_key" json:"send_key"`

	// Webhook
	URL     string            `toml:"url" json:"url"`
	Method  string            `toml:"method" json:"method"`
	Headers map[string]string `toml:"headers" json:"headers"`

	// WeCom (企业微信)
	CorpID     string `toml:"corp_id" json:"corp_id"`
	CorpSecret string `toml:"corp_secret" json:"corp_secret"`
	AgentID    int    `toml:"agent_id" json:"agent_id"`
	ToUser     string `toml:"to_user" json:"to_user"`
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件: %w", err)
	}

	return &cfg, nil
}

// channelToMap 将通道配置转换为 map，只包含该类型通道的相关字段
func channelToMap(ch ChannelConfig) map[string]any {
	m := map[string]any{
		"type":    ch.Type,
		"enabled": ch.Enabled,
	}
	if ch.RequestTimeoutSec > 0 {
		m["request_timeout_sec"] = ch.RequestTimeoutSec
	}

	switch ch.Type {
	case "email":
		m["host"] = ch.Host
		m["port"] = ch.Port
		m["username"] = ch.Username
		m["password"] = ch.Password
		m["from"] = ch.From
		m["to"] = ch.To
		if ch.UseTLS {
			m["use_tls"] = ch.UseTLS
		}
	case "bark":
		m["endpoint"] = ch.Endpoint
		if ch.Title != "" {
			m["title"] = ch.Title
		}
	case "gotify":
		m["endpoint"] = ch.Endpoint
		m["token"] = ch.Token
		if ch.Priority != 0 {
			m["priority"] = ch.Priority
		}
	case "serverchan":
		m["send_key"] = ch.SendKey
	case "webhook":
		m["url"] = ch.URL
		m["method"] = ch.Method
		if ch.AllowPrivateNetwork {
			m["allow_private_network"] = ch.AllowPrivateNetwork
		}
		if len(ch.Headers) > 0 {
			m["headers"] = ch.Headers
		}
	case "wecom":
		m["corp_id"] = ch.CorpID
		m["corp_secret"] = ch.CorpSecret
		m["agent_id"] = ch.AgentID
		if ch.ToUser != "" {
			m["to_user"] = ch.ToUser
		}
	default:
		// 未知类型：序列化所有非零字段
		m["host"] = ch.Host
		m["port"] = ch.Port
		m["username"] = ch.Username
		m["password"] = ch.Password
		m["from"] = ch.From
		m["to"] = ch.To
		m["endpoint"] = ch.Endpoint
		m["token"] = ch.Token
		m["send_key"] = ch.SendKey
		m["url"] = ch.URL
		m["method"] = ch.Method
	}
	return m
}

// Save 保存配置到文件（原子写入：先写临时文件，再重命名）
func (c *Config) Save(path string) error {
	// 在目标文件同目录创建临时文件，确保同一文件系统以支持原子 Rename
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".config-*.toml.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	tmpPath := tmpFile.Name()

	// 构建用于保存的配置，过滤掉每个通道不相关的字段
	saveConfig := struct {
		Storage  StorageConfig `toml:"storage"`
		Server   ServerConfig  `toml:"server"`
		Channels []any         `toml:"channels"`
	}{
		Storage:  c.Storage,
		Server:   c.Server,
		Channels: make([]any, 0, len(c.Channels)),
	}

	for _, ch := range c.Channels {
		saveConfig.Channels = append(saveConfig.Channels, channelToMap(ch))
	}

	encoder := toml.NewEncoder(tmpFile)
	if err := encoder.Encode(saveConfig); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("编码配置文件: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("关闭临时文件: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("重命名配置文件: %w", err)
	}

	return nil
}
