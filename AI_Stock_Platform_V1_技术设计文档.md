# AI Stock Research & Paper Trading Platform
## V1 技术设计文档

> 版本：V1.2  
> 文档状态：开发基线  
> 变更：数据源按现有日报实现落地（东财主源 + 腾讯/新浪 K 线兜底 + 巨潮公告）；V1 内置算法对齐五套综合选股 + RPS/板块强度，其余算法后续再扩
> 技术栈：React + React Native + Go + Python  
> 目标：Web + Android，AI 智能投研 + 智能选股 + 新闻热点 + 模拟交易  
> 交易模式：仅模拟交易，不接入真实券商交易

---

# 1. 项目概述

## 1.1 项目定位

构建一个面向个人投资研究的 AI 股票分析平台，通过第三方金融数据接口获取行情、K 线、资金、新闻等数据，由 Go 后端负责业务 API、实时行情和模拟交易，由 Python 负责量化指标、选股、AI Agent 与分析能力。

核心能力：

1. 实时行情
2. 股票搜索与详情
3. K 线与技术指标
4. AI 股票分析
5. 智能选股
6. 热点新闻
7. 个股新闻聚合
8. 交易计划生成
9. 模拟交易
10. 自选股与智能提醒
11. 多模型分析配置与综合选择
12. 可扩展选股/评分算法筛选

## 1.2 V1 产品边界

### V1 包含

- A 股股票基础信息
- 行情与 K 线
- MA / MACD / RSI 等技术指标
- 新闻与热点
- 行业/板块数据
- 资金流相关数据
- AI 股票分析
- AI 智能选股
- AI 交易计划
- 多 LLM 接入、切换、回退与综合选择
- 算法注册与筛选编排（V1 先落地五套综合选股 + RPS/板块强度）
- 东方财富主行情，腾讯/新浪 K 线兜底，巨潮公告
- 自选股
- 消息提醒
- 模拟账户
- 模拟买入/卖出
- 持仓、订单、成交、收益统计

### V1 不包含

- 真实券商交易
- 自动实盘下单
- 高频交易
- Level-2 深度行情
- 复杂机器学习预测
- 复杂策略编辑器
- 跟单/社区
- 策略商城
- 桌面端
- 多券商账户

---

# 2. 总体技术架构

```text
                         ┌──────────────────────┐
                         │   第三方金融数据源     │
                         │ 东财 / 腾讯 / 新浪 / 巨潮 │
                         └──────────┬───────────┘
                                    │
                                    ▼
┌───────────────────────────────────────────────────────────┐
│                        Go Backend                         │
│                                                           │
│ Auth / User / Stock / Market / News / Watchlist           │
│ Screening / ModelConfig / Algorithm / Notification        │
│ Paper Trading / Strategy / AI                             │
│                                                           │
│ REST API + WebSocket + SSE                                │
└───────────────┬───────────────────────┬───────────────────┘
                │                       │
                ▼                       ▼
        ┌──────────────┐        ┌──────────────────┐
        │ PostgreSQL   │        │ Redis            │
        │ 业务数据      │        │ 实时缓存/热点      │
        └──────────────┘        └──────────────────┘
                │
                │
                ▼
        ┌──────────────────┐
        │ ClickHouse       │
        │ 行情/历史数据     │
        └──────────────────┘

                ▲
                │ HTTP / gRPC
                │
        ┌──────────────────────────┐
        │ Python AI / Quant        │
        │                          │
        │ LLM Router / Ensemble    │
        │ Agent                    │
        │ Technical Analysis       │
        │ Algorithm Registry       │
        │ Screening / Scoring      │
        │ Trading Plan             │
        │ Backtest                 │
        └──────────────────────────┘

        ┌──────────────────────┐
        │ Web                  │
        │ React + TS + Vite    │
        └──────────────────────┘

        ┌──────────────────────┐
        │ Android              │
        │ React Native + Expo  │
        └──────────────────────┘
```

---

# 3. 技术栈

## 3.1 Web

| 技术 | 用途 |
|---|---|
| React | UI |
| TypeScript | 类型系统 |
| Vite | 构建 |
| Tailwind CSS | 样式 |
| shadcn/ui | Web UI 组件 |
| ECharts | K线/指标图表 |
| TanStack Query | 服务端状态 |
| Zustand | 客户端状态 |
| React Router / Expo Router | 路由 |
| Lucide React | 图标 |

## 3.2 Android

| 技术 | 用途 |
|---|---|
| React Native | Android |
| Expo | 工程基础 |
| Expo Router | 路由 |
| NativeWind | Tailwind 风格 |
| Zustand | 客户端状态 |
| TanStack Query | API 数据 |
| React Native Reanimated | 动画 |
| Gesture Handler | 手势 |

Web 与 Android 不强求 100% UI 复用。

共享：

- TypeScript 类型
- API Client
- Business Logic
- Design Tokens
- 数据模型

平台独立：

- UI Component
- 页面布局
- 导航交互
- 图表交互

---

# 4. Monorepo

```text
ai-stock/
├── apps/
│   ├── web/
│   └── mobile/
│
├── packages/
│   ├── types/
│   ├── api-client/
│   ├── design-tokens/
│   └── business/
│
├── services/
│   ├── api-go/
│   └── ai-python/
│
├── infra/
│   ├── docker/
│   ├── postgres/
│   ├── redis/
│   └── clickhouse/
│
├── docs/
│
├── package.json
├── pnpm-workspace.yaml
└── README.md
```

---

# 5. Go Backend

## 5.1 目录

```text
services/api-go/
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── auth/
│   ├── user/
│   ├── stock/
│   ├── market/
│   ├── news/
│   ├── watchlist/
│   ├── screening/
│   ├── notification/
│   ├── trading/
│   ├── strategy/
│   ├── modelconfig/
│   ├── algorithm/
│   └── ai/
│
├── pkg/
│   ├── response/
│   ├── middleware/
│   ├── logger/
│   └── provider/
│
└── migrations/
```

## 5.2 Go 模块职责

### stock

负责：

- 股票搜索
- 股票基础信息
- 股票分类
- 行业关系

### market

负责：

- 实时行情
- K 线
- 指标数据
- 行情缓存
- WebSocket

### news

负责：

- 新闻
- 热点
- 新闻聚合
- 新闻标签

### screening

负责：

- 选股条件
- 算法 Profile 选择
- Python 筛选服务调用
- 筛选结果缓存

### modelconfig

负责：

