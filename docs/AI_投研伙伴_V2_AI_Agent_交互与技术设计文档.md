# AI 投研伙伴 V2
## AI Agent + Conversation First + Research Workspace 技术设计文档

> 版本：V2.0  
> 定位：AI 投研伙伴 / AI 股票研究员  
> 核心交互：Conversation First + Agent 工作流 + Research Workspace  
> 客户端：React Web + React Native Android  
> 后端：Go + Python  
> 交易模式：模拟交易，不直接执行真实交易

---

# 1. 产品定位

本产品不再以传统“股票行情 App”为核心，而是：

> **一个可以持续对话、主动研究、理解上下文、追踪用户关注方向的 AI 投研伙伴。**

用户不需要先理解产品功能，只需要提出目标：

- 帮我看看今天市场怎么样
- 帮我分析一下贵州茅台
- 最近机器人板块有什么值得研究？
- 帮我找一些趋势比较强的股票
- 看看我这次模拟交易为什么亏了

AI 自动决定需要获取哪些数据、调用哪些工具、进行哪些分析，并将结果持续呈现在 Research Workspace 中。

核心闭环：

```text
用户提问
  ↓
Conversation
  ↓
AI Agent
  ↓
Research Plan
  ↓
Tool Calling
  ↓
行情 / 新闻 / Quant / 财务
  ↓
结构化分析
  ↓
Research Workspace
  ↓
继续追问
  ↓
模拟交易
  ↓
复盘 / 持续跟踪
```

---

# 2. 核心设计原则

## 2.1 Conversation First

用户首先面对的是 AI 对话，而不是菜单、指标或筛选器。

## 2.2 Agent Second

AI 不只是回答，而是：

```text
理解 → 规划 → 调用工具 → 获取数据 → 分析 → 解释 → 继续研究
```

## 2.3 Data Third

行情、K线、技术指标、新闻、资金、行业、财务等数据作为 AI 的能力，而不是要求用户自己操作。

## 2.4 Structured UI

AI 输出不能只有 Markdown，必须输出结构化 UI Block：

```text
文字
股票卡片
K线
指标
新闻
对比表
风险
交易计划
```

## 2.5 Human in the Loop

AI 可以研究、分析和生成模拟交易计划，但不直接执行真实交易。

---

# 3. 产品信息架构

Web 推荐一级导航：

```text
AI
研究
自选
模拟
我的
```

但默认首页是 AI。

```text
AI 投研
├── 会话
│   ├── 今日市场
│   ├── 个股研究
│   ├── 行业研究
│   ├── 热点研究
│   └── 选股研究
├── Research Workspace
├── 模拟交易
└── 用户偏好 / Memory
```

---

# 4. Web 核心布局

采用“左边聊、右边研究”的模式：

```text
┌──────────────┬──────────────────────────────────────┐
│ Conversation │          Research Workspace          │
│              │                                      │
│ 今日市场      │ 当前研究对象                         │
│ 贵州茅台      │ 贵州茅台                             │
│ AI板块        │ ┌────────────────────────────────┐ │
│ 机器人选股    │ │ K线                            │ │
│ 新能源        │ │                                │ │
│              │ └────────────────────────────────┘ │
│ + 新建会话    │                                      │
│              │ 技术 | 资金 | 新闻 | 行业             │
│              │                                      │
│              │ AI 分析                             │
└──────────────┴──────────────────────────────────────┘
```

核心理念：

> 左侧负责 Conversation，右侧负责 Research。

---

# 5. Conversation

## 5.1 首页

```text
AI 投研助手

今天想研究什么？

[ 帮我看看今天市场有什么热点 ]

推荐：

· 今天市场怎么样？
· 帮我分析一只股票
· 找最近强势行业
· 看看我的自选股
· 帮我做一次模拟交易复盘
```

## 5.2 会话列表

```text
今天
  今天市场怎么样
  贵州茅台分析
  机器人板块研究

昨天
  AI算力选股
  新能源行业分析
```

## 5.3 连续追问

用户：

