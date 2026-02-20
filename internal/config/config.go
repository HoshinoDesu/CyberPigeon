package config

import (
	"fmt"
	"os"

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

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建配置文件: %w", err)
	}
	defer f.Close()

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
		switch ch.Type {
		case "email":
			saveConfig.Channels = append(saveConfig.Channels, struct {
				Type              string   `toml:"type"`
				Enabled           bool     `toml:"enabled"`
				RequestTimeoutSec int      `toml:"request_timeout_sec,omitempty"`
				Host              string   `toml:"host"`
				Port              int      `toml:"port"`
				Username          string   `toml:"username"`
				Password          string   `toml:"password"`
				From              string   `toml:"from"`
				To                []string `toml:"to"`
				UseTLS            bool     `toml:"use_tls,omitempty"`
			}{
				Type:              ch.Type,
				Enabled:           ch.Enabled,
				RequestTimeoutSec: ch.RequestTimeoutSec,
				Host:              ch.Host,
				Port:              ch.Port,
				Username:          ch.Username,
				Password:          ch.Password,
				From:              ch.From,
				To:                ch.To,
				UseTLS:            ch.UseTLS,
			})
		case "bark":
			saveConfig.Channels = append(saveConfig.Channels, struct {
				Type              string `toml:"type"`
				Enabled           bool   `toml:"enabled"`
				RequestTimeoutSec int    `toml:"request_timeout_sec,omitempty"`
				Endpoint          string `toml:"endpoint"`
				Title             string `toml:"title,omitempty"`
			}{
				Type:              ch.Type,
				Enabled:           ch.Enabled,
				RequestTimeoutSec: ch.RequestTimeoutSec,
				Endpoint:          ch.Endpoint,
				Title:             ch.Title,
			})
		case "gotify":
			saveConfig.Channels = append(saveConfig.Channels, struct {
				Type              string `toml:"type"`
				Enabled           bool   `toml:"enabled"`
				RequestTimeoutSec int    `toml:"request_timeout_sec,omitempty"`
				Endpoint          string `toml:"endpoint"`
				Token             string `toml:"token"`
				Priority          int    `toml:"priority,omitempty"`
			}{
				Type:              ch.Type,
				Enabled:           ch.Enabled,
				RequestTimeoutSec: ch.RequestTimeoutSec,
				Endpoint:          ch.Endpoint,
				Token:             ch.Token,
				Priority:          ch.Priority,
			})
		case "serverchan":
			saveConfig.Channels = append(saveConfig.Channels, struct {
				Type              string `toml:"type"`
				Enabled           bool   `toml:"enabled"`
				RequestTimeoutSec int    `toml:"request_timeout_sec,omitempty"`
				SendKey           string `toml:"send_key"`
			}{
				Type:              ch.Type,
				Enabled:           ch.Enabled,
				RequestTimeoutSec: ch.RequestTimeoutSec,
				SendKey:           ch.SendKey,
			})
		case "webhook":
			saveConfig.Channels = append(saveConfig.Channels, struct {
				Type                string            `toml:"type"`
				Enabled             bool              `toml:"enabled"`
				RequestTimeoutSec   int               `toml:"request_timeout_sec,omitempty"`
				AllowPrivateNetwork bool              `toml:"allow_private_network,omitempty"`
				URL                 string            `toml:"url"`
				Method              string            `toml:"method"`
				Headers             map[string]string `toml:"headers,omitempty"`
			}{
				Type:                ch.Type,
				Enabled:             ch.Enabled,
				RequestTimeoutSec:   ch.RequestTimeoutSec,
				AllowPrivateNetwork: ch.AllowPrivateNetwork,
				URL:                 ch.URL,
				Method:              ch.Method,
				Headers:             ch.Headers,
			})
		default:
			// 未知类型，保存所有字段
			saveConfig.Channels = append(saveConfig.Channels, ch)
		}
	}

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(saveConfig); err != nil {
		return fmt.Errorf("编码配置文件: %w", err)
	}

	return nil
}