- LLM Provider / Model 配置
- 分析 Profile（单模型 / 回退 / 综合）
- 密钥不入库，只存引用
- 对 Python 下发运行时模型路由

### algorithm

负责：

- 算法目录与启用状态
- 算法 Profile（单算法 / 流水线 / 加权综合）
- 参数校验
- 后续新增算法只注册，不改业务主链路

### trading

负责：

- 模拟账户
- 委托
- 成交
- 持仓
- 收益

### ai

负责：

- AI 会话
- SSE
- 按任务绑定分析 Profile
- Python Agent 调用
- AI 卡片结果
- 多模型投票/综合结果透传

---

# 6. Python AI / Quant

```text
services/ai-python/
├── agent/
│   ├── base_agent.py
│   ├── research_agent.py
│   ├── screening_agent.py
│   ├── news_agent.py
│   └── strategy_agent.py
│
├── llm/
│   ├── providers/
│   ├── router.py
│   ├── ensemble.py
│   └── schemas.py
│
├── tools/
│   ├── quote.py
│   ├── kline.py
│   ├── indicator.py
│   ├── capital.py
│   ├── sector.py
│   ├── news.py
│   ├── financial.py
│   └── risk.py
│
├── quant/
│   ├── indicators/
│   │   ├── ma.py
│   │   ├── macd.py
│   │   ├── rsi.py
│   │   └── kdj.py
│   ├── algorithms/
│   │   ├── base.py
│   │   ├── registry.py
│   │   └── builtin/
│   │       ├── year_high.py
│   │       ├── deep_rebound.py
│   │       ├── forward_train.py
│   │       ├── daily_observe.py
│   │       ├── ma5_align.py
│   │       ├── rps.py
│   │       └── board_strength.py
│   ├── factors/
│   ├── scoring/
│   └── screening/
│
├── backtest/
│
└── main.py
```

---

# 7. AI Agent 架构

AI 不直接访问数据库。

采用：

```text
User
  │
  ▼
Go AI API
  │
  │  附带 analysis_profile / algorithm_profile
  ▼
Python Agent
  │
  ├── LLM Router（按 Profile 选模型）
  ├── Algorithm Registry（按 Profile 选算法）
  ├── get_stock_quote
  ├── get_stock_kline
  ├── get_technical_indicators
  ├── get_capital_flow
  ├── get_sector_info
  ├── get_sector_strength
  ├── get_stock_news
  ├── get_financial_data
  ├── get_risk_analysis
  └── generate_trading_plan
```

Agent 负责：

1. 理解用户问题
2. 判断需要哪些数据
3. 调用工具
4. 调用量化算法得到可计算结论
5. 按模型 Profile 做单模型 / 回退 / 综合推理
6. 输出结构化分析结果（含模型与算法来源）

---

# 8. 模型配置与多模型编排

模型不能写死在代码里。V1 就要把「接入、切换、回退、综合」做成配置。

原则：

```text
Quant 先算数
LLM 再解释
多个 LLM 可以对同一份结构化上下文各自给出结论
编排层决定用哪几个、怎么合成
```

密钥、Endpoint、模型名必须可配置；业务代码只依赖 `ModelProvider` 接口。

## 8.1 能力分层

```text
Provider     供应商接入（OpenAI 兼容 / Anthropic / Azure / Ollama / 自定义）
Model        具体模型（deepseek-chat / gpt-4.1 / qwen-plus ...）
Task         任务类型（个股分析 / 选股理解 / 新闻摘要 / 交易计划 / 综合裁判）
Profile      一次分析怎么用模型（单模型、回退链、综合选择）
```

## 8.2 Provider 与 Model

系统维护模型目录，可启用/停用，不改代码即可切换。

```json
{
  "providers": [
    {
      "code": "openai_compatible",
      "baseUrl": "https://api.example.com/v1",
      "apiKeyRef": "env:LLM_OPENAI_KEY",
      "enabled": true
    },
    {
      "code": "deepseek",
      "baseUrl": "https://api.deepseek.com",
      "apiKeyRef": "env:LLM_DEEPSEEK_KEY",
      "enabled": true
    }
  ],
  "models": [
    {
      "code": "deepseek-chat",
      "providerCode": "deepseek",
      "modelName": "deepseek-chat",
      "roles": ["analyst", "screener"],
      "maxTokens": 8192,
      "timeoutMs": 60000,
      "costTier": "low",
      "enabled": true
    },
    {
      "code": "qwen-plus",
      "providerCode": "openai_compatible",
      "modelName": "qwen-plus",
      "roles": ["analyst", "judge"],
      "maxTokens": 16384,
      "timeoutMs": 90000,
      "costTier": "mid",
      "enabled": true
    }
  ]
}
```

约束：

- API Key 只存环境变量或密钥管理引用，不进 PostgreSQL 明文
- 同一 Provider 可挂多个 Model
- `roles` 决定模型能出现在哪些任务里：`analyst` / `screener` / `news` / `planner` / `judge`
- 模型可按 `costTier`、延迟、上下文长度做路由偏好

## 8.3 分析 Profile

一次 AI 调用绑定一个 Profile，而不是绑定写死的模型名。

V1 内置任务：

```text
stock_analysis      个股分析
screening_nl        自然语言转选股条件
news_digest         新闻摘要与情绪
trading_plan        交易计划
ensemble_judge      多模型结论合成（可选）
```

Profile 运行模式：

```text
single      只用一个主模型
fallback    主模型失败则按链回退
ensemble    多个模型并行分析，再综合选择
```

示例：个股分析默认 Profile

```json
{
  "code": "stock_analysis_default",
  "task": "stock_analysis",
  "mode": "ensemble",
  "models": [
    { "modelCode": "deepseek-chat", "weight": 0.4, "temperature": 0.2 },
    { "modelCode": "qwen-plus", "weight": 0.4, "temperature": 0.2 },
    { "modelCode": "gpt-4.1-mini", "weight": 0.2, "temperature": 0.1 }
  ],
  "fallback": ["deepseek-chat", "qwen-plus"],
  "judgeModelCode": "qwen-plus",
  "aggregation": {
    "score": "weighted_average",
    "direction": "majority_vote",
    "summary": "judge_synthesize",
    "cards": "merge_by_type"
  },
  "timeoutMs": 90000,
  "maxParallel": 3
}
```

用户侧：

- 系统默认 Profile
- 用户可在「分析设置」里选择：快速单模型 / 稳健回退 / 多模型综合
- 单次对话也可临时指定 `profileCode` 或临时 `modelCodes`