> 帮我分析一下贵州茅台

然后：

> MACD 为什么改善？

再：

> 那和比亚迪比呢？

再：

> 如果模拟交易怎么做？

整个过程保持同一个 Research Context，不重新开始。

---

# 6. Agent 工作过程

例如用户：

> 帮我看看贵州茅台最近怎么样。

AI 展示：

```text
正在研究 贵州茅台

✓ 获取实时行情
✓ 获取近期 K 线
✓ 计算技术指标
✓ 分析所属行业
✓ 检索近期新闻
● 正在整理结论
```

然后逐步出现：

```text
贵州茅台
¥1500.20  +1.52%

技术面
趋势：偏强
MACD：改善
RSI：54

近期新闻
...

风险
...
```

这类交互应接近 ChatGPT / Codex 的 Agent 工作流，而不是传统 loading。

---

# 7. Agent 状态

```text
idle
thinking
planning
tool_calling
analyzing
rendering
waiting_user
completed
error
```

前端映射：

```text
thinking      → 正在理解你的问题
planning      → 正在制定研究方案
tool_calling  → 正在获取市场数据
analyzing     → 正在分析
rendering     → 正在整理结果
waiting_user  → 等待你继续研究
```

---

# 8. Research Plan

复杂任务生成 Research Plan：

```json
{
  "goal": "分析贵州茅台",
  "steps": [
    {"id": "quote", "title": "获取行情", "status": "completed"},
    {"id": "technical", "title": "分析技术指标", "status": "completed"},
    {"id": "news", "title": "检索近期新闻", "status": "completed"},
    {"id": "risk", "title": "评估风险", "status": "running"}
  ]
}
```

UI：

```text
研究进度

✓ 行情
✓ 技术面
✓ 新闻
● 风险分析
○ 模拟交易计划
```

---

# 9. Agent Tool Layer

AI Agent 不直接访问数据库。

```text
Research Agent
├── get_quote
├── get_kline
├── get_indicators
├── get_capital_flow
├── get_sector
├── get_news
├── get_financial_data
├── screen_stocks
├── compare_stocks
├── analyze_risk
└── generate_paper_trade_plan
```

每个 Tool 都必须有：

- Input Schema
- Output Schema
- 权限
- 超时
- 错误处理
- 数据来源

---

# 10. Agent 分层

不要设计一个超级 Agent。

```text
                    Research Agent
                          │
          ┌───────────────┼────────────────┐
          ▼               ▼                ▼
     Market Agent    News Agent      Quant Agent
          │               │                │
          └───────────────┼────────────────┘
                          ▼
                    Strategy Agent
```

## Research Agent

负责：

- 理解问题
- 制定研究计划
- 协调其他 Agent
- 总结结果

## Market Agent

负责：

- 行情
- K线
- 技术指标

## News Agent

负责：

- 新闻
- 公告
- 热点
- 事件

## Quant Agent

负责：

- 指标
- 因子
- 选股
- 评分

## Strategy Agent

负责：

- 交易假设
- 模拟交易计划
- 交易复盘

---

# 11. AI 与 Quant 的边界

必须保持：

```text
LLM
理解 / 规划 / 推理 / 解释

Quant Engine
MA / MACD / RSI / KDJ
趋势 / 波动率 / 因子 / 评分 / 筛选

Go Backend
用户 / 权限 / 会话 / 业务 / 模拟交易
```

不要让 LLM 自己计算大量行情指标。

---

# 12. Research Workspace

Workspace 不是普通股票详情页，而是：

> **AI 当前正在研究的对象。**

统一支持：

```text
stock
sector
topic
market
portfolio
```

例如研究股票：

```text
┌───────────────────────────────────┐
│ 贵州茅台 600519                   │
│ ¥1500.20 +1.52%                  │
│                                   │
│ [概览] [技术] [资金] [新闻] [行业] │
│                                   │
│ K线                                │
│                                   │
│ 技术面                             │
│ MA20 ↑  MACD →  RSI 54            │
│                                   │
│ AI 观点                            │
│ ...                               │
└───────────────────────────────────┘
```

