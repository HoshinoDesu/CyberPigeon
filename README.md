# SMS Forwarder

一个简单的短信转发工具，基于 ModemManager 开发，支持将短信转发到多种通知渠道。

## 界面预览

![Web UI](img/ui.png)

## 功能特性

- 支持多种转发接通道：Email, Bark, Gotify, ServerChan, Webhook。
- 提供 Web 管理界面，支持查看短信列表和设备状态。
- 支持 USSD 代码执行（如查询话费）。
- 适配 Linux x64 和 Linux ARM64 平台 (仅支持 Linux 系统)。

## 已测试设备

- Sierra Wireless AirPrime® EM7430
- Qualcomm Snapdragon® 410 UFI (UFI-001C,UFI003等)

## 编译方法

使用 Go 语言进行编译：

```bash
# 编译 Linux ARM64 版本
GOOS=linux GOARCH=arm64 go build -o sms-forwarder-linux-arm64
```

## 配置说明

1. 将 `config.example.toml` 重命名为 `config.toml`。
2. 编辑 `config.toml` 文件，配置短信转发通道和相关参数。

## 运行

直接运行编译后的二进制文件：

```bash
./sms-forwarder-linux-arm64
```

程序默认监听端口可在配置文件中修改。

## 注意事项

- 程序依赖 ModemManager，请确保运行环境已安装并运行 ModemManager 服务。
- 请确保有足够的权限访问 DBus 系统总线。
