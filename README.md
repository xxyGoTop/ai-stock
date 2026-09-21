# AI Stock

面向个人投研的 A 股分析平台，当前主路径是 **V2 投研伙伴（Conversation First）**：对话驱动研究，右侧 Research Workspace 联动；底层仍是 **行情 + 五算法选股 + 每日笔记 + AI 解读 + 模拟交易**。只做模拟盘，不接真实券商。

Web 是 React，业务 API 是 Go，量化计算与 LLM Agent 在 Python。Android 端目前是占位。

- V1 页面式设计：[docs/AI_Stock_Platform_V1_技术设计文档.md](docs/AI_Stock_Platform_V1_技术设计文档.md)
- V2 伙伴 / Agent 设计：[docs/AI_投研伙伴_V2_AI_Agent_交互与技术设计文档.md](docs/AI_投研伙伴_V2_AI_Agent_交互与技术设计文档.md)

仓库：https://github.com/xxyGoTop/ai-stock

## 现在能做什么

### 投研伙伴（首页 `/`）

- **进房简报**：指数、北向资金、热门/主流板块、热点主题、盘前/盘中/尾盘/收盘节奏卡
- **多会话**：左侧会话列表（sessionStorage），可新建 / 切换 / 删除；切到个股详情再回对话不丢上下文
- **Agent Run（SSE 真流式）**：`POST /api/v1/ai/companion/chat/stream`  
  事件：`message.start` → `research.plan` → `tool.*` → `block` → `message.delta` → `message.end`  
  前端展示状态条 + 研究计划 + 工具行；可**取消**或**打断并发送**开启新一轮
- **结构化块**：指数、北向、板块、选股结果（分页）、个股卡、每日笔记、AI 解读、自选、**异动**、建议 chips
- **研究工作台**：市场总览 / 个股行情 · K 线 · 新闻公告 · 分析 · 模拟下单，与对话意图联动
- **自选异动**：扫自选涨跌幅 / 量比 / 换手；简报展示；快捷「异动」；页内约 3 分钟主动提醒（指纹去重）
- **每日推荐归档**：盘前/盘中等推荐写入 `data/daily-picks/`，设置页可按日翻看（含 30 只分页）

### 经典能力（仍可用）

- **行情**：搜索、指数、个股详情、日 K、均线 / MACD / RSI
- **选股**：五套算法全市场扫描（结果缓存）
- **热点**：东财重要快讯 + 热门板块
- **每日笔记 / AI 分析**：同日一份 pick 风格笔记；Profile 调模型，失败自动回退到量化规则
- **模拟交易**：100 万初始、限价、T+1、佣金/印花税、今日盈亏、重置
- **自选**：最多 40 只；对话列表和顶栏可删除、一键清空
- **设置**：分析 Profile、模型状态、每日推荐记录
- **Agent Prompt**：`services/ai-python/prompts/` 一目录一 Agent

导航以图标为主（对话 / 模拟 / 设置）；行情、热点、选股已折叠进对话快捷入口。

## 不会做什么

- 真实下单、跟单、策略商城
- 复杂机器学习预测 / 高频 / Level-2
- 把 API Key 写进数据库（只读环境变量）

结论仅供研究，不是投资建议。

## 架构（当前）

```text
浏览器 :5273  (Vite → 代理 /api → Go，SSE 关缓冲)
    │
    ▼
Go API :18080
    ├── companion/     进房简报、Chat、Agent Run SSE、自选异动扫描
    ├── watchlist/     自选 JSON
    ├── dailypicks/    每日推荐归档 JSON
    ├── paper/         模拟账户
    ├── provider/      东财 / 腾讯 / 新浪行情聚合 + 北向
    └── 转调 Python :8090
            ├── quant/algorithms   五算法 Registry + 选股
            ├── agent              每日笔记 / 个股分析
            ├── llm                Router + 回退 / 综合
            └── prompts/           各 Agent 提示词
```

原则：**Quant 先算数，LLM 只解释**。选股公式不写进 Prompt。

### 前端关键路径