---

# 13. Conversation 与 Workspace 联动

用户：

> MACD 为什么改善？

AI：

> 主要是最近几个交易日 DIF 与 DEA 的差值开始收窄，我给你展开一下。

Workspace 自动定位：

```text
技术指标 → MACD
```

用户：

> 那和比亚迪比呢？

Workspace 自动切换：

```text
贵州茅台 VS 比亚迪
```

---

# 14. 股票对比

支持：

```text
A vs B
A vs B vs C
```

维度：

```text
价格趋势
技术指标
行业
资金
新闻
财务
风险
```

示例：

```text
             茅台       比亚迪

趋势          ↑          ↑↑
MA20          ↑          ↑
MACD          →          ↑
RSI           54         61
行业强度      72         81
新闻热度      68         74
```

AI 负责解释差异，而不是只展示数字。

---

# 15. AI Memory

分三类：

## Conversation Memory

当前会话上下文。

## Research Memory

用户研究过：

```text
贵州茅台
比亚迪
机器人
AI算力
新能源
```

## User Preference

仅保存用户明确表达或设置的偏好，例如：

```text
关注方向：
AI
机器人
新能源
```

不应因为一次对话就永久推断用户画像。

---

# 16. Memory 表

PostgreSQL：

```text
user_memories

id
user_id
type
key
value
source
confidence
created_at
updated_at
```

type：

```text
preference
interest
research_context
```

---

# 17. 长上下文控制

不能把全部历史对话发送给 LLM。

采用：

```text
当前消息
+
最近 N 轮
+
Conversation Summary
+
Research Context
+
Relevant Memory
```

长会话自动生成 Summary：

```json
{
  "topic": "贵州茅台研究",
  "importantFacts": [
    "当前正在研究技术趋势"
  ],
  "openQuestions": [
    "需要进一步分析行业"
  ]
}
```

---

# 18. AI UI Block 协议

AI 输出：

```json
{
  "type": "research_response",
  "blocks": [
    {"type": "text", "content": "..."},
    {"type": "stock", "symbol": "600519"},
    {"type": "chart", "chartType": "kline"},
    {"type": "indicator", "name": "MACD"},
    {"type": "news", "items": []},
    {"type": "comparison", "items": []},
    {"type": "risk", "items": []},
    {"type": "paper_trade_plan", "data": {}}
  ]
}
```

这样 Web 与 Android 可以共享协议。

---

# 19. SSE 流式协议

```http
POST /api/v1/ai/chat
Accept: text/event-stream
```

事件：

```text
message.start
research.plan
tool.start
tool.result
analysis.start
block
message.delta
message.end
```

例如：

```text
event: research.plan
data: {...}

event: tool.start
data: {"tool":"get_quote"}

event: tool.result
data: {...}

event: block
data: {"type":"stock","symbol":"600519"}

event: message.delta
data: {"content":"从技术面来看..."}

event: message.end
data: {}
```

---

# 20. 陪伴式体验

每日进入：

```text
早上好。

我帮你看了一下今天的市场。

你的关注方向里：

AI：热度上升
机器人：保持活跃
新能源：暂时平稳

你的自选股中有 2 只出现明显异动。

要不要我先从这里开始？
```

用户：

> 好。

AI：

```text
我先看你的两个异动。

第一个是 XXX。

主要原因：
...
```

核心不是“推送一条数据”，而是：

> **推送一条已经完成初步研究的结论。**

---

# 21. 主动提醒

主动能力只基于：

- 用户授权
- 用户自选
- 用户关注方向
- 用户设置

例如：

```text
你的自选股 XXX 今天出现较明显异动。

我快速看了一下：

✓ 成交量
✓ 技术面
✓ 新闻
✓ 行业

目前值得关注的是……

[继续研究]
```

---

# 22. Notification Agent

后台：

```text
市场事件
 ↓
扫描用户自选
 ↓
扫描用户关注方向
 ↓
判断重要性
 ↓
生成摘要
 ↓
判断是否值得通知
 ↓
发送
```