## 8.4 综合选择怎么做

所有参与模型拿到同一份 Quant 结果和同一套输出协议，禁止各算各的指标。

```text
行情 / 指标 / 资金 / 新闻 / 算法分
                │
                ▼
        结构化 Context Pack
                │
     ┌──────────┼──────────┐
     ▼          ▼          ▼
   Model A    Model B    Model C
     │          │          │
     └──────────┼──────────┘
                ▼
         Ensemble Aggregator
                │
                ▼
     统一 JSON + 各模型投票明细
```

聚合规则 V1：

| 字段 | 综合方式 |
|---|---|
| 各维度分数 / 综合分 | 加权平均 |
| 多空方向 / 建议动作 | 多数投票，平票看权重或 judge |
| 风险等级 | 取更保守的一侧 |
| 卡片结论 | 按 cardType 合并，冲突交给 judge |
| 摘要与交易计划 | 可选 judge 模型再写一版 |

输出必须带可追溯信息：

```json
{
  "profileCode": "stock_analysis_default",
  "mode": "ensemble",
  "usedModels": ["deepseek-chat", "qwen-plus"],
  "votes": [
    { "modelCode": "deepseek-chat", "direction": "bullish", "score": 74 },
    { "modelCode": "qwen-plus", "direction": "neutral", "score": 61 }
  ],
  "final": {
    "direction": "bullish",
    "score": 68.8
  }
}
```

前端可展示「X 个模型综合」而不是假装只有一个上帝模型。

## 8.5 路由与降级

```text
1. 解析任务 → 找到 Profile
2. 过滤未启用 / 无额度 / 不支持该 role 的模型
3. single：直接调用
4. fallback：按链重试
5. ensemble：并行调用，部分失败则用成功子集综合
6. 全部失败：返回可重试错误，不编造分析
```

还需要：

- 单用户 Token / 次数限额
- 模型级超时
- 调用日志（model、latency、token、成功/失败）
- 成本统计，便于以后关掉贵模型

## 8.6 V1 落地范围

V1 必须有：

- 至少 2 个可配置 Provider
- 模型启用/停用
- 3 个内置 Profile：快速、默认、综合
- 失败回退
- 综合模式下分数加权 + 方向投票

V1 不做：

- 按行情自动换模型的强化学习
- 用户自训练模型
- 无限制的任意第三方任意参数裸调

---

# 9. 算法注册与可扩展筛选

选股和评分不要把规则写死在 Agent Prompt 里。

算法是插件：先注册，再被 Profile 选用。V1 先把框架和少量内置算法做出来，后续只加算法实现，不改主链路。

## 9.1 为什么要算法配置

```text
LLM 负责：自然语言 → 条件 / 解释结果
Algorithm 负责：可重复、可回测、可解释的计算
Profile 负责：用哪几个算法、权重多少、怎么合并
```

后续增加尾盘、竞价、蓝色钻石、事件驱动等，只需：

1. 实现算法
2. 注册元数据与参数 Schema
3. 放到某个筛选 Profile 里启用

## 9.2 算法元数据

```json
{
  "code": "year_high",
  "name": "率先一年新高",
  "category": "screening",
  "version": "1.0.0",
  "enabled": true,
  "inputs": ["kline", "rps", "turnover", "board_rps5", "gainers_page1"],
  "outputs": ["pass", "score", "reason"],
  "paramsSchema": {
    "minRps": { "type": "number", "default": 95 },
    "maxTurnover": { "type": "number", "default": 20 },
    "requireFirstPage": { "type": "bool", "default": true }
  }
}
```

分类：

```text
screening    过滤股票集合
scoring      打分
signal       买卖点提示（后续）
risk         风险过滤（后续）
```

统一输出：

```json
{
  "algorithmCode": "year_high",
  "symbol": "600519",
  "pass": true,
  "score": 46,
  "reason": "近5日创250日新高，RPS250≥95，且在当日涨幅榜第一版",
  "metrics": {
    "rps250": 97.2,
    "yearHighBarsAgo": 0,
    "inGainersPage1": true
  }
}
```

## 9.3 V1 内置算法（对齐现有日报选股）

V1 先把已经在用、可复现的规则算法迁进来。公式参数进配置，不要写死在 Prompt。

### 综合选股五套（默认 Profile，优先级即默认权重序）

| code | 名称 | 基准分 | 要点 |
|---|---|---|---|
| `year_high` | 率先一年新高 | 46 | 先按板块指数 RPS5 圈主流板块；RPS120/250 ≥ 95；近 5 日创 250 日新高；换手 < 20%；**只看当日涨幅榜第一版（约 29 只）**，挤不进第一版不纳入 |
| `deep_rebound` | 高 RPS 深调回升 | 40 | 同上 RPS 前提；收盘 ≥ 一年最高价 50%；第 1 / 第 2 个基底可参与，第 3 个起谨慎 |
| `forward_train` | 顺向火车轨 | 36 | RPS120+RPS250 > 185；站稳均线且均线顺向；近 20 日回撤可控；换手 < 15%；优先右侧年高、10 日线下买点 |
| `daily_observe` | 火车每日观察 | 30 | 换手 < 25%；120 日回撤与年高位置达标；高 RPS 且在年高附近（XG1~XG4 任一） |
| `ma5_align` | 五日线多头共振 | 24 | 硬门槛：站上 MA5 且 MA5 向上，最好 MA5>MA10>MA20；量比确认；乖离过大 / 连续涨停 / ST 一票否决；短线，严格止损 |

同一只票可命中多套。综合分 = 命中算法基准分 × 盘面权重 + 质量加分（主流板块、第一版、基底序号等）。默认取前 30。

盘面（涨跌家数 / 情绪 / 指数中期信号 / 主线）会**改这五套权重**，不是只印在报告顶部：

```text
指数偏多 / 情绪健康     年新高、火车轨加权
指数 avoid / 情绪退潮   年新高、火车轨大幅降权（假突破）
情绪过热                 偏向深调回升
盘面过冷或指数 avoid     整表只作观察池，不当作可买清单
当日跌幅过大（如 < -7%） 不进可买入
```

### 配套计算（算法依赖，不是独立选股入口）

```text
rps                 RPS20/50/120/250（全市场涨幅横向百分位；窗口用 5日/60日/YTD 就近映射）
board_strength      板块综合强度：涨幅 + 上涨扩散 + 主力资金 + 相对近5日加速
board_rps5          板块指数 RPS5，用于圈主流板块
gainers_page1       当日涨幅榜第一版
ma / macd / kdj / rsi / bias   短线共振与否决
```

