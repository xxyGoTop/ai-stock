# AI Stock

面向个人投研的 A 股分析平台：**行情 + 智能选股 + 个股每日笔记 + AI 多模型解读 + 模拟交易**。只做模拟盘，不接真实券商。

Web 是 React，业务 API 是 Go，量化计算和 Agent 在 Python。Android 端目前是占位。设计基线见 [docs/AI_Stock_Platform_V1_技术设计文档.md](docs/AI_Stock_Platform_V1_技术设计文档.md)。

仓库：https://github.com/xxyGoTop/ai-stock

## 现在能做什么

- **行情**：搜索股票、指数快照、个股详情、日 K 与均线 / MACD / RSI
- **选股**：五套算法扫描全市场（结果会缓存，切走 Tab 不会丢）
- **热点**：东财重要快讯 + 热门板块
- **每日笔记**：同一只股票同一天只留一份 pick 风格分析（算法命中、指标研判、资金、机构/游资、交易计划）
- **AI 分析**：按 Profile 调大模型；空结论或额度用尽会自动换下一个，最终兜底量化规则
- **模拟交易**：100 万初始资金、限价成交、T+1、手续费 / 印花税、今日盈亏、重置
- **自选股**：最多 40 只，个股页一键加减
- **模型配置**：设置页可切换分析 Profile，查看模型是否就绪、额度是否用尽
- **Agent Prompt**：提示词独立放在 `services/ai-python/prompts/`，一个目录一个 Agent，后续加分析员不用改调度代码

## 不会做什么

- 真实下单、跟单、策略商城
- 复杂机器学习预测
- 高频或 Level-2
- 把 API Key 写进数据库（只读环境变量）

结论仅供研究，不是投资建议。

## 架构

```text
浏览器 :5273
    │
    ▼
Go API :18080          行情聚合 / 模拟账户 / 自选 / 转调 Python
    │
    ├── 东财（搜索、快照、板块、快讯）
    ├── 腾讯 / 新浪（K 线兜底）
    └── Python :8090
            ├── 五套算法 Registry + 选股
            ├── 每日笔记 / 指标研判
            ├── LLM Router（单模型 / 回退 / 综合）
            └── prompts/  各 Agent 提示词
```

原则：**Quant 先算数，LLM 只解释**。选股公式不写进 Prompt。

## 五套选股算法

| 编码 | 中文 | 在看什么 |
|---|---|---|
| `year_high` | 年新高 | 率先创 250 日新高 |
| `deep_rebound` | 深调回升 | 大跌后止跌回升 |
| `forward_train` | 火车轨 | 连续阳线对齐推进 |
| `daily_observe` | 每日观察 | 当日观察池强度 |
| `ma5_align` | 五日线 | 五日线向上且多头共振 |

算法在 `services/ai-python/quant/algorithms/`，通过 Registry 注册，后续加新算法按现有文件抄一份即可。

## 数据源

| 能力 | 主源 | 兜底 |
|---|---|---|
| 搜索 / 快照 / 板块 | 东方财富 | 腾讯 |
| 日 K | 东财 / 腾讯新接口 / 新浪，取最新 | 实时 bar 补当天 |
| 热点快讯 | 东财重要快讯 | — |
| 公告 | 巨潮（后续） | — |

公开行情请控制频率。密钥不要提交。

## 模型与回退

配置在 `services/ai-python/config/llm.json`。密钥只写本地 `.env`（已 gitignore），模板见 `.env.example`。

| 环境变量 | 用途 |
|---|---|
| `LLM_AIHUBMIX_KEY` | AIHubMix，也认 `AIHUBMIX_API_KEY` |
| `LLM_DEEPSEEK_KEY` | DeepSeek |
| `LLM_QWEN_KEY` | 通义千问兼容接口 |

不配密钥也能用内置 `quant-rules`。

默认回退链：

```text
agents-a1-free → intern-s2-free → deepseek-chat → qwen-plus → quant-rules
```

某个模型返回空 `{}`、429、额度不足时跳过约 6 小时，再试下一个。设置页三个 Profile：

- **快速**：只走量化规则
- **AIHubMix 回退**：上面这条链
- **多模型综合**：几家并行再合成

## Agent Prompt

提示词不再写在 Python 字符串里。目录约定：

```text
services/ai-python/prompts/
  loader.py                 扫描目录、渲染 {{变量}}
  stock_analyst/            个股分析员（已接入）
  trading_planner/          交易计划员（预留）
  screening_nl/             选股理解员（预留）
  news_digest/              新闻摘要员（预留）
  ensemble_judge/           综合裁判（预留）
```

每个 Agent 三个文件：`agent.json`、`system.md`、`user.md`。用户模板可用 `{{stock.name}}`、`{{indicators.ma5}}`、`{{algorithm_lines}}`。

接到一次分析时，在 Profile 上写 `agentCode`。列表接口：`GET /api/v1/agents`。更细的说明见 [services/ai-python/prompts/README.md](services/ai-python/prompts/README.md)。

后续加 Agent：复制一个目录 → 改提示词 → 在 `llm.json` 里挂上 `agentCode`。

## 模拟交易规则

- 初始资金 1,000,000
- 限价，能成交才记持仓
- T+1：当天买的不能当天卖
- 佣金万一 2.5，卖出另收千一印花税
- 账户存在 `services/api-go/data/paper.json`（不进 Git）
- 支持今日盈亏和一键重置

## 本地启动

需要 Node 20、Go 1.18+、Python 3.10+。访问 GitHub 不稳时走本机 Clash SOCKS：

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
| Web | http://localhost:5273 |
| Go API | http://localhost:18080/health |
| Python | http://localhost:8090/health |

## 仓库结构

```text
apps/web                 React + Vite + ECharts
apps/mobile              React Native 占位
packages/types           共享类型
packages/api-client      Web 调 Go 的封装
packages/business        业务工具
services/api-go          行情、模拟账户、自选、代理 Python
services/ai-python
  quant/algorithms       五套算法
  agent                  个股分析、每日笔记
  llm                    模型配置、Router、额度
  prompts                Agent 提示词（独立目录）
  config/llm.json        Provider / Model / Profile
docs/                    V1 技术设计文档
```

## 常用接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/stocks/search?q=` | 搜索 |
| GET | `/api/v1/kline?symbol=&limit=` | 日 K |
| POST | `/api/v1/screening` | 选股 |
| GET | `/api/v1/hot` | 热点 |
| GET | `/api/v1/ai/daily-note?symbol=` | 每日笔记 |
| POST | `/api/v1/ai/analyze` | AI 分析 |
| GET | `/api/v1/agents` | Agent 目录 |
| GET | `/api/v1/analysis-profiles` | 分析 Profile |
| GET/POST | `/api/v1/paper/account` `/orders` `/reset` | 模拟盘 |
| GET/POST/DELETE | `/api/v1/watchlist` | 自选 |

## 开发约定

- 密钥只放本地 `.env`，不要写进 `.env.example` 或提交
- 选股公式和模型名不要散落在 Prompt 里
- 空模型输出必须回退，不能当成成功结论
- 页面改完后用浏览器走一遍主流程再收工
