# CyberPigeon API 用例

本文档整理了 Web 控制台和自动化脚本常用的 API 调用示例。

说明：文中的 IMEI、号码、ICCID、验证码、Token 和 Chat ID 均为脱敏后的示例占位值，请替换成你自己的实际参数。

## 基本说明

- 默认监听地址示例：`http://127.0.0.1:8080`
- 当启用管理密码后，除认证状态接口外，其余 `/api/*` 都需要先登录。
- 登录成功后，服务端通过 Cookie `cyberpigeon_session` 维持会话。
- 下方示例统一使用 `curl` 的 `cookies.txt` 保存和复用会话。

## 1. 查询认证状态

```bash
curl http://127.0.0.1:8080/api/auth/status
```

示例返回：

```json
{
  "auth_enabled": true,
  "requires_setup": false,
  "authenticated": false,
  "message": "登录已失效，请重新登录"
}
```

## 2. 首次设置管理密码

仅适用于旧版本升级后首次进入，需要完成初始化的场景。

```bash
curl -X POST http://127.0.0.1:8080/api/auth/setup \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{"password":"abc123"}'
```

说明：

- 管理密码至少 6 位。
- 仅允许字母和数字。
- 设置成功后会自动签发登录会话 Cookie。

## 3. 登录

```bash
curl -X POST http://127.0.0.1:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{"password":"abc123"}'
```

成功返回：

```json
{
  "success": true
}
```

## 4. 退出登录

```bash
curl -X POST http://127.0.0.1:8080/api/auth/logout \
  -b cookies.txt
```

## 5. 修改管理密码

```bash
curl -X POST http://127.0.0.1:8080/api/auth/change-password \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -c cookies.txt \
  -d '{"current_password":"abc123","new_password":"abc456"}'
```

说明：

- 修改成功后，其他浏览器或客户端的旧会话会全部失效。
- 当前请求会收到新的会话 Cookie。

## 6. 获取设备列表

```bash
curl http://127.0.0.1:8080/api/modems \
  -b cookies.txt
```

示例返回：

```json
[
  {
    "imei": "IMEI_EXAMPLE_001",
    "model": "Modem Example",
    "manufacturer": "Vendor Example",
    "number": "+8613800000000",
    "signal_quality": 20,
    "operator_name": "Carrier Example",
    "iccid": "ICCID_EXAMPLE_001"
  }
]
```

## 7. 获取短信列表

```bash
curl "http://127.0.0.1:8080/api/messages?limit=50&offset=0" \
  -b cookies.txt
```

示例返回：

```json
{
  "items": [
    {
      "id": "msg_example_001",
      "modem": "IMEI_EXAMPLE_001",
      "number": "ServiceExample",
      "text": "Your verification code is 123456. Valid within 5 minutes.",
      "timestamp": "2026-03-11T01:14:13+08:00",
      "saved": "2026-03-11T01:14:15+08:00"
    }
  ],
  "total": 42
}
```

参数说明：

- `limit`：每页条数，默认 50，最大 200。
- `offset`：分页偏移量，从 0 开始。

## 8. 删除短信

```bash
curl -X POST http://127.0.0.1:8080/api/messages/delete \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"id":"msg_example_001"}'
```

## 9. 获取系统设置

```bash
curl http://127.0.0.1:8080/api/settings \
  -b cookies.txt
```

示例返回：

```json
{
  "device_name": "Home Gateway",
  "device_name_in_title": true,
  "device_name_in_body": true
}
```

## 10. 保存系统设置

```bash
curl -X POST http://127.0.0.1:8080/api/settings/save \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"device_name":"Home Gateway","device_name_in_title":true,"device_name_in_body":false}'
```

## 11. 执行 USSD

```bash
curl -X POST http://127.0.0.1:8080/api/ussd \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"imei":"IMEI_EXAMPLE_001","code":"*100#"}'
```

成功返回：

```json
{
  "reply": "余额查询结果"
}
```

说明：

- `code` 仅允许数字、`*` 和 `#`。
- 当设备不存在时会返回 404。

## 12. 获取推送通道配置

```bash
curl http://127.0.0.1:8080/api/channels \
  -b cookies.txt
```

说明：

- 返回值是通道配置数组。
- 服务端会自动补齐未配置的默认通道模板。

## 13. 保存推送通道配置

```bash
curl -X POST http://127.0.0.1:8080/api/channels/save \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '[{"type":"telegram","enabled":true,"bot_token":"TELEGRAM_BOT_TOKEN","chat_id":"TELEGRAM_CHAT_ID","request_timeout_sec":10}]'
```

## 14. 测试推送通道

```bash
curl -X POST http://127.0.0.1:8080/api/channels/test \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '[{"type":"telegram","enabled":true,"bot_token":"TELEGRAM_BOT_TOKEN","chat_id":"TELEGRAM_CHAT_ID","request_timeout_sec":10}]'
```

成功返回：

```json
{
  "success": true,
  "message": "测试消息已发送"
}
```

## 常见错误

```text
401 Unauthorized
```

表示未登录、登录已失效，或未携带有效 Cookie。

```text
428 Precondition Required
```

表示当前实例尚未设置管理密码，需要先调用 `/api/auth/setup`。

```text
429 Too Many Requests
```

表示管理密码连续输错次数过多，需要等待一段时间后再重试。