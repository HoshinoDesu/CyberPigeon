# SMS Forwarder - AI编程助手指南

## 项目概览

SMS Forwarder 是一个用 Go 编写的短信转发工具，从 ModemManager (Linux D-Bus) 接收 SMS，支持存储到 JSON 并转发到多个通知渠道。

**关键依赖**: `godbus/dbus/v5` (D-Bus 交互), `BurntSushi/toml` (配置解析)

## 架构组件

```
main.go → forwarder → [modem/manager + storage + notifier]
                          ↓                        ↓
                    D-Bus/ModemManager      多通道发送 (email/bark/gotify/serverchan/webhook)
```

### 数据流向

1. **Modem Manager** ([modem/manager.go](internal/modem/manager.go)) 通过 D-Bus 连接 ModemManager，订阅设备添加/移除事件
2. **Messaging** ([modem/messaging.go](internal/modem/messaging.go)) 为每个调制解调器创建独立 D-Bus 连接，订阅 SMS "Added" 信号
3. **Forwarder** ([forwarder/forwarder.go](internal/forwarder/forwarder.go)) 协调所有组件：
   - 维护设备 `ObjectPath` 到 IMEI 的映射 (`equipment`, `modems` maps)
   - 处理设备插拔 (hotplug)，通过 `cancels` map 管理 goroutine 生命周期
   - 去重设备：同一 IMEI 只保留一个活动订阅
4. **Storage** ([storage/storage.go](internal/storage/storage.go)) 使用互斥锁保护 JSON 文件的读写操作
5. **Notifier** ([notifier/notifier.go](internal/notifier/notifier.go)) 遍历所有通道发送通知，记录失败但继续尝试后续通道

## D-Bus 交互模式

**关键点**: ModemManager 使用 D-Bus Object Manager 接口

- **系统总线连接**: 所有操作都在 `dbus.SystemBus()` 上进行
- **独立订阅连接**: [messaging.go#L39-L52](internal/modem/messaging.go#L39-L52) 为每个调制解调器的 SMS 订阅创建独立的 `dbus.SystemBusPrivate()` 连接，确保信号隔离
- **信号匹配**: 使用 `WithMatchPathNamespace` 限制信号范围，避免接收无关事件
- **属性读取**: `GetProperty()` 获取 `EquipmentIdentifier`, `Model` 等属性

## 配置模式

[config.example.toml](config.example.toml) 展示了配置结构：

- **单一配置文件**: TOML 格式，支持 `[[channels]]` 数组定义多个通知渠道
- **通道类型**: 所有通道共用 `ChannelConfig` 结构体，通过 `type` 字段区分 (email/bark/gotify/serverchan/webhook)
- **可选存储**: `storage.enabled` 控制是否保存到 JSON 文件
- **调制解调器别名**: 可选的 `[[modems]]` 数组通过 IMEI 设置设备别名

## 并发与生命周期管理

- **Context 驱动**: [main.go#L49-L57](main.go#L49-L57) 使用 `context.WithCancel` 管理整个应用生命周期
- **每设备 Context**: [forwarder.go#L110-L113](internal/forwarder/forwarder.go#L110-L113) 为每个调制解调器创建独立的子 context
- **设备热插拔**: `addModem` 检测重复设备时先 `cancel()` 旧订阅再创建新订阅
- **优雅关闭**: [main.go#L62-L68](main.go#L62-L68) 捕获 SIGINT/SIGTERM 信号后取消 context，触发所有 goroutine 退出

## 常见模式

### 添加新通知渠道

1. 在 [notifier/](internal/notifier/) 创建 `<type>.go` (参考 [bark.go](internal/notifier/bark.go) 的简单实现)
2. 实现 `Channel` 接口: `Send(Message) error` 和 `Type() string`
3. 在 [notifier.go#L88](internal/notifier/notifier.go#L88) 的 `createChannel()` switch 中添加 case
4. 在 [config.go](internal/config/config.go) 的 `ChannelConfig` 添加所需配置字段
5. 更新 [config.example.toml](config.example.toml) 添加示例配置

### D-Bus 调试

- **查看可用调制解调器**: `mmcli -L`
- **查看短信**: `mmcli -m <modem-id> --messaging-list-sms`
- **监控 D-Bus 信号**: `dbus-monitor --system "type='signal',interface='org.freedesktop.DBus.ObjectManager'"`

## 运行与构建

```bash
# 开发运行
go run main.go -config config.toml

# 构建
go build -o sms-forwarder

# 测试 (需要 ModemManager)
go test ./...
```

## 特殊注意

- **仅限 Linux**: 依赖 ModemManager D-Bus 接口，不支持 Windows/macOS
- **设备去重**: [forwarder.go#L95-L104](internal/forwarder/forwarder.go#L95-L104) 使用 `EquipmentIdentifier` (IMEI) 作为唯一标识，防止同一设备重复订阅
- **异步通知**: 所有通知通道并行发送，一个失败不影响其他通道
- **SMS 接收延迟**: [messaging.go#L85](internal/modem/messaging.go#L85) 调用 `waitForSMSReceived` 等待 SMS 状态变为已接收 (D-Bus 信号和属性更新有时序差异)
