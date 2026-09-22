# Agent Prompts

每个子目录是一个独立 Agent。新增时复制一份现有目录即可，不必改业务代码。

```text
prompts/
  loader.py            扫描目录、渲染 {{变量}}
  stock_analyst/       个股分析（已接入分析 Profile）
  trading_planner/     交易计划（预留）
  screening_nl/        自然语言选股理解（板块匹配 + 算法选择）
  news_digest/         新闻摘要（预留）
  ensemble_judge/      多模型裁判（预留）
```

## 目录约定

| 文件 | 作用 |
|---|---|
| `agent.json` | 编码、中文名、任务、角色、是否启用 |
| `system.md` | 系统提示词 |
| `user.md` | 用户提示词模板 |

`agent.json` 字段：

```json
{
  "code": "stock_analyst",
  "name": "个股分析员",
  "task": "stock_analysis",
  "role": "analyst",
  "version": "1.0",
  "enabled": true,
  "description": "根据量化上下文输出方向、评分、风险和建议"
}
```

`role` 与模型目录对齐：`analyst` / `screener` / `news` / `planner` / `judge`。

## 模板变量

用户模板里可用 `{{stock.name}}`、`{{indicators.ma5}}` 这类点路径，缺失时写成 `-`。另外两个合成字段：

- `{{algorithm_lines}}`：五套算法命中/未命中列表
- `{{context_json}}`：精简上下文 JSON，给尚未定制字段的 Agent 兜底

## 接到一次分析

在 `config/llm.json` 的 Profile 上写 `agentCode`：

```json
{
  "code": "stock_analysis_default",
  "agentCode": "stock_analyst"
}
```

未写时默认 `stock_analyst`。目录列表见 `GET /v1/agents`。
