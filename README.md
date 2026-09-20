# AI Stock

AI 股票投研 + 智能选股 + 模拟交易平台（V1）。Web + Android，后端 Go，量化 / Agent 用 Python。

当前进度：**Sprint 1 行情基础设施** — 搜索、实时行情、K 线、均线 / MACD / KDJ / RSI，以及 K 线新鲜度告警。

## 数据源

| 能力 | 主源 | 兜底 |
|---|---|---|
| 搜索 / 快照 / 板块 | 东方财富 | 腾讯 |
| 日 K | 东财 / 腾讯 / 新浪 探测取最新 | 实时 bar 补当天 |
| 公告（后续） | 巨潮 | — |

密钥不要提交。公开行情接口请控制频率。

## 本地启动

需要 Node 20、Go 1.18+。本机若访问 GitHub 不稳定，走已有代理（Clash SOCKS `127.0.0.1:1080`）：

```powershell
$env:ALL_PROXY="socks5://127.0.0.1:1080"
$env:HTTPS_PROXY="socks5://127.0.0.1:1080"
```

Go 模块建议：

```powershell
$env:GOPROXY="https://goproxy.cn,direct"
```

```powershell
# 后端
cd services/api-go
go run ./cmd/server

# 前端（另开终端）
cd ../..
npm install --registry=https://registry.npmmirror.com
npm run dev:web
```

- API: http://localhost:18080/health
- Web: http://localhost:5173

## 仓库结构

```text
apps/web            React + Vite
apps/mobile         React Native 占位
packages/           共享类型 / API Client
services/api-go     业务 API + Provider
services/ai-python  Agent / 算法（占位）
docs/               设计文档
```

设计说明见 `docs/AI_Stock_Platform_V1_技术设计文档.md`。
