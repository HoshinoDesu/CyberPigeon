# CyberPigeon Web

Next.js 静态导出的管理控制台，构建产物会同步到 `../internal/server/web`，由 Go 二进制 `embed` 托管。

## 开发

```bash
npm install
npm run dev
```

后端 API 默认在同源 `/api`、`/ws`。本地开发可在 `next.config.ts` 增加 `rewrites` 代理到 Go 服务。

## 构建并同步到 Go embed 目录

```bash
npm run export:go
```

等价于：

```bash
npm run build
node ./scripts/sync-to-go.mjs
```

然后在仓库根目录：

```bash
go build -o CyberPigeon
```
