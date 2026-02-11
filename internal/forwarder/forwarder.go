package forwarder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/sms-forwarder/internal/config"
	"github.com/sms-forwarder/internal/modem"
	"github.com/sms-forwarder/internal/notifier"
	"github.com/sms-forwarder/internal/storage"
)

// Forwarder 短信转发器
type Forwarder struct {
	cfg       *config.Config
	manager   *modem.Manager
	storage   *storage.Storage
	notifier  *notifier.Notifier
	mu        sync.Mutex
	cancels   map[dbus.ObjectPath]context.CancelFunc
	equipment map[string]dbus.ObjectPath
	modems    map[dbus.ObjectPath]string
	processed map[string]time.Time // key: timestamp_from_text
}

// New 创建转发器
func New(cfg *config.Config, manager *modem.Manager, store *storage.Storage) (*Forwarder, error) {
	notif, err := notifier.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建通知器: %w", err)
	}

	return &Forwarder{
		cfg:       cfg,
		manager:   manager,
		storage:   store,
		notifier:  notif,
		cancels:   make(map[dbus.ObjectPath]context.CancelFunc),
		equipment: make(map[string]dbus.ObjectPath),
		modems:    make(map[dbus.ObjectPath]string),
		processed: make(map[string]time.Time),
	}, nil
}

