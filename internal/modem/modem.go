package modem

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const ModemInterface = ModemManagerInterface + ".Modem"

// ModemState ModemManager 调制解调器状态
// 参考: MMModemState
type ModemState int32

const (
	ModemStateFailed        ModemState = -1
	ModemStateUnknown       ModemState = 0
	ModemStateInitializing  ModemState = 1
	ModemStateLocked        ModemState = 2
	ModemStateDisabled      ModemState = 3
	ModemStateDisabling     ModemState = 4
	ModemStateEnabling      ModemState = 5
	ModemStateEnabled       ModemState = 6
	ModemStateSearching     ModemState = 7
	ModemStateRegistered    ModemState = 8
	ModemStateDisconnecting ModemState = 9
	ModemStateConnecting    ModemState = 10
	ModemStateConnected     ModemState = 11
)

func (s ModemState) String() string {
	switch s {
	case ModemStateFailed:
		return "failed"
	case ModemStateUnknown:
		return "unknown"
	case ModemStateInitializing:
		return "initializing"
	case ModemStateLocked:
		return "locked"
	case ModemStateDisabled:
		return "disabled"
	case ModemStateDisabling:
		return "disabling"
	case ModemStateEnabling:
		return "enabling"
	case ModemStateEnabled:
		return "enabled"
	case ModemStateSearching:
		return "searching"
	case ModemStateRegistered:
		return "registered"
	case ModemStateDisconnecting:
		return "disconnecting"
	case ModemStateConnecting:
		return "connecting"
	case ModemStateConnected:
		return "connected"
	default:
		return fmt.Sprintf("state(%d)", int32(s))
	}
}

// Modem 调制解调器
type Modem struct {
	conn                *dbus.Conn
	objectPath          dbus.ObjectPath
	dbusObject          dbus.BusObject
	EquipmentIdentifier string // IMEI
	Model               string
	Manufacturer        string
	Number              string     // MSISDN
	SignalQuality       uint32     // 信号质量 (0-100)
	OperatorName        string     // 运营商名称
	ICCID               string     // SIM 卡 ICCID
	State               ModemState // 调制解调器状态
}

// NewModem 创建调制解调器实例
func NewModem(conn *dbus.Conn, path dbus.ObjectPath) (*Modem, error) {
	obj := conn.Object(ModemManagerInterface, path)

	modem := &Modem{
		conn:       conn,
		objectPath: path,
		dbusObject: obj,
	}

	// 获取设备信息
	if variant, err := obj.GetProperty(ModemInterface + ".EquipmentIdentifier"); err == nil {
		if v, ok := variant.Value().(string); ok {
			modem.EquipmentIdentifier = v
		}
	}

	if variant, err := obj.GetProperty(ModemInterface + ".Model"); err == nil {
		if v, ok := variant.Value().(string); ok {
			modem.Model = v
		}
	}

	if variant, err := obj.GetProperty(ModemInterface + ".Manufacturer"); err == nil {
		if v, ok := variant.Value().(string); ok {
			modem.Manufacturer = v
		}
	}

	// 尝试获取 MSISDN
	if variant, err := obj.GetProperty(ModemInterface + ".OwnNumbers"); err == nil {
		if numbers, ok := variant.Value().([]string); ok && len(numbers) > 0 {
			modem.Number = numbers[0]
		}
	}

	// 获取信号质量
	modem.UpdateSignalQuality()

	// 获取运营商名称
	modem.UpdateOperatorName()

	// 获取 ICCID
	modem.UpdateICCID()

	// 获取调制解调器状态
	modem.UpdateState()

	return modem, nil
}

// Path 返回 DBus 路径
func (m *Modem) Path() dbus.ObjectPath {
	return m.objectPath
}

// Messaging 返回消息接口
func (m *Modem) Messaging() *Messaging {
	return &Messaging{modem: m}
}

// String 返回调制解调器描述
func (m *Modem) String() string {
	if m.Model != "" {
		return fmt.Sprintf("%s (%s)", m.Model, m.EquipmentIdentifier)
	}
	return m.EquipmentIdentifier
}

// UpdateSignalQuality 更新信号质量
func (m *Modem) UpdateSignalQuality() {
	if variant, err := m.dbusObject.GetProperty(ModemInterface + ".SignalQuality"); err == nil {
		// SignalQuality 返回 (quality, recent) 元组
		if tuple, ok := variant.Value().([]interface{}); ok && len(tuple) >= 1 {
			if quality, ok := tuple[0].(uint32); ok {
				m.SignalQuality = quality
			}
		}
	}
}

// UpdateOperatorName 更新运营商名称
func (m *Modem) UpdateOperatorName() {
	const Modem3gppInterface = ModemManagerInterface + ".Modem.Modem3gpp"
	if variant, err := m.dbusObject.GetProperty(Modem3gppInterface + ".OperatorName"); err == nil {
		if name, ok := variant.Value().(string); ok {
			m.OperatorName = name
		}
	}
}

// UpdateICCID 更新 SIM 卡 ICCID
func (m *Modem) UpdateICCID() {
	const SimInterface = ModemManagerInterface + ".Sim"
	// 首先获取 Sim 对象路径
	if variant, err := m.dbusObject.GetProperty(ModemInterface + ".Sim"); err == nil {
		if simPath, ok := variant.Value().(dbus.ObjectPath); ok && simPath != "/" {
			// 获取 Sim 对象的 SimIdentifier 属性
			simObj := m.conn.Object(ModemManagerInterface, simPath)
			if simVariant, err := simObj.GetProperty(SimInterface + ".SimIdentifier"); err == nil {
				if iccid, ok := simVariant.Value().(string); ok {
					m.ICCID = iccid
				}
			}
		}
	}
}

// UpdateState 更新调制解调器状态
func (m *Modem) UpdateState() {
	if variant, err := m.dbusObject.GetProperty(ModemInterface + ".State"); err == nil {
		switch v := variant.Value().(type) {
		case int32:
			m.State = ModemState(v)
		case uint32:
			m.State = ModemState(int32(v))
		}
	}
}

// Enable 启用或禁用调制解调器（ModemManager Enable(b)）
func (m *Modem) Enable(enable bool) error {
	call := m.dbusObject.Call(ModemInterface+".Enable", 0, enable)
	if call.Err != nil {
		action := "禁用"
		if enable {
			action = "启用"
		}
		return fmt.Errorf("%s调制解调器失败: %w", action, call.Err)
	}
	m.UpdateState()
	return nil
}

// EnsureEnabled 若当前为 disabled，则尝试启用
func (m *Modem) EnsureEnabled() error {
	m.UpdateState()
	if m.State != ModemStateDisabled {
		return nil
	}
	if err := m.Enable(true); err != nil {
		return err
	}
	return nil
}

// RunUSSD 执行 USSD 代码
func (m *Modem) RunUSSD(code string) (string, error) {
	const UssdInterface = ModemManagerInterface + ".Modem.Modem3gpp.Ussd"

	// 调用 Initiate 方法
	// Initiate (s command) -> (s reply)
	var reply string
	err := m.dbusObject.Call(UssdInterface+".Initiate", 0, code).Store(&reply)
	if err != nil {
		return "", fmt.Errorf("USSD 执行失败: %w", err)
	}

	// 有些 Modem 可能不通过返回值返回结果，而是通过 Respond 方法或信号
	// 但通常 Initiate 会等待直到会话结束或返回
	return reply, nil
}