输出：

```json
{
  "shouldNotify": true,
  "title": "你的自选股出现异动",
  "summary": "...",
  "action": "research_stock"
}
```

---

# 23. 自然语言选股

用户：

> 帮我找最近趋势比较强，而且所属行业也比较强的股票。

Agent：

```text
我会从几个维度筛选：

✓ 趋势
✓ 成交量
✓ 行业强度
✓ 资金
✓ 风险
```

然后：

```text
找到 18 个候选。

我进一步比较后，有 5 个值得继续研究。
```

结果：

```text
股票 A
技术 82
行业 88
资金 76

[研究] [加入自选]
```

---

# 24. 热点研究

用户：

> 最近有什么热点？

AI：

```text
我先从新闻、行业表现和资金变化
三个维度整理一下。
```

结果：

```text
AI算力
热度 92
相关股票 18
新闻 126

机器人
热度 87
相关股票 24
新闻 98

新能源
热度 74
相关股票 31
新闻 81
```

用户：

> 深入看看机器人。

直接进入：

```text
机器人 Research Workspace
```

---

# 25. 股票研究模板

根据问题动态展开，常见模块：

```text
当前状态
技术面
资金面
行业
新闻
基本面
风险
关注信号
模拟交易计划
```

不是每次都强制全部输出。

---

# 26. 模拟交易

模拟交易不再是孤立模块，而是：

```text
AI 研究
 ↓
交易假设
 ↓
交易计划
 ↓
用户确认
 ↓
模拟下单
 ↓
持仓
 ↓
市场变化
 ↓
AI 跟踪
 ↓
复盘
```

AI 不自动执行真实交易。

---

# 27. 模拟交易计划

结构：

```json
{
  "symbol": "600519",
  "direction": "watch",
  "entryRange": [1480, 1510],
  "stopLoss": 1430,
  "targets": [1580, 1650],
  "position": 0.2,
  "riskLevel": "medium",
  "invalidConditions": [
    "跌破MA20",
    "行业强度明显下降"
  ]
}
```

UI：

```text
模拟交易计划

关注区间：1480 - 1510
风险位置：1430
目标区域：1580 / 1650
模拟仓位：20%
风险：中等

[加入自选] [模拟买入]
```

所有计划都应明确：

> 仅用于研究与模拟交易，不构成投资建议。

---

# 28. 模拟交易复盘

用户：

> 帮我看看这次模拟交易为什么亏了。

AI 获取：

```text
订单
成交
持仓
当时行情
当时交易计划
之后市场变化
```

分析：

```text
计划入场：1480-1510
实际成交：1525

计划风险位置：1430
实际卖出：1418

主要偏差：
1. 成交价格高于计划区间
2. 实际执行偏离原交易计划
3. 行业强度随后下降
```

---

# 29. Go Backend

```text
services/api-go/
├── cmd/server/
├── internal/
│   ├── auth/
│   ├── user/
│   ├── conversation/
│   ├── memory/
│   ├── research/
│   ├── stock/
│   ├── market/
│   ├── news/
│   ├── watchlist/
│   ├── screening/
│   ├── notification/
│   ├── trading/
│   └── ai/
├── pkg/
│   ├── provider/
│   ├── response/
│   ├── middleware/
│   └── logger/
└── migrations/
```

---

# 30. Python AI / Quant

```text
services/ai-python/
├── agent/
│   ├── research_agent.py
│   ├── market_agent.py
│   ├── news_agent.py
│   ├── quant_agent.py
│   └── strategy_agent.py
├── tools/
│   ├── market.py
│   ├── technical.py
│   ├── capital.py
│   ├── news.py
│   ├── sector.py
│   ├── financial.py
│   ├── screening.py
│   └── risk.py
├── quant/
│   ├── indicators/
│   ├── factors/
│   ├── scoring/
│   └── screening/
└── main.py
```

---

# 31. 数据架构