### 后续扩充（本阶段只预留注册，不实现）

```text
blue_diamond        蓝色钻石 / 砖石底（回踩20日线观察池）
tail_end            尾盘：折算全天量 + 日内位置 + 五日线
morning_auction     早盘竞价：高开聚板块，一字/顶板默认不追
event_turnaround    ST 摘帽 / 重整 / 收购事件
custom_user_formula 用户自定义
```

不要在 V1 做复杂机器学习预测。新算法走同一 Registry。

## 9.4 算法 Profile

和模型 Profile 对称，一次筛选绑定一个算法编排。

模式：

```text
single      只用一个算法
pipeline    按顺序过滤（前一个 pass 才进下一个）
ensemble    多算法并行，再按权重/交并集综合
```

示例：默认选股 Profile

```json
{
  "code": "screening_default",
  "mode": "ensemble",
  "algorithms": [
    { "code": "year_high", "weight": 1.0, "baseScore": 46 },
    { "code": "deep_rebound", "weight": 1.0, "baseScore": 40 },
    { "code": "forward_train", "weight": 1.0, "baseScore": 36 },
    { "code": "daily_observe", "weight": 1.0, "baseScore": 30 },
    { "code": "ma5_align", "weight": 1.0, "baseScore": 24 }
  ],
  "fusion": {
    "setLogic": "weighted_rank",
    "marketTapeAdjust": true,
    "limit": 30,
    "crashDropPct": -7
  }
}
```

`weight` 初值为 1，运行时由盘面 `strategyBias` 加减。用户可在分析设置里改优先级，不必改代码。

`setLogic`：

```text
intersection     全部 pass 才入选
union            任一 pass 即入选
weighted_rank    先各自打分，再加权排序，用 minPassCount / minFinalScore 截断
```

## 9.5 筛选执行链路

```text
用户自然语言或手动条件
        │
        ▼
Screening Agent（用 screening_nl 模型 Profile）
        │
        ▼
结构化 conditions + algorithm_profile
        │
        ▼
Quant Engine
  ├── 拉全市场快照 / K线 / 资金 / 板块 / 涨幅榜
  ├── 算 RPS、均线、板块强度
  ├── 算盘面 strategyBias（改五套权重）
  ├── Registry 并行跑启用算法
  └── Fusion 得到股票列表 + 命中算法标签 + 解释
        │
        ▼
可选：再用分析 Profile 对 Top N 做多模型点评
```

不要让 LLM 自己扫全市场算指标。用户也可跳过自然语言，直接勾选算法和参数。

## 9.6 扩展约定

新增算法必须满足：

- 实现统一 `Algorithm` 接口：`code` / `validate_params` / `run`
- 声明依赖数据，由 Engine 准备，算法内不直连第三方
- 输出 `pass + score + reason + metrics`
- 参数全部来自配置，禁止魔法数写死
- 可单测、可对历史截面回放

Go 只保存配置和调度结果；Python Registry 才执行计算。

---

# 10. AI Tools

## 10.1 get_stock_quote

```json
{
  "symbol": "600519",
  "market": "SH"
}
```

返回：

```json
{
  "symbol": "600519",
  "price": 1500.20,
  "change": 1.52,
  "volume": 123456,
  "turnover": 456789000
}
```

## 10.2 get_stock_kline

```json
{
  "symbol": "600519",
  "period": "1d",
  "limit": 240
}
```

## 10.3 get_technical_indicators

```json
{
  "symbol": "600519",
  "indicators": [
    "MA5",
    "MA10",
    "MA20",
    "MACD",
    "KDJ",
    "RSI",
    "RPS20",
    "RPS50",
    "RPS120",
    "RPS250",
    "BIAS5"
  ]
}
```

## 10.4 get_capital_flow

返回：

```json
{
  "main_inflow": 120000000,
  "main_outflow": 80000000,
  "net_inflow": 40000000
}
```

## 10.5 get_stock_news

返回：

```json
[
  {
    "title": "xxx",
    "source": "xxx",
    "publishedAt": "2026-09-20T10:20:00+08:00",
    "sentiment": "positive"
  }
]
```

## 10.6 generate_trading_plan

返回结构化计划：

```json
{
  "symbol": "600519",
  "direction": "watch",
  "entry": {
    "min": 1480,
    "max": 1510
  },
  "stopLoss": 1430,
  "targets": [
    1580,
    1650
  ],
  "position": "20%",
  "riskLevel": "medium",
  "reason": [
    "MA20附近获得支撑",
    "MACD出现改善",
    "行业指数保持强势"
  ]
}
```

注意：该结果仅用于研究与模拟交易，不直接执行真实交易。

---

# 11. AI 输出协议

AI 不建议只返回 Markdown。

采用：

```json
{
  "type": "stock_analysis",
  "symbol": "600519",
  "profileCode": "stock_analysis_default",
  "algorithmProfileCode": "scoring_default",
  "summary": {},
  "cards": [],
  "risk": {},
  "tradingPlan": {},
  "ensemble": {
    "mode": "ensemble",
    "usedModels": [],
    "votes": [],
    "final": {}
  }
}
```

## Card 类型

```text
technical
capital
sector
news
financial
risk
trading_plan
```

例如：

```json
{
  "type": "card",
  "cardType": "technical",
  "title": "技术面",
  "score": 72,
  "items": [
    {
      "name": "趋势",
      "value": "偏强"
    },
    {
      "name": "MACD",
      "value": "改善"
    },
    {
      "name": "RSI",
      "value": "中性"
    }
  ]
}
```

这样 Web 和 Android 可以共享同一套 AI 数据协议。

---

# 12. SSE AI 流式协议

接口：

```http
POST /api/v1/ai/chat
Accept: text/event-stream
```

事件：

```text
event: thinking
data: {}

event: profile
data: {
  "analysisProfile": "stock_analysis_default",
  "algorithmProfile": "scoring_default"
}

event: tool_call
data: {
  "tool": "get_stock_quote"
}

event: tool_result
data: {}

event: algorithm
data: {
  "code": "year_high",
  "status": "done"
}

event: model
data: {
  "modelCode": "deepseek-chat",
  "status": "running"
}

event: vote
data: {
  "modelCode": "deepseek-chat",
  "direction": "bullish",
  "score": 74
}

event: card
data: {
  "cardType": "technical"
}

event: text
data: {
  "content": "从技术指标来看..."
}

event: done
data: {}
```

