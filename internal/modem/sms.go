package modem

import (
	"time"

	"github.com/godbus/dbus/v5"
)

const ModemSMSInterface = ModemManagerInterface + ".Sms"

// SMSState 短信状态
type SMSState uint32

const (
	SMSStateUnknown   SMSState = iota // 未知状态
	SMSStateStored                    // 已存储
	SMSStateReceiving                 // 接收中
	SMSStateReceived                  // 已接收
	SMSStateSending                   // 发送中
	SMSStateSent                      // 已发送
)

// String 返回状态描述
func (s SMSState) String() string {
	switch s {
	case SMSStateUnknown:
		return "unknown"
	case SMSStateStored:
		return "stored"
	case SMSStateReceiving:
		return "receiving"
	case SMSStateReceived:
		return "received"
	case SMSStateSending:
		return "sending"
	case SMSStateSent:
		return "sent"
	default:
		return "unknown"
	}
}

// SMS 短信
type SMS struct {
	objectPath dbus.ObjectPath
	State      SMSState
	Number     string
	Text       string
	Timestamp  time.Time
}

// Path 返回 DBus 路径
func (sms *SMS) Path() dbus.ObjectPath {
	return sms.objectPath
}

// retrieveWithConn 使用指定连接获取短信
func (msg *Messaging) retrieveWithConn(conn *dbus.Conn, objectPath dbus.ObjectPath) (*SMS, error) {
	obj := conn.Object(ModemManagerInterface, objectPath)

	sms := SMS{objectPath: objectPath}

	// 获取状态
	variant, err := obj.GetProperty(ModemSMSInterface + ".State")
	if err != nil {
		return nil, err
	}
	sms.State = SMSState(variant.Value().(uint32))

	// 获取号码
	variant, err = obj.GetProperty(ModemSMSInterface + ".Number")
	if err != nil {
		return nil, err
	}
	sms.Number = variant.Value().(string)

	// 获取文本
	variant, err = obj.GetProperty(ModemSMSInterface + ".Text")
	if err != nil {
		return nil, err
	}
	sms.Text = variant.Value().(string)

	// 获取时间戳
	variant, err = obj.GetProperty(ModemSMSInterface + ".Timestamp")
	if err != nil {
		return nil, err
	}
	if t := variant.Value().(string); t != "" {
		// 处理时区格式（+08:00 需要转为 +08:00）
		if len(t) >= 3 && (t[len(t)-3] == '+' || t[len(t)-3] == '-') {
			t = t + ":00"
		}
		sms.Timestamp, err = time.Parse(time.RFC3339, t)
		if err != nil {
			return nil, err
		}
	}

	return &sms, nil
}