// Run 运行转发器
func (f *Forwarder) Run(ctx context.Context) error {
	if len(f.cfg.Channels) == 0 && (f.storage == nil || !f.cfg.Storage.Enabled) {
		slog.Info("未配置任何通道或存储，转发器将不执行任何操作")
		<-ctx.Done()
		return nil
	}

	// 获取现有调制解调器
	modems, err := f.manager.Modems()
	if err != nil {
		return fmt.Errorf("获取调制解调器列表: %w", err)
	}
	for path, m := range modems {
		f.addModem(ctx, path, m)
	}

	// 订阅调制解调器事件
	unsubscribe, err := f.manager.Subscribe(func(event modem.ModemEvent) error {
		switch event.Type {
		case modem.ModemEventAdded:
			if event.Modem == nil {
				return nil
			}
			f.addModem(ctx, event.Path, event.Modem)
		case modem.ModemEventRemoved:
			f.removeModem(event.Path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("订阅调制解调器事件: %w", err)
	}
	defer unsubscribe()

	<-ctx.Done()
	f.stopAll()
	return nil
}

// addModem 添加调制解调器
func (f *Forwarder) addModem(ctx context.Context, path dbus.ObjectPath, m *modem.Modem) {
	if ctx.Err() != nil {
		return
	}

	f.mu.Lock()
	// 处理重复设备
	var oldCancel context.CancelFunc
	if m.EquipmentIdentifier != "" {
		if existingPath, ok := f.equipment[m.EquipmentIdentifier]; ok && existingPath != path {
			oldCancel = f.cancels[existingPath]
			delete(f.cancels, existingPath)
			delete(f.modems, existingPath)
			delete(f.equipment, m.EquipmentIdentifier)
			slog.Info("检测到重复设备，取消旧订阅", "imei", m.EquipmentIdentifier, "old_path", existingPath, "new_path", path)
		}
	}

	// 检查是否已订阅
	if _, ok := f.cancels[path]; ok {
		f.mu.Unlock()
		return
	}

	modemCtx, cancel := context.WithCancel(ctx)
	f.cancels[path] = cancel
	if m.EquipmentIdentifier != "" {
		f.equipment[m.EquipmentIdentifier] = path
		f.modems[path] = m.EquipmentIdentifier
	}
	f.mu.Unlock()

	// 立即取消旧订阅
	if oldCancel != nil {
		oldCancel()
	}

	slog.Info("订阅调制解调器", "imei", m.EquipmentIdentifier, "model", m.Model)

	// 启动订阅
	go func() {
		if err := m.Messaging().Subscribe(modemCtx, func(message *modem.SMS) error {
			return f.handleMessage(m, message)
		}); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("调制解调器订阅停止", "error", err, "imei", m.EquipmentIdentifier)
		}
		f.removeModem(path)
	}()
}

// removeModem 移除调制解调器
func (f *Forwarder) removeModem(path dbus.ObjectPath) {
	var cancel context.CancelFunc
	f.mu.Lock()
	cancel = f.cancels[path]
	delete(f.cancels, path)
	if equipmentID, ok := f.modems[path]; ok {
		delete(f.modems, path)
		delete(f.equipment, equipmentID)
		slog.Info("移除调制解调器", "imei", equipmentID)
	}
	f.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// stopAll 停止所有订阅
func (f *Forwarder) stopAll() {
	f.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(f.cancels))
	for _, cancel := range f.cancels {
		cancels = append(cancels, cancel)
	}
	f.cancels = make(map[dbus.ObjectPath]context.CancelFunc)
	f.equipment = make(map[string]dbus.ObjectPath)
	f.modems = make(map[dbus.ObjectPath]string)
	f.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

// handleMessage 处理接收到的短信
func (f *Forwarder) handleMessage(m *modem.Modem, sms *modem.SMS) error {
	// 只处理接收的短信，跳过发送的短信
	if sms.State == modem.SMSStateSent || sms.State == modem.SMSStateSending {
		slog.Debug("跳过已发送的短信", "state", sms.State.String(), "from", sms.Number)
		return nil
	}

	// incoming := sms.State == modem.SMSStateReceived || sms.State == modem.SMSStateReceiving

	smsPath := sms.Path()
	slog.Info("handleMessage调用", "path", smsPath, "from", sms.Number, "state", sms.State.String(), "text_preview", sms.Text[:min(20, len(sms.Text))])

	// 优先过滤旧消息（避免重启后重复发送）
	// if incoming && !sms.Timestamp.IsZero() && time.Since(sms.Timestamp) > 5*time.Minute {
	// 	slog.Info("跳过 5 分钟前的短信", "timestamp", sms.Timestamp, "imei", m.EquipmentIdentifier)
	// 	return nil
	// }

	// 检查存储中是否已存在（持久化去重）
	if f.storage != nil && f.cfg.Storage.Enabled {
		if f.storage.Has(m.EquipmentIdentifier, sms) {
			slog.Debug("跳过已存储的重复短信", "timestamp", sms.Timestamp, "from", sms.Number)
			return nil
		}
	}

	if f.isDuplicateSMS(sms) {
		slog.Debug("跳过重复短信", "path", sms.Path(), "from", sms.Number)
		return nil
	}

	slog.Info("收到短信",
		"imei", m.EquipmentIdentifier,
		"from", sms.Number,
		"timestamp", sms.Timestamp,
		"text", sms.Text,
	)

	// 保存到存储
	if f.storage != nil && f.cfg.Storage.Enabled {
		if err := f.storage.Save(m.EquipmentIdentifier, sms); err != nil {
			slog.Error("保存短信失败", "error", err)
		}
	}

	// 转发到通知通道
	if len(f.cfg.Channels) > 0 {
		msg := f.formatMessage(m, sms)
		if err := f.notifier.Send(msg); err != nil {
			slog.Error("发送通知失败", "error", err)
			return err
		}
	}

	return nil
}

func (f *Forwarder) isDuplicateSMS(sms *modem.SMS) bool {
	if sms == nil {
		return false
	}

	// 使用 timestamp+from+text 作为唯一标识
	key := fmt.Sprintf("%d_%s_%s", sms.Timestamp.Unix(), sms.Number, sms.Text)

	now := time.Now()
	cutoff := now.Add(-2 * time.Hour)

	f.mu.Lock()
	defer f.mu.Unlock()

	// 清理过期记录
	for k, t := range f.processed {
		if t.Before(cutoff) {
			delete(f.processed, k)
		}
	}

	if lastTime, exists := f.processed[key]; exists {
		slog.Info("检测到重复短信", "key", key, "last_processed", lastTime, "gap_ms", now.Sub(lastTime).Milliseconds())
		return true
	}
	f.processed[key] = now
	slog.Debug("短信首次处理", "key", key)
	return false
}

// formatMessage 格式化消息
func (f *Forwarder) formatMessage(m *modem.Modem, sms *modem.SMS) notifier.Message {
	incoming := sms.State == modem.SMSStateReceived || sms.State == modem.SMSStateReceiving
	sender, recipient := sms.Number, m.Number
	if !incoming {
		sender, recipient = recipient, sender
	}

	modemName := f.getModemName(m)

	return notifier.Message{
		Modem:     modemName,
		From:      sender,
		To:        recipient,
		Timestamp: sms.Timestamp,
		Text:      strings.TrimSpace(sms.Text),
		Incoming:  incoming,
	}
}

// getModemName 获取调制解调器名称
func (f *Forwarder) getModemName(m *modem.Modem) string {
	if m.Model != "" {
		return strings.TrimSpace(m.Model)
	}
	return m.EquipmentIdentifier
}

// GetModems 获取所有调制解调器信息
func (f *Forwarder) GetModems() []*modem.Modem {
	f.mu.Lock()
	defer f.mu.Unlock()

	modems := make([]*modem.Modem, 0, len(f.modems))
	for path := range f.modems {
		if m, err := modem.NewModem(f.manager.Conn(), path); err == nil {
			modems = append(modems, m)
		}
	}
	return modems
}
