# AI Stock

AI 股票投研 + 智能选股 + 模拟交易平台（V1）。Web + Android，后端 Go，量化 / Agent 用 Python。

当前进度：**行情 / 选股 / 热点 / 个股每日算法笔记 / 模拟交易**。个股详情先出当日 pick 笔记（同一天只存一份），再出 K 线。

## 数据源

| 能力 | 主源 | 兜底 |
|---|---|---|
| 搜索 / 快照 / 板块 | 东方财富 | 腾讯 |
| 日 K | 东财 / 腾讯 / 新浪 探测取最新 | 实时 bar 补当天 |
| 热点快讯 | 东财重要快讯 | — |
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
# 三个终端
npm run dev:api
npm run dev:python
npm run dev:web
```

Python 依赖：`pip install -r services/ai-python/requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple`

- API: http://localhost:18080/health
- Python: http://localhost:8090/health
- Web: http://localhost:5273

可选大模型密钥（不配也能用内置 `quant-rules`）：

```
LLM_AIHUBMIX_KEY   # 或 AIHUBMIX_API_KEY，https://aihubmix.com
LLM_DEEPSEEK_KEY
LLM_QWEN_KEY
```

AIHubMix 默认回退链：`agents-a1-free` → `intern-s2-free` → DeepSeek / Qwen → 量化规则。某个模型额度用尽（429/限流）后会自动跳过约 6 小时。

## 仓库结构

```text
apps/web            React + Vite
apps/mobile         React Native 占位
packages/           共享类型 / API Client
services/api-go     业务 API + Provider
services/ai-python  五算法 Registry + 选股 + LLM Router
docs/               设计文档
```

设计说明见 `docs/AI_Stock_Platform_V1_技术设计文档.md`。