前端可以实时渲染：

```text
选用分析 Profile
 ↓
AI思考
 ↓
获取行情
 ↓
获取K线
 ↓
计算指标 / 跑算法
 ↓
获取新闻
 ↓
多模型分析（可并行）
 ↓
综合投票 / Judge 合成
 ↓
输出交易计划
```

---

# 13. 数据存储

## 13.1 PostgreSQL

用于业务数据。

```text
users
user_settings

watchlists
watchlist_items

ai_conversations
ai_messages

paper_accounts
paper_orders
paper_positions
paper_trades

notification_rules
notifications

strategies

llm_providers
llm_models
analysis_profiles
algorithms
algorithm_profiles
algorithm_runs
```

---

# 14. PostgreSQL 核心表

## users

```sql
id
username
email
password_hash
created_at
updated_at
```

## watchlists

```sql
id
user_id
name
created_at
```

## watchlist_items

```sql
id
watchlist_id
symbol
sort_order
created_at
```

## paper_accounts

```sql
id
user_id
initial_cash
available_cash
market_value
total_asset
profit
profit_rate
created_at
updated_at
```

## paper_orders

```sql
id
account_id
symbol
side
order_type
price
quantity
status
created_at
```

side：

```text
BUY
SELL
```

status：

```text
PENDING
FILLED
CANCELLED
REJECTED
```

## paper_positions

```sql
id
account_id
symbol
quantity
available_quantity
avg_cost
market_price
market_value
profit
profit_rate
updated_at
```

## paper_trades

```sql
id
order_id
symbol
side
price
quantity
amount
fee
created_at
```

## user_settings

```sql
id
user_id
analysis_profile_code
algorithm_profile_code
default_model_code
created_at
updated_at
```

## llm_providers

```sql
id
code
name
kind
base_url
api_key_ref
enabled
created_at
updated_at
```

## llm_models

```sql
id
code
provider_id
model_name
roles
max_tokens
timeout_ms
cost_tier
enabled
created_at
updated_at
```

## analysis_profiles

```sql
id
code
task
mode
config_json
is_system
enabled
created_at
updated_at
```

`config_json` 保存模型列表、权重、回退链、聚合方式。系统预置 Profile 不可删，用户可复制后改。

## algorithms

```sql
id
code
name
category
version
enabled
params_schema_json
default_params_json
created_at
updated_at
```

算法实现代码在 Python Registry；本表只是目录与开关。V1 预置：`year_high` / `deep_rebound` / `forward_train` / `daily_observe` / `ma5_align`。后续扩充算法时插入记录即可。

## algorithm_profiles

```sql
id
code
mode
config_json
is_system
enabled
created_at
updated_at
```

## algorithm_runs

```sql
id
user_id
profile_code
input_json
result_json
status
created_at
```

用于选股/评分一次执行的结果缓存与追溯。

---

# 15. ClickHouse

用于高频查询和历史行情。

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

推荐：

- PostgreSQL：业务数据
- ClickHouse：行情与分析数据
- Redis：实时缓存

不要把大量 K 线数据全部放 PostgreSQL。

---

# 16. Redis

用途：

```text
实时行情
热点股票
涨跌幅排行
成交量排行
资金流排行
WebSocket 状态
AI Session
用户缓存
```

Key 示例：

```text
quote:600519
quote:000001
hot:stocks
rank:gainers
rank:volume
ai:session:{id}
user:{id}:watchlist
```

---

# 17. 第三方数据 Provider

必须设计 Provider 抽象层。业务层永远不要直接依赖某个网站的 URL。

V1 数据来源对齐现有可用的公开接口（全部需限流、多节点轮换、失败重试）：

| 能力 | 主源 | 兜底 | 说明 |
|---|---|---|---|
| 股票列表 / 实时快照 | 东方财富 push2 clist | 腾讯行情 | 价、涨跌、量比、换手、成交额、市值、行业、主力净流入 |
| 个股 / 指数实时行情 | 东方财富 | 腾讯 | 指数必须显式带市场，避免 000001 被当成平安银行 |
| 日 K | 东财 push2his、腾讯前复权、新浪 | 三源探测后取最新 | 见下方新鲜度 |
| 集合竞价 / 开盘快照 | 东方财富 | — | 9:15–9:30 虚拟匹配价、竞价额；默认排除北交所以免 30cm 淹没主板 |
| 板块快照 / 成分 / 板块 RPS5 | 东方财富 | — | 行业为主，概念可选 |
| 涨跌家数 / 涨停池 / 强势股数 | 东方财富 | — | 盘面情绪与主线 |
| 资金流 / 概念 | 东方财富多 host | — | push2 / push2delay 轮换，整批失败时重跑 |
| 热点新闻 | 东财重要快讯 | — | 优先 A 股标签 |
| 公告检索 | 巨潮资讯 | — | 摘帽 / 重整 / 收购等关键词，比行情站公告稳 |
| 名称联想 | 腾讯 / 新浪 / 东财 | 互相补 | 中文名 → 6 位代码 |
| 盘口深度（买一等） | 腾讯 | — | 封单厚度等后续能力用 |

雪球大 V 观点、登录 Cookie **不纳入 V1 主链路**（可选后续）。

```go
type MarketProvider interface {
    GetQuote(symbol string) (*Quote, error)
    GetKline(symbol string, period string) ([]Kline, error)
    GetActiveStocks(opts ListOpts) ([]StockSnapshot, error)
    GetAuctionStocks(opts ListOpts) ([]AuctionSnapshot, error)
    GetCapitalFlow(symbol string) (*CapitalFlow, error)
    GetBoards(opts BoardOpts) ([]BoardSnapshot, error)
    GetMarketBreadth() (*Breadth, error)
    GetLimitPools(date string) (*LimitPools, error)
    GetNews(symbol string) ([]News, error)
    GetHotNews(limit int) ([]News, error)
}

type AnnouncementProvider interface {
    Search(keyword string, opts SearchOpts) ([]Announcement, error)
}
```

V1 实现：

```text
EastmoneyProvider     主行情 / 列表 / 板块 / 资金 / 新闻 / 竞价
TencentProvider       实时报价、名称、K 线兜底、盘口
SinaProvider          K 线兜底、名称联想
CninfoProvider        公告
KlineAggregator       探测三源日期，选最新；必要时用实时 bar 补当天
```

### K 线新鲜度（必须做）

日 K 在盘中经常缺当天那根。必须用**两条独立链路**交叉验证：

