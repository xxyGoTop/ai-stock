from __future__ import annotations

import json
import urllib.error
import urllib.request

from .config import get_provider, resolve_key
from .quant_analyst import analyze_quant


SYSTEM = """你是A股投研助手。只根据给定的量化上下文做解释，不要自己编造指标数字。
必须返回 JSON，字段：
direction: bullish|neutral|bearish
score: 0-100 整数
risk: low|mid|high
action: 简短建议
summary: 不超过120字中文
cards: [{cardType,title,score,items:[{name,value}]}]
cardType 只能是 technical/algorithm/risk/capital。
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
        "response_format": {"type": "json_object"},
        "messages": [
            {"role": "system", "content": SYSTEM},
            {"role": "user", "content": json.dumps(context, ensure_ascii=False)[:8000]},
        ],
    }
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
        raise RuntimeError(f"{model['code']} HTTP {exc.code}") from exc
    text = (((body.get("choices") or [{}])[0].get("message") or {}).get("content")) or "{}"
    parsed = _parse_json(text)
    parsed["modelCode"] = model["code"]
    return parsed


def _parse_json(text: str) -> dict:
    text = text.strip()
    if text.startswith("```"):
        text = text.strip("`")
        if text.startswith("json"):
            text = text[4:]
    data = json.loads(text)
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
        "summary": str(data.get("summary") or "")[:200],
        "cards": data.get("cards") if isinstance(data.get("cards"), list) else [],
    }
