项目说明、启动方式和功能清单见仓库根目录 [README.md](../README.md)。

- V1 页面式投研：[`AI_Stock_Platform_V1_技术设计文档.md`](./AI_Stock_Platform_V1_技术设计文档.md)
- V2 投研伙伴（Conversation First）：[`AI_投研伙伴_V2_AI_Agent_交互与技术设计文档.md`](./AI_投研伙伴_V2_AI_Agent_交互与技术设计文档.md)

## 当前已落地（相对 V2）

- 首页 `/`：AI 对话 + 右侧 Research Workspace + 左侧多会话
- 进房简报：指数、北向、板块、热点、节奏卡；有自选时附带**异动**块
- **Agent Run SSE**：研究计划 / 工具 / 结构化块 / 文本增量；可取消与打断重发
- 对话意图：行情、热点、选股、推荐、个股分析、自选、K 线、模拟、**自选异动**
- 每日推荐归档 + 设置页翻看
- 页内自选异动 / 今日操作主动提醒（服务端 Notification Agent 定时扫描 + 收件箱去重）
- 会话与 research_events 服务端持久化（`data/conversations.json`；浏览器仍缓存 sessionStorage）
- 对话块：`comparison` 双标的对比表、`risk` 风险卡；Workspace 支持 VS 视图
- **Memory**：关注偏好 + 研究历史 + 会话摘要；自由对话注入 LLM；设置页可配置
- **行业 / 题材 Workspace**：对话点板块/题材或说「看看半导体板块」→ 右侧打开 sector/topic 研究台（涨跌、领涨、成分）

换机续作与今晚改动清单见根 README「今晚改动摘要」一节。