```text
第三方金融数据
       │
       ▼
Provider Adapter
       │
       ├───────────────┐
       ▼               ▼
   Go Backend       Python Quant
       │               │
       ▼               ▼
     Redis       Indicator Engine
       │               │
       └───────┬───────┘
               ▼
          ClickHouse
               │
               ▼
            AI Agent
               │
               ▼
       Research Response
```

Provider 必须抽象：

```go
type MarketProvider interface {
    GetQuote(symbol string) (*Quote, error)
    GetKline(symbol string, period string) ([]Kline, error)
    GetCapitalFlow(symbol string) (*CapitalFlow, error)
    GetNews(symbol string) ([]News, error)
}
```

业务代码不能直接依赖第三方 SDK。

---

# 32. 数据库

## PostgreSQL

```text
users
user_settings

conversations
conversation_messages

research_sessions
research_events

user_memories

watchlists
watchlist_items

paper_accounts
paper_orders
paper_positions
paper_trades

notification_rules
notifications
```

## ClickHouse

```text
stock_quotes
stock_kline_1d
stock_kline_5m
stock_kline_1m
stock_indicators
capital_flow
sector_quotes
sector_indicators
news
```

## Redis

```text
quote:{symbol}
hot:stocks
rank:gainers
rank:volume
ai:session:{id}
user:{id}:watchlist
```

---

# 33. research_sessions

```text
id
user_id
conversation_id
subject_type
subject_id
goal
status
plan_json
result_json
created_at
updated_at
```

subject_type：

```text
stock
sector
topic
market
portfolio
```

---

# 34. research_events

记录 Agent 工作过程：

```text
id
research_session_id
event_type
tool_name
input_json
output_json
status
created_at
```

用途：

- Agent 调试
- 前端展示
- 问题追踪
- Token / Tool 成本统计

---

# 35. API

## Conversation

```text
POST /api/v1/conversations
GET  /api/v1/conversations
GET  /api/v1/conversations/:id
DELETE /api/v1/conversations/:id
```

## AI

```text
POST /api/v1/ai/chat
POST /api/v1/ai/research
```

## Research

```text
GET /api/v1/research/:id
GET /api/v1/research/:id/events
```

## Memory

```text
GET /api/v1/memories
PATCH /api/v1/memories/:id
```

## Market

```text
GET /api/v1/stocks/search
GET /api/v1/stocks/:symbol
GET /api/v1/quotes
GET /api/v1/kline
GET /api/v1/indicators
```

## Paper Trading

```text
GET /api/v1/paper/account
GET /api/v1/paper/orders
POST /api/v1/paper/orders
GET /api/v1/paper/positions
GET /api/v1/paper/trades
```

## Realtime

```text
/ws/market
```

---

# 36. React Web

```text
apps/web/
├── src/
│   ├── app/
│   ├── routes/
│   ├── components/
│   │   ├── chat/
│   │   ├── agent/
│   │   ├── research/
│   │   ├── stock/
│   │   ├── chart/
│   │   └── trading/
│   ├── hooks/
│   ├── stores/
│   ├── api/
│   └── types/
```

核心组件：

```text
ChatSidebar
ConversationList
ChatComposer
AgentProgress
ToolCall
ResearchBlock
StockCard
KlineChart
IndicatorCard
NewsList
ComparisonTable
RiskCard
TradingPlan
ResearchWorkspace
```

---

# 37. Android

```text
apps/mobile/
├── app/
│   ├── index.tsx
│   ├── research/
│   ├── conversation/
│   ├── stock/
│   ├── watchlist/
│   └── paper/
├── components/
│   ├── chat/
│   ├── research/
│   ├── stock/
│   └── trading/
└── stores/
```

移动端不强行复制 Web 双栏。

核心：

```text
Conversation
 ↓
Research Block
 ↓
展开详情
```

---

# 38. 技术栈

## Web

```text
React
TypeScript
Vite
Tailwind CSS
shadcn/ui
ECharts
TanStack Query
Zustand
Lucide
```

## Android

