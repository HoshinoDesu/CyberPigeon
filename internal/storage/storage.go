package storage

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sms-forwarder/internal/modem"
)

// Message 存储的短信消息
type Message struct {
	ID        string    `json:"id"`
	Modem     string    `json:"modem"`     // 调制解调器 IMEI
	Number    string    `json:"number"`    // 发送方号码
	Text      string    `json:"text"`      // 短信内容
	Timestamp time.Time `json:"timestamp"` // 接收时间
	Saved     time.Time `json:"saved"`     // 保存时间
}

// Storage 短信存储
type Storage struct {
	path           string
	mu             sync.RWMutex
	messages       []Message
	ids            map[string]bool // 用于快速查重
	messageHandler func(Message)   // 新消息回调
}

// New 创建存储实例
func New(path string) (*Storage, error) {
	s := &Storage{
		path:     path,
		messages: []Message{},
		ids:      make(map[string]bool),
	}

	// 尝试加载已有数据
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("加载存储文件: %w", err)
	}

	return s, nil
}

// GenerateID 生成消息唯一 ID
func GenerateID(modemIMEI string, sms *modem.SMS) string {
	// 使用 MD5(IMEI + DBusPath + TimestampNano + Number + Text) 作为唯一 ID
	// 优先借助 DBus Path 与纳秒时间，减少秒级时间戳导致的误判
	data := fmt.Sprintf("%s|%s|%d|%s|%s", modemIMEI, sms.Path(), sms.Timestamp.UnixNano(), sms.Number, sms.Text)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Save 保存短信
func (s *Storage) Save(modemIMEI string, sms *modem.SMS) error {
	s.mu.Lock()

	id := GenerateID(modemIMEI, sms)

	// 检查是否已存在
	if s.ids[id] {
		return nil // 已存在，跳过
	}

	msg := Message{
		ID:        id,
		Modem:     modemIMEI,
		Number:    sms.Number,
		Text:      sms.Text,
		Timestamp: sms.Timestamp,
		Saved:     time.Now(),
	}

	s.messages = append(s.messages, msg)
	s.ids[id] = true
	handler := s.messageHandler
	s.mu.Unlock()

	// 触发新消息回调（在解锁后执行，避免阻塞）
	if handler != nil {
		go handler(msg)
	}

	return s.save()
}

// Has 检查消息是否存在
func (s *Storage) Has(modemIMEI string, sms *modem.SMS) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id := GenerateID(modemIMEI, sms)
	return s.ids[id]
}

// SetMessageHandler 设置新消息处理器
func (s *Storage) SetMessageHandler(handler func(Message)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageHandler = handler
}

// List 列出所有短信
func (s *Storage) List() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// ListWithPagination 分页获取短信 (按时间倒序)
func (s *Storage) ListWithPagination(limit, offset int) ([]Message, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := int64(len(s.messages))
	if total == 0 {
		return []Message{}, 0
	}

	// 既然需要倒序（最新的在前），offset=0 代表 slice 的最后一个元素
	// index = len - 1 - offset
	start := int64(len(s.messages)) - 1 - int64(offset)
	if start < 0 {
		return []Message{}, total
	}

	end := start - int64(limit) + 1
	if end < 0 {
		end = 0
	}

	// 收集结果
	count := start - end + 1
	result := make([]Message, 0, count)
	for i := start; i >= end; i-- {
		result = append(result, s.messages[i])
	}

	return result, total
}

// Delete 删除短信
func (s *Storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, msg := range s.messages {
		if msg.ID == id {
			// 删除该消息
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			delete(s.ids, id)
			return s.save()
		}
	}
	return fmt.Errorf("消息不存在")
}

// load 从文件加载
func (s *Storage) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &s.messages); err != nil {
		return err
	}

	// 重建索引
	s.ids = make(map[string]bool)
	for _, msg := range s.messages {
		s.ids[msg.ID] = true
	}

	return nil
}

// save 保存到文件
func (s *Storage) save() error {
	data, err := json.MarshalIndent(s.messages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0600)
}

// Close 关闭存储
func (s *Storage) Close() error {
	return nil
}