| 路径 | 说明 |
|---|---|
| `apps/web/src/pages/Chat.tsx` | 伙伴主界面：会话 + SSE Agent Run + 异动轮询 |
| `apps/web/src/components/ResearchProgress.tsx` | Agent 状态条 + 研究计划 / 工具 |
| `apps/web/src/components/ChatBlocks.tsx` | 结构化消息块（含 `anomaly`） |
| `apps/web/src/lib/chatSession.ts` | 多会话 sessionStorage |
| `packages/api-client` | `companionChatStream`、`getWatchAnomalies` 等 |
| `packages/types` | `AgentStreamEvent`、`WatchAnomaly` 等 |

### 后端关键路径

| 路径 | 说明 |
|---|---|
| `services/api-go/internal/companion/stream.go` | Agent Run SSE 推送 |
| `services/api-go/internal/companion/anomaly.go` | 自选异动规则与扫描 |
| `services/api-go/internal/companion/briefing.go` / `chat.go` | 简报与意图路由 |
| `services/api-go/internal/dailypicks/` | 每日推荐持久化 |
| `services/api-go/internal/httpserver/server.go` | 路由注册 |

## 今晚改动摘要（2026-09-20，方便换机续作）

已落地、可接着改：

1. **Conversation First 主界面**：双栏对话 + 工作台；会话侧栏；中文标题与图标导航
2. **SSE Agent Run**：真流式相互推送（服务端事件 + 客户端 Abort 取消 / 打断重发）
3. **研究进度 UI**：状态机文案、计划步骤、工具行；完成后写入历史消息的 `progress` / `tools`
4. **每日 30 推荐 + 设置页归档**：`GET /api/v1/daily-picks`
5. **自选异动 MVP**：`GET /api/v1/watchlist/anomalies`；意图 `watch_anomaly`；简报块 + 页内主动提醒
6. **Vite SSE 代理**：`apps/web/vite.config.ts` 对 `text/event-stream` 关缓冲

建议明天优先：

- [ ] 异动规则可配置（阈值 / 阈值，设置页）
- [ ] Notification Agent 后台定时（不仅页内 poll）
- [ ] 服务端会话 / research_events 持久化（现在是 sessionStorage）
- [ ] 更丰富 Workspace 块（对比、风险卡等，见 V2 文档 Phase 2–4）

本地数据（不进 Git）：`services/api-go/data/watchlist.json`、`paper.json`、`daily-picks/`。换机后自选与模拟账户需重新加或自行拷贝。

## 五套选股算法

| 编码 | 中文 | 在看什么 |
|---|---|---|
| `year_high` | 年新高 | 率先创 250 日新高 |
| `deep_rebound` | 深调回升 | 大跌后止跌回升 |
| `forward_train` | 火车轨 | 连续阳线对齐推进 |
| `daily_observe` | 每日观察 | 当日观察池强度 |
| `ma5_align` | 五日线 | 五日线向上且多头共振 |

算法在 `services/ai-python/quant/algorithms/`，通过 Registry 注册。

## 数据源

| 能力 | 主源 | 兜底 |
|---|---|---|
| 搜索 / 快照 / 板块 | 东方财富 | 腾讯 |
| 日 K | 东财 / 腾讯 / 新浪 | 实时 bar 补当天 |
| 热点快讯 | 东财重要快讯 | — |
| 北向资金 | 东财 | — |

公开行情请控制频率。密钥不要提交。

## 模型与回退

配置在 `services/ai-python/config/llm.json`。密钥只写本地 `.env`（已 gitignore），模板见 `.env.example`。

| 环境变量 | 用途 |
|---|---|
| `LLM_AIHUBMIX_KEY` | AIHubMix，也认 `AIHUBMIX_API_KEY` |
| `LLM_DEEPSEEK_KEY` | DeepSeek |
| `LLM_QWEN_KEY` | 通义千问兼容接口 |
| `ARK_API_KEY` | 火山方舟（官方变量名，优先） |
| `LLM_ARK_KEY` | 火山方舟备用 |
| `LLM_ARK_MODEL` | 可选，覆盖豆包 Seed 的 Model ID / 推理接入点 `ep-xxx` |