```text
K线基准日 = 东财 / 腾讯 / 新浪 里日期最新的一份
实时行情日 = 行情时钟接口
若实时已到今天、K 线只到昨天 → 显式告警，指标按旧收盘算，不当作当日盘中结论
开盘前 / 周末 / 节假日不告警
```

某个源连续失败则本次运行内跳过，避免每只票都空等。

好处：

- 更换数据源成本低
- 可以多个 Provider 组合
- 避免供应商锁定
- 方便测试

---

# 18. API 设计

## Auth

```text
POST /api/v1/auth/login
POST /api/v1/auth/register
POST /api/v1/auth/logout
```

## 股票

```text
GET /api/v1/stocks/search?q=
GET /api/v1/stocks/:symbol
```

## 行情

```text
GET /api/v1/quotes
GET /api/v1/kline
GET /api/v1/indicators
```

## 新闻

```text
GET /api/v1/news
GET /api/v1/hot
GET /api/v1/stocks/:symbol/news
```

## 自选

```text
GET /api/v1/watchlist
POST /api/v1/watchlist/items
DELETE /api/v1/watchlist/items/:id
```

## 选股

```text
POST /api/v1/screening
GET /api/v1/screening/:id
GET /api/v1/algorithms
GET /api/v1/algorithm-profiles
GET /api/v1/algorithm-profiles/:code
```

`POST /api/v1/screening` 可带：

```json
{
  "query": "按率先一年新高和五日线共振找票，优先主流板块第一版",
  "algorithmProfileCode": "screening_default",
  "analysisProfileCode": "screening_nl"
}
```

## 模型与分析配置

```text
GET  /api/v1/llm/providers
GET  /api/v1/llm/models
GET  /api/v1/analysis-profiles
GET  /api/v1/analysis-profiles/:code
PUT  /api/v1/settings/analysis
```

管理员（或本机单用户）还可：

```text
POST /api/v1/admin/llm/providers
PUT  /api/v1/admin/llm/models/:code
PUT  /api/v1/admin/analysis-profiles/:code
PUT  /api/v1/admin/algorithms/:code
PUT  /api/v1/admin/algorithm-profiles/:code
```

密钥只写环境变量，Admin API 只改 `api_key_ref` 和启用状态。

## AI

```text
POST /api/v1/ai/chat
GET /api/v1/ai/conversations
GET /api/v1/ai/conversations/:id
```

`POST /api/v1/ai/chat` 可带：

```json
{
  "message": "分析一下贵州茅台",
  "profileCode": "stock_analysis_default",
  "modelCodes": ["deepseek-chat", "qwen-plus"]
}
```

不传则用用户默认 Profile。`modelCodes` 仅允许从已启用模型里临时覆盖。

## 模拟交易

```text
GET /api/v1/paper/account
GET /api/v1/paper/orders
POST /api/v1/paper/orders
DELETE /api/v1/paper/orders/:id

GET /api/v1/paper/positions
GET /api/v1/paper/trades
```

## WebSocket

```text
/ws/market
```

---

# 19. WebSocket 行情协议

客户端：

```json
{
  "action": "subscribe",
  "symbols": [
    "SH.600519",
    "SZ.000001"
  ]
}
```

服务端：

```json
{
  "type": "quote",
  "symbol": "SH.600519",
  "price": 1500.2,
  "change": 1.52,
  "changePercent": 0.85,
  "volume": 123456
}
```

---

# 20. Web 页面结构

```text
/
├── 首页
├── 市场
├── 自选
├── 选股
├── AI投研
├── 模拟交易
└── 我的
    ├── 分析设置（模型 Profile / 算法 Profile）
    └── 模拟账户
```

选股页需要能选择算法 Profile，并展示每个算法的 pass/score/reason。

AI 分析入口需要能选择「快速 / 默认 / 多模型综合」。

## 股票详情页

```text
StockDetail
├── StockHeader
├── QuoteSummary
├── KlineChart
├── IndicatorPanel
├── TechnicalCard
├── CapitalCard
├── SectorCard
├── NewsCard
├── RiskCard
├── AIAnalysis
│   ├── ProfilePicker
│   └── ModelVotes
└── TradingPlanCard
```

---

# 21. 首页

```text
┌────────────────────────────────────────┐
│ AI 股票研究助手                         │
│ [ 输入股票 / 问题 ]                     │
├────────────────────────────────────────┤
│ 大盘                                    │
│ 上证   深证   创业板                    │
├────────────────────────────────────────┤
│ 今日热点                                │
│ 热点1  热点2  热点3                     │
├────────────────────────────────────────┤
│ AI智能发现                              │
│ 股票A   股票B   股票C                   │
├────────────────────────────────────────┤
│ 自选股                                  │
│ 股票 / 价格 / 涨跌 / AI状态             │
└────────────────────────────────────────┘
```

---

# 22. Android 页面

底部 Tab：

```text
首页
市场
自选
选股
我的
```

AI：

```text
首页顶部搜索框
+
悬浮 AI 按钮
+
股票详情页 AI 分析入口
+
我的 → 分析设置
```

不建议 V1 单独增加 AI Tab。分析设置放在「我的」即可。

---

# 23. 智能选股

支持自然语言，也可直接选「综合选股（五算法）」Profile。

```text
帮我按率先一年新高和五日线共振找票，
优先今天主流板块里能进涨幅榜第一版的。
```

Agent 转换成结构：

```json
{
  "algorithmProfileCode": "screening_default",
  "preferredAlgorithms": ["year_high", "ma5_align"],
  "conditions": [
    { "field": "in_gainers_page1", "operator": "=", "value": true },
    { "field": "board_rps5_rank", "operator": "<=", "value": 10 }
  ]
}
```

Python Quant Engine 按算法 Profile 执行筛选，详见第 9 章。

AI 负责：

```text
自然语言 → 条件 + 推荐算法 Profile
```

Quant Engine 负责：

```text
算法 Profile → 各算法计算 → Fusion → 股票集合 + 解释
```

不要让 LLM 自己计算全部指标。用户也可跳过自然语言，直接勾选算法和参数。

---

# 24. 股票综合评分

V1 选股综合分以第 9 章五套算法的命中分为主，而不是另做一套机器学习。

个股研究卡片仍可展示分项分，权重放在算法 Profile / 盘面 bias 里，不要写死在代码中。

分项分示例（详情页展示用，不替代五算法综合）：

```text
技术面       30%
资金面       20%
行业强度     15%
消息面       15%
基本面       10%
风险         10%
```

最终：

