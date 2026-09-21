from __future__ import annotations

import json
import urllib.error
import urllib.request

from prompts.loader import render_messages

from .config import effective_model_name, get_provider, resolve_key
from .quant_analyst import analyze_quant
from .quota import QuotaError, is_quota_error


def call_model(model: dict, context: dict, temperature: float = 0.2, agent_code: str | None = None) -> dict:
    provider = get_provider(model["providerCode"]) or {}
    if provider.get("kind") == "local" or model["code"] == "quant-rules":
        return analyze_quant(context)

    messages = render_messages(agent_code, context)
    text = complete_text(
        model,
        messages,
        temperature=temperature,
        json_mode=bool(model.get("jsonMode", True)),
    )
    parsed = _parse_json(text)
    parsed["modelCode"] = model["code"]
    return parsed


def complete_text(
    model: dict,
    messages: list[dict],
    temperature: float = 0.4,
    json_mode: bool = False,
) -> str:
    provider = get_provider(model["providerCode"]) or {}
    if provider.get("kind") == "local" or model["code"] == "quant-rules":
        raise RuntimeError("本地规则不支持自由对话")
    key = resolve_key(provider.get("apiKeyRef") or "")
    if not key:
        raise RuntimeError(f"{model['code']} 未配置密钥")
    payload = {
        "model": effective_model_name(model),
        "temperature": temperature,
        "max_tokens": min(int(model.get("maxTokens") or 2048), 4096),
        "messages": messages,
    }
    if json_mode:
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
    return str((((body.get("choices") or [{}])[0].get("message") or {}).get("content")) or "").strip()


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