不配密钥也能用内置 `quant-rules`。对话默认走火山方舟 [Chat API](https://www.volcengine.com/docs/82379/1112500)（`https://ark.cn-beijing.volces.com/api/v3`）：

```text
doubao-seed-2-1-pro → deepseek-v4-flash → agents-a1-free → intern-s2-free → deepseek-chat → qwen-plus → quant-rules
```

空 `{}`、429、额度不足时跳过约 6 小时再试下一个。对话输入框可切模型。设置页 Profile：快速 / AIHubMix 回退 / 火山方舟 / 多模型综合。

## Agent Prompt

```text
services/ai-python/prompts/
  loader.py
  stock_analyst/     个股分析员（已接入）
  trading_planner/   交易计划员（预留）
  screening_nl/      选股理解员（预留）
  news_digest/       新闻摘要员（预留）
  ensemble_judge/    综合裁判（预留）
```

每个 Agent：`agent.json` + `system.md` + `user.md`。说明见 [services/ai-python/prompts/README.md](services/ai-python/prompts/README.md)。

## 模拟交易规则

- 初始资金 1,000,000；限价成交；T+1
- 佣金万一 2.5，卖出另收千一印花税
- 账户：`services/api-go/data/paper.json`（不进 Git）

## 本地启动

需要 Node 20、Go 1.18+、Python 3.10+。访问 GitHub / 模块不稳时：

```powershell
$env:ALL_PROXY="socks5://127.0.0.1:1080"
$env:HTTPS_PROXY="socks5://127.0.0.1:1080"
$env:GOPROXY="https://goproxy.cn,direct"
```

```powershell
copy .env.example .env
# 按需填 LLM_* 密钥

npm install
pip install -r services/ai-python/requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple
```

三个终端：

```powershell
npm run dev:api
npm run dev:python
npm run dev:web
```

| 服务 | 地址 |
|---|---|
| Web | http://127.0.0.1:5273 |
| Go API | http://127.0.0.1:18080/health |
| Python | http://127.0.0.1:8090/health |

快速自测：

```powershell
# 异动
curl.exe -s http://127.0.0.1:18080/api/v1/watchlist/anomalies

# Agent Run SSE（UTF-8 body 建议写临时文件再 --data-binary）
curl.exe -sN -X POST http://127.0.0.1:18080/api/v1/ai/companion/chat/stream -H "Content-Type: application/json" --data-binary "@body.json"
```

## 仓库结构

```text
apps/web                 React + Vite + ECharts（伙伴主界面）
apps/mobile              React Native 占位
packages/types           共享类型（含 AgentStreamEvent / WatchAnomaly）
packages/api-client      Web 调 Go（含 companionChatStream）
packages/business        业务工具
services/api-go
  internal/companion     简报 / Chat / SSE / 异动
  internal/dailypicks    每日推荐归档
  internal/watchlist     自选
  internal/paper         模拟盘
  internal/provider      行情聚合
services/ai-python
  quant/algorithms       五套算法
  agent / llm / prompts
docs/                    V1 + V2 设计文档
```

## 常用接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/ai/companion/briefing` | 进房简报（含异动块） |
| POST | `/api/v1/ai/companion/chat` | 同步对话 |
| POST | `/api/v1/ai/companion/chat/stream` | **Agent Run SSE** |
| GET | `/api/v1/watchlist/anomalies` | 自选异动扫描 |
| GET | `/api/v1/daily-picks` | 每日推荐列表 / 按日查询 |
| GET | `/api/v1/market/northbound` | 北向资金 |
| GET | `/api/v1/stocks/search?q=` | 搜索 |
| GET | `/api/v1/kline?symbol=&limit=` | 日 K |
| POST | `/api/v1/screening` | 选股 |
| GET | `/api/v1/hot` | 热点 |
| GET | `/api/v1/ai/daily-note?symbol=` | 每日笔记 |
| POST | `/api/v1/ai/analyze` | AI 分析 |
| GET | `/api/v1/agents` | Agent 目录 |
| GET/POST/DELETE | `/api/v1/watchlist` | 自选 |
| GET/POST | `/api/v1/paper/account` `/orders` `/reset` | 模拟盘 |

## 开发约定

- 密钥只放本地 `.env`，不要提交
- 选股公式和模型名不要散落在 Prompt 里
- 空模型输出必须回退
- SSE 相关改动要验证 Vite 代理下是否边推边显（勿整包缓冲）
- 换机开发：拉代码 → `npm install` → 起三服务 → 按「今晚改动摘要」续作