```text
综合分 = 
技术分 × 30%
+ 资金分 × 20%
+ 行业分 × 15%
+ 消息分 × 15%
+ 基本面 × 10%
+ 风险分 × 10%
```

注意：

评分用于研究排序，不代表收益预测。  
后续新增评分算法时，只改算法 Profile 的权重和启用项，综合分公式不要散落在多处。

---

# 25. 热点系统

数据流：

```text
新闻
 ↓
新闻清洗
 ↓
实体识别
 ↓
股票关联
 ↓
行业关联
 ↓
关键词聚类
 ↓
热点计算
```

热点对象：

```json
{
  "topic": "AI算力",
  "heat": 92,
  "newsCount": 125,
  "stocks": [
    "xxx",
    "xxx"
  ]
}
```

---

# 26. 智能推送

V1 支持：

### 自选股提醒

```text
涨幅 > 5%
跌幅 < -5%
突破 MA20
MACD 金叉
```

### 热点提醒

```text
热点热度 > 80
```

### 新闻提醒

```text
重要新闻
公司公告
重大事件
```

### AI 异动提醒

例如：

```text
你的自选股 XXX 今日成交量
明显高于过去20日平均水平。
```

---

# 27. 模拟交易

## 交易流程

```text
AI生成交易计划
        ↓
用户查看
        ↓
用户确认
        ↓
模拟下单
        ↓
订单
        ↓
成交
        ↓
持仓更新
        ↓
收益统计
```

AI 不直接执行交易。

---

# 28. 模拟交易账户

默认：

```text
初始资金：1000000
```

账户：

```text
总资产
可用资金
持仓市值
总收益
收益率
今日盈亏
```

---

# 29. 模拟订单

支持：

```text
买入
卖出
```

订单类型 V1：

```text
限价单
```

后续：

```text
市价单
条件单
止盈止损
```

---

# 30. 模拟成交规则

建议 V1：

```text
订单创建
 ↓
根据行情判断是否成交
 ↓
成交后更新持仓
 ↓
扣除模拟手续费
```

需要明确：

- A 股交易时间
- T+1
- 涨跌停
- 最小交易单位
- 手续费
- 印花税
- 滑点

这些规则必须配置化。

---

# 31. 交易计划

AI 输出：

```json
{
  "symbol": "600519",
  "direction": "watch",
  "entryRange": [
    1480,
    1510
  ],
  "stopLoss": 1430,
  "targets": [
    1580,
    1650
  ],
  "position": 0.2,
  "riskLevel": "medium",
  "invalidConditions": [
    "跌破MA20",
    "行业指数转弱"
  ]
}
```

前端：

```text
交易计划

关注区间
¥1480 - ¥1510

止损参考
¥1430

目标区域
¥1580 / ¥1650

模拟仓位
20%

风险
中等

失效条件
• 跌破 MA20
• 行业强度明显下降
```

明确标注：

> AI 交易计划仅用于投资研究与模拟交易，不构成投资建议。

---

# 32. 前端状态管理

## TanStack Query

管理：

```text
行情
K线
新闻
股票详情
自选股
选股结果
模拟账户
订单
持仓
```

## Zustand

管理：

```text
用户状态
AI Session
当前股票
UI 状态
主题
WebSocket 状态
```

---

# 33. 前端 API Client

共享：

```text
packages/api-client
```

示例：

```ts
export interface Stock {
  symbol: string
  name: string
  market: string
}

export interface Quote {
  symbol: string
  price: number
  change: number
  changePercent: number
}
```

Web 和 Android 都使用：

```ts
getStock()
getQuote()
getKline()
getNews()
getAIAnalysis()
listAnalysisProfiles()
listAlgorithms()
createPaperOrder()
```

---

# 34. Design Tokens

共享：

```text
packages/design-tokens
```

例如：

```ts
export const colors = {
  background: '#0B0F14',
  surface: '#111820',
  border: '#26313D',
  text: '#F5F7FA',
  muted: '#8B98A7',
  rise: '#EF4444',
  fall: '#22C55E',
  accent: '#6366F1'
}
```

Web 使用 Tailwind。

Android 使用 NativeWind。

---

# 35. 安全

## 登录

建议：

```text
JWT Access Token
+
Refresh Token
```

## API

必须：

```text
HTTPS
JWT
Rate Limit
参数校验
SQL Injection 防护
日志审计
```

## AI

必须限制：

```text
Prompt Injection
工具调用权限
Token 消耗
单用户请求频率
模型并行数
密钥泄露（日志禁止打印 Key）
```

AI Tool 必须经过白名单。

---

# 36. AI Agent 安全边界

Agent 可以：

```text
读取行情
读取新闻
读取指标
执行筛选
生成分析
生成模拟交易计划
按已启用 Profile 调用模型与算法
```

Agent 不可以：

```text
直接真实下单
修改用户资金
修改数据库核心数据
修改系统级模型密钥与 Provider
绕过权限
执行任意 SQL
任意加载未注册算法
```

---

# 37. 日志与监控

Go：

```text
request log
error log
business log
trade log
```

Python：

```text
agent log
tool call log
model latency
token usage
ensemble vote
algorithm run
tool error
```

核心指标：

```text
API P95
WebSocket 在线数
AI 首 token 延迟
AI 完成耗时
模型调用成功率
算法执行耗时
Provider 成功率
模拟订单成功率
```

---

# 38. 部署

V1 推荐：

```text
Docker Compose
```

服务：

```text
web
api-go
ai-python
postgres
redis
clickhouse
```

结构：

```text
Internet
   │
   ▼
Nginx / Gateway
   │
   ├── Web
   ├── Go API
   ├── WebSocket
   └── SSE
        │
        └── Python AI
```

---

# 39. CI/CD

建议：

```text
GitHub Actions / GitLab CI
```

流程：

```text
git push
 ↓
lint
 ↓
test
 ↓
build
 ↓
docker build
 ↓
deploy
```

Web：

```text
npm/pnpm build
```

Go：

```text
go test ./...
go build
```

Python：

```text
pytest
ruff
```

---

# 40. V1 开发计划

## Sprint 1：行情基础设施

目标：

```text
东财 / 腾讯 / 新浪 / 巨潮 Provider
K线新鲜度交叉验证
股票搜索
行情
K线
MA / MACD / KDJ / RSI / RPS
股票详情
```

交付：

- Go API
- Redis
- ClickHouse
- Web 股票详情
- Android 股票详情

---

## Sprint 2：AI 投研

目标：

