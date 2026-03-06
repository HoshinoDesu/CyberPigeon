package modem

import (
	"fmt"
	"regexp"
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
	state, ok := variant.Value().(uint32)
	if !ok {
		return nil, fmt.Errorf("短信状态字段类型错误")
	}
	sms.State = SMSState(state)

	// 获取号码
	variant, err = obj.GetProperty(ModemSMSInterface + ".Number")
	if err != nil {
		return nil, err
	}
	number, ok := variant.Value().(string)
	if !ok {
		return nil, fmt.Errorf("短信号码字段类型错误")
	}
	sms.Number = number

	// 获取文本
	variant, err = obj.GetProperty(ModemSMSInterface + ".Text")
	if err != nil {
		return nil, err
	}
	text, ok := variant.Value().(string)
	if !ok {
		return nil, fmt.Errorf("短信文本字段类型错误")
	}
	sms.Text = text

	// 获取时间戳
	variant, err = obj.GetProperty(ModemSMSInterface + ".Timestamp")
	if err != nil {
		return nil, err
	}
	t, ok := variant.Value().(string)
	if !ok {
		return nil, fmt.Errorf("短信时间字段类型错误")
	}
	if t != "" {
		// 处理短时区格式，如 "+08" -> "+08:00"、"-05" -> "-05:00"
		shortTZPattern := regexp.MustCompile(`([+-]\d{2})$`)
		t = shortTZPattern.ReplaceAllString(t, "${1}:00")
		sms.Timestamp, err = time.Parse(time.RFC3339, t)
		if err != nil {
			return nil, err
		}
	}

	return &sms, nil
}
