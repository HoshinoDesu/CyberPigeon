package modem

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	ModemManagerInterface = "org.freedesktop.ModemManager1"
	ModemManagerPath      = "/org/freedesktop/ModemManager1"
	ObjectManagerInterface = "org.freedesktop.DBus.ObjectManager"
)

// Manager 调制解调器管理器
type Manager struct {
	conn       *dbus.Conn
	dbusObject dbus.BusObject
}

// NewManager 创建管理器
func NewManager() (*Manager, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("连接系统总线: %w", err)
	}

	obj := conn.Object(ModemManagerInterface, ModemManagerPath)
	return &Manager{
		conn:       conn,
		dbusObject: obj,
	}, nil
}

// Modems 获取所有调制解调器
func (m *Manager) Modems() (map[dbus.ObjectPath]*Modem, error) {
	// 使用 ObjectManager 接口获取所有管理的对象
	managedObjects := make(map[dbus.ObjectPath]map[string]map[string]dbus.Variant)
	err := m.dbusObject.Call(ObjectManagerInterface+".GetManagedObjects", 0).Store(&managedObjects)
	if err != nil {
		return nil, fmt.Errorf("获取调制解调器列表: %w", err)
	}

	modems := make(map[dbus.ObjectPath]*Modem)
	for objectPath, interfaces := range managedObjects {
		// 检查对象是否包含 Modem 接口
		if _, hasModem := interfaces[ModemInterface]; !hasModem {
			continue
		}
		
		modem, err := NewModem(m.conn, objectPath)
		if err != nil {
			continue
		}
		modems[objectPath] = modem
	}

	return modems, nil
}

// Subscribe 订阅调制解调器事件
func (m *Manager) Subscribe(handler func(event ModemEvent) error) (func(), error) {
	if err := m.conn.AddMatchSignal(
		dbus.WithMatchInterface(ObjectManagerInterface),
		dbus.WithMatchMember("InterfacesAdded"),
		dbus.WithMatchPathNamespace(ModemManagerPath),
	); err != nil {
		return nil, err
	}

	if err := m.conn.AddMatchSignal(
		dbus.WithMatchInterface(ObjectManagerInterface),
		dbus.WithMatchMember("InterfacesRemoved"),
		dbus.WithMatchPathNamespace(ModemManagerPath),
	); err != nil {
		return nil, err
	}

	signalChan := make(chan *dbus.Signal, 10)
	m.conn.Signal(signalChan)

	go func() {
		for sig := range signalChan {
			var event ModemEvent
			event.Path = sig.Body[0].(dbus.ObjectPath)

			switch sig.Name {
			case ObjectManagerInterface + ".InterfacesAdded":
				event.Type = ModemEventAdded
				modem, err := NewModem(m.conn, event.Path)
				if err == nil {
					event.Modem = modem
				}
			case ObjectManagerInterface + ".InterfacesRemoved":
				event.Type = ModemEventRemoved
			}

			_ = handler(event)
		}
	}()

	unsubscribe := func() {
		m.conn.RemoveSignal(signalChan)
		close(signalChan)
	}

	return unsubscribe, nil
}

// Conn 返回 D-Bus 连接
func (m *Manager) Conn() *dbus.Conn {
	return m.conn
}

// ModemEventType 调制解调器事件类型
type ModemEventType int

const (
	ModemEventAdded ModemEventType = iota
	ModemEventRemoved
)

// ModemEvent 调制解调器事件
type ModemEvent struct {
	Type  ModemEventType
	Path  dbus.ObjectPath
	Modem *Modem
}