```text
React Native
Expo
Expo Router
NativeWind
TanStack Query
Zustand
Reanimated
Gesture Handler
```

## Backend

```text
Go
Python
PostgreSQL
Redis
ClickHouse
Docker
```

---

# 39. Monorepo

```text
ai-stock/
├── apps/
│   ├── web/
│   └── mobile/
├── packages/
│   ├── types/
│   ├── api-client/
│   ├── business/
│   └── design-tokens/
├── services/
│   ├── api-go/
│   └── ai-python/
├── infra/
│   ├── docker/
│   ├── postgres/
│   ├── redis/
│   └── clickhouse/
└── docs/
```

共享：

```text
TypeScript Types
API Client
Business Logic
Design Tokens
```

不强制共享：

```text
UI
Navigation
Chart Interaction
Layout
```

---

# 40. 安全边界

Agent 可以：

```text
读取行情
读取新闻
读取指标
执行筛选
生成研究
生成模拟交易计划
```

Agent 不可以：

```text
直接真实下单
修改用户资金
执行任意 SQL
绕过权限
```

Tool 必须白名单化。

未来真实交易必须采用：

```text
AI
 ↓
交易计划
 ↓
用户确认
 ↓
独立 Trading Service
 ↓
Broker API
```

---

# 41. V2 开发阶段

## Phase 1：Conversation

实现：

```text
会话
AI Chat
SSE
Agent Progress
Conversation History
```

目标：

> 像 ChatGPT 一样和股票 AI 对话。

## Phase 2：Research Agent

实现：

```text
股票研究
行情 Tool
K线 Tool
指标 Tool
新闻 Tool
Research Workspace
```

目标：

> AI 能真正“研究”股票，而不是聊天。

## Phase 3：Quant Agent

实现：

```text
自然语言选股
行业分析
股票对比
评分
热点
```

## Phase 4：Memory + Companion

实现：

```text
用户关注
研究历史
主动提醒
每日市场摘要
```

## Phase 5：Paper Trading

实现：

```text
交易计划
模拟订单
持仓
收益
复盘
```

---

# 42. MVP 最小闭环

第一版只做：

```text
AI Chat
+
股票研究
+
K线
+
技术指标
+
新闻
+
Research Workspace
+
自选股
+
模拟交易
```

完整体验：

```text
打开 App
 ↓
“帮我看看今天市场”
 ↓
AI 自动研究
 ↓
发现热点
 ↓
“深入看看机器人”
 ↓
机器人 Research Workspace
 ↓
“有哪些股票值得研究？”
 ↓
Quant Agent 筛选
 ↓
“分析其中一个”
 ↓
股票研究
 ↓
“如果模拟交易呢？”
 ↓
Trading Plan
 ↓
用户确认
 ↓
模拟交易
 ↓
AI 后续跟踪
 ↓
复盘
```

---

# 43. 产品最终形态

用户最终感知到的不是：

> 一个股票行情软件。

而是：

> **一个懂市场、懂数据、能帮我持续研究股票的 AI 投研伙伴。**

最终架构：

```text
                 AI 投研伙伴
                      │
          ┌───────────┴───────────┐
          │                       │
     Conversation             Research
          │                       │
          │              ┌────────┼────────┐
          │              │        │        │
          │             行情      新闻      Quant
          │              │        │        │
          └──────────────┴────────┴────────┘
                         │
                    AI 研究结论
                         │
                    模拟交易
                         │
                       复盘
                         │
                    持续陪伴
```

核心原则：

```text
Conversation First
        ↓
Agent Driven
        ↓
Research Workspace
        ↓
Structured UI
        ↓
Paper Trading
        ↓
Memory & Companion
```

AI 负责：

```text
理解
规划
查询
计算
分析
解释
追踪
复盘
```

用户负责：

```text
提出目标
查看信息
判断与决策
确认模拟交易
```

这样后续增加财报、公告、行业研究、策略回测、组合管理、多市场甚至真实券商能力，都可以继续挂在 Conversation + Research Workspace 这套核心交互之上。