```text
模型 Provider / Model 配置
分析 Profile
LLM Router + Ensemble
Python Agent
Tool 系统
SSE
AI 股票分析
AI 卡片
```

交付：

```text
LLM Router
Analysis Profile
AI Research Agent
Technical Card
Capital Card
News Card
Risk Card
Model Votes
```

---

## Sprint 3：智能选股 + 热点

目标：

```text
算法 Registry
算法 Profile
自然语言选股
量化筛选
热点
新闻
智能提醒
```

交付：

```text
Algorithm Registry
Screening Agent
Quant Engine
Hot Topic
Notification
五套综合选股 + RPS / 板块强度
东财主源 + K 线三源兜底 + 新鲜度告警
```

---

## Sprint 4：模拟交易

目标：

```text
模拟账户
订单
成交
持仓
收益
AI交易计划
```

交付：

```text
Paper Account
Paper Order
Paper Position
Paper Trade
Trading Plan
```

---

# 41. 推荐开发顺序

不要一开始同时开发所有功能。

推荐：

```text
① 数据 Provider
        ↓
② Go 行情 API
        ↓
③ 股票详情
        ↓
④ K线 + 指标
        ↓
⑤ Python Quant + Algorithm Registry
        ↓
⑥ 模型配置 + LLM Router
        ↓
⑦ AI Agent
        ↓
⑧ AI 股票分析（含多模型综合）
        ↓
⑨ 智能选股（算法 Profile）
        ↓
⑩ 热点推送
        ↓
⑪ 模拟交易
```

---

# 42. 第一阶段最小可用版本

如果希望尽快看到产品效果，第一版只做：

```text
首页
股票搜索
股票详情
K线
MA/MACD/RSI
AI分析（先单模型，Profile 可切换）
自选股
模拟买卖
```

完整链路：

```text
用户搜索股票
      ↓
股票详情
      ↓
查看 K 线
      ↓
点击 AI 分析
      ↓
AI 获取行情
      ↓
AI 获取指标
      ↓
AI 获取新闻
      ↓
生成分析卡片
      ↓
生成模拟交易计划
      ↓
用户确认
      ↓
模拟下单
```

---

# 43. V1 成功标准

## 技术

```text
行情 API 稳定
WebSocket 稳定
AI SSE 稳定
第三方 Provider 可替换
LLM 可配置且可切换
算法可注册，新增算法不改主链路
```

## 产品

```text
用户可以 10 秒内找到股票
用户可以快速理解股票状态
用户可以通过自然语言进行研究
用户可以看到 AI 分析过程
用户可以切换分析模型 / 综合模式
用户可以用算法 Profile 筛选股票
用户可以生成模拟交易计划
用户可以完成模拟交易
```

## AI

重点不是：

```text
预测某只股票一定涨
```

而是：

```text
数据获取
+
指标计算
+
信息归纳
+
风险识别
+
结构化分析
+
交易计划
```

---

# 44. 后续 V2

V2 可以考虑：

```text
策略回测
因子研究
组合分析
策略编辑器
AI 策略生成
AI 回测
个性化 Agent
事件驱动分析
财报分析
公告分析
更多市场
更多选股算法
用户自定义算法公式
按场景自动选模型
```

后续如果加入真实交易：

```text
AI
 ↓
交易计划
 ↓
用户确认
 ↓
券商 API
 ↓
真实订单
```

真实交易必须与 AI Agent 权限严格隔离。

---

# 45. 核心设计原则

## 原则 1：AI 不负责算数据

```text
LLM = 理解 + 推理 + 解释
Quant = 计算
Go = 业务
ClickHouse = 行情
Redis = 实时
```

## 原则 2：数据源必须可替换

```text
业务代码
   ↓
Provider Interface
   ↓
Eastmoney / Tencent / Sina / Cninfo
```

K 线必须多源聚合，并做新鲜度告警。不要只绑一个行情站。

## 原则 3：AI 输出必须结构化

```text
AI JSON
 ↓
Web Card
 ↓
Android Card
```

## 原则 4：模拟交易和真实交易隔离

V1：

```text
AI → Paper Trading
```

未来：

```text
AI → Trading Plan → User Confirm → Broker
```

## 原则 5：先模块化单体，再拆微服务

V1：

```text
Go Modular Monolith
+
Python AI Service
```

不要一开始：

```text
10+ 微服务
Kafka
K8s
Service Mesh
```

等数据量和团队规模真正需要时再拆。

## 原则 6：模型与算法都必须可配置、可替换

```text
业务代码
   ↓
Analysis Profile / Algorithm Profile
   ↓
LLM Router / Algorithm Registry
   ↓
具体模型实现 / 具体算法实现
```

新增模型：配 Provider + Model，挂到 Profile。  
新增算法：实现接口 + 注册元数据，挂到 Profile。  
不要把模型名和选股公式散落在 Agent Prompt 里。

---

# 46. 最终架构

```text
                    ┌────────────────────┐
                    │  Eastmoney / Tencent  │
                    │  Sina / Cninfo        │
                    └─────────┬──────────┘
                              │
                              ▼
                    ┌────────────────────┐
                    │   Provider Layer   │
                    └─────────┬──────────┘
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
       ┌──────────────┐                ┌──────────────┐
       │ Go Backend   │                │ Python AI    │
       │              │◄──────────────►│ / Quant      │
       │ Business     │                │              │
       │ Market       │                │ LLM Router   │
       │ ModelConfig  │                │ Agent        │
       │ Algorithm    │                │ Algorithms   │
       │ Trading      │                │ Screening    │
       │ API          │                │ Ensemble     │
       └──────┬───────┘                └──────────────┘
              │
       ┌──────┼───────────────┐
       ▼      ▼               ▼
 PostgreSQL Redis        ClickHouse
       │      │               │
       └──────┼───────────────┘
              │
       ┌──────┴───────────┐
       ▼                  ▼
 React Web          React Native
                    Android
```

---

# 47. 结论

V1 建议坚持：

```text
React Web
React Native Android
        ↓
      Go
        ↓
PostgreSQL + Redis + ClickHouse
        ↓
    Python AI
        ↓
可配置 LLM Router + Algorithm Registry
```

产品核心闭环：

```text
行情
 ↓
分析
 ↓
AI研究
 ↓
选股
 ↓
交易计划
 ↓
模拟交易
 ↓
收益复盘
```

这套架构能够在不引入过度复杂基础设施的情况下，为后续增加模型、扩充选股算法、量化策略、回测和真实券商交易预留扩展空间。
