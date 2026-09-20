from __future__ import annotations

import json
import urllib.error
import urllib.request

from .config import get_provider, resolve_key
from .quant_analyst import analyze_quant
from .quota import QuotaError, is_quota_error

SYSTEM = """你是A股投研助手。只根据给定量化数据做中文解读，禁止编造没有出现的数字。
必须只返回一个 JSON 对象，不要 markdown，不要空对象。字段：
direction: bullish|neutral|bearish
score: 0-100 整数
risk: low|mid|high
action: 简短中文建议
summary: 80-120字中文结论
cards: 2-4 张卡片，[{cardType,title,score,items:[{name,value}]}]
cardType 只能是 technical/algorithm/risk/capital，title/name/value 都用中文。
示例：
{"direction":"neutral","score":48,"risk":"mid","action":"观望","summary":"现价在短期均线附近，五套算法未形成共振，先等回踩。","cards":[{"cardType":"technical","title":"技术面","score":48,"items":[{"name":"均线","value":"未多头"}]}]}
"""


def call_model(model: dict, context: dict, temperature: float = 0.2) -> dict:
    provider = get_provider(model["providerCode"]) or {}
    if provider.get("kind") == "local" or model["code"] == "quant-rules":
        return analyze_quant(context)

    key = resolve_key(provider.get("apiKeyRef") or "")
    if not key:
        raise RuntimeError(f"{model['code']} 未配置密钥")

    payload = {
        "model": model["modelName"],
        "temperature": temperature,
        "max_tokens": min(int(model.get("maxTokens") or 1024), 2048),
        "messages": [
            {"role": "system", "content": SYSTEM},
            {"role": "user", "content": _user_prompt(context)},
        ],
    }
    if model.get("jsonMode", True):
        payload["response_format"] = {"type": "json_object"}
    url = (provider.get("baseUrl") or "").rstrip("/") + "/chat/completions"
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {key}",
        },
        method="POST",
    )
    timeout = max(8, int(model.get("timeoutMs") or 60000) / 1000)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as res:
            body = json.loads(res.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "ignore")[:400]
        err = RuntimeError(f"{model['code']} HTTP {exc.code} {detail}")
        if is_quota_error(err) or exc.code in (402, 429):
            raise QuotaError(str(err)) from exc
        raise err from exc
    text = (((body.get("choices") or [{}])[0].get("message") or {}).get("content")) or ""
    parsed = _parse_json(text)
    parsed["modelCode"] = model["code"]
    return parsed


def _parse_json(text: str) -> dict:
    text = (text or "").strip()
    if text.startswith("```"):
        text = text.strip("`")
        if text.startswith("json"):
            text = text[4:]
    if not text.startswith("{"):
        start, end = text.find("{"), text.rfind("}")
        if start >= 0 and end > start:
            text = text[start : end + 1]
    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        data = {}
    direction = data.get("direction") if data.get("direction") in ("bullish", "neutral", "bearish") else "neutral"
    try:
        score = int(data.get("score"))
    except (TypeError, ValueError):
        score = 50
    return {
        "direction": direction,
        "score": max(0, min(100, score)),
        "risk": data.get("risk") if data.get("risk") in ("low", "mid", "high") else "mid",
        "action": str(data.get("action") or "观望")[:40],
        "summary": str(data.get("summary") or "").strip()[:200],
        "cards": data.get("cards") if isinstance(data.get("cards"), list) else [],
    }


def _user_prompt(context: dict) -> str:
    stock = context.get("stock") or {}
    ind = context.get("indicators") or {}
    hits = context.get("algorithmHits") or []
    lines = [
        f"股票：{stock.get('name') or ''} {stock.get('symbol')}，行业 {stock.get('industry') or '-'}，现价 {stock.get('price')}，涨跌 {stock.get('changePercent')}%，换手 {stock.get('turnover')}%",
        f"指标：MA5/10/20={ind.get('ma5')}/{ind.get('ma10')}/{ind.get('ma20')}，RSI6={ind.get('rsi6')}，乖离={ind.get('bias5')}，MACD柱={ind.get('hist')}，多头排列={ind.get('bullAlign')}",
        "算法：",
    ]
    for h in hits:
        flag = "命中" if h.get("pass") else "未命中"
        lines.append(f"- {h.get('short') or h.get('algorithmCode')} {flag}：{h.get('reason') or ''}")
    lines.append("请按系统要求返回完整中文 JSON，禁止只返回 {}。")
    return "\n".join(lines)[:8000]
