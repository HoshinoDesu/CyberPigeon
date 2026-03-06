package modem

import (
	"context"
	"log/slog"
	"time"

	"github.com/godbus/dbus/v5"
)

const ModemMessagingInterface = ModemInterface + ".Messaging"

// Messaging 消息接口
type Messaging struct {
	modem *Modem
}

// List 列出所有短信
func (msg *Messaging) List() ([]*SMS, error) {
	var messages []dbus.ObjectPath
	err := msg.modem.dbusObject.Call(ModemMessagingInterface+".List", 0).Store(&messages)
	if err != nil {
		return nil, err
	}

	s := make([]*SMS, 0, len(messages))
	for _, message := range messages {
		sms, err := msg.Retrieve(message)
		if err != nil {
			slog.Error("获取短信失败", "path", message, "error", err)
			continue
		}
		s = append(s, sms)
	}
	return s, nil
}

// Subscribe 订阅新短信
func (msg *Messaging) Subscribe(ctx context.Context, subscriber func(message *SMS) error) error {
	conn, err := dbus.SystemBusPrivate()
	if err != nil {
		return err
	}
	if err := conn.Auth(nil); err != nil {
		conn.Close()
		return err
	}
	if err := conn.Hello(); err != nil {
		conn.Close()
		return err
	}

	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("关闭 DBus 连接失败", "error", err)
		}
	}()

	if err := conn.AddMatchSignal(
		dbus.WithMatchMember("Added"),
		dbus.WithMatchPathNamespace(msg.modem.objectPath),
	); err != nil {
		return err
	}

	signalChan := make(chan *dbus.Signal, 10)
	conn.Signal(signalChan)
	defer conn.RemoveSignal(signalChan)

	for {
		select {
		case sig := <-signalChan:
			if sig == nil || len(sig.Body) < 2 {
				continue
			}
			// Body[0] 是 ObjectPath, Body[1] 是 bool (received)
			received, ok := sig.Body[1].(bool)
			if !ok || !received {
				continue
			}

			path, ok := sig.Body[0].(dbus.ObjectPath)
			if !ok {
				continue
			}

			sms, err := msg.waitForSMSReceived(ctx, conn, path, 100*time.Millisecond)
			if err != nil {
				slog.Error("处理短信失败", "error", err, "path", path)
				continue
			}

			if err := subscriber(sms); err != nil {
				slog.Error("订阅者处理失败", "error", err, "path", path)
			}

		case <-ctx.Done():
			slog.Info("取消订阅短信", "modem", msg.modem.EquipmentIdentifier)
			return ctx.Err()
		}
	}
}

// waitForSMSReceived 等待短信完全接收
func (msg *Messaging) waitForSMSReceived(ctx context.Context, conn *dbus.Conn, path dbus.ObjectPath, interval time.Duration) (*SMS, error) {
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}

	const maxWait = 30 * time.Second
	deadline := time.After(maxWait)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		sms, err := msg.retrieveWithConn(conn, path)
		if err != nil {
			return nil, err
		}
		if sms.State == SMSStateReceived {
			return sms, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			// 超时后返回当前状态的短信，避免永久阻塞
			slog.Warn("等待短信接收超时，返回当前状态", "path", path, "state", sms.State.String())
			return sms, nil
		case <-ticker.C:
		}
	}
}

// Retrieve 获取短信详情
func (msg *Messaging) Retrieve(objectPath dbus.ObjectPath) (*SMS, error) {
	return msg.retrieveWithConn(msg.modem.conn, objectPath)
}

// Delete 删除短信
func (msg *Messaging) Delete(path dbus.ObjectPath) error {
	return msg.modem.dbusObject.Call(ModemMessagingInterface+".Delete", 0, path).Err
}
