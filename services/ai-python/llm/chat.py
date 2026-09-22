from __future__ import annotations

from prompts.loader import get_agent

from .config import get_model, model_callable
from .providers import complete_text

CHAT_ORDER = ["doubao-seed-2-1-lite", "doubao-seed-2-1-pro", "deepseek-v4-flash", "quant-rules"]


def chat_reply(
    message: str,
    history: list[dict] | None = None,
    model_code: str | None = None,
    memory_context: str | None = None,
) -> dict:
    question = (message or "").strip()
    if not question:
        return {"reply": "说一下你想看的行情、指标，或股票代码。", "modelCode": "quant-rules"}

    model = _resolve(model_code)
    if not model or model["code"] == "quant-rules":
        return {
            "reply": "当前没有可用的对话模型。配好火山方舟密钥后，可在输入框左侧切换豆包或 DeepSeek V4。你也可以直接说「今天行情」「帮我选股」「分析茅台」。",
            "modelCode": "quant-rules",
        }

    agent = get_agent("companion_chat")
    system = agent.system
    mem = (memory_context or "").strip()
    if mem:
        system = (
            system
            + "\n\n## Relevant Memory\n"
            + mem
            + "\n（仅作背景参考；不要编造记忆中没有的事实。用户未明确设置的偏好不要永久推断。）"
        )
    messages = [{"role": "system", "content": system}]
    for turn in history or []:
        role = turn.get("role")
        content = str(turn.get("content") or "").strip()
        if role not in ("user", "assistant") or not content:
            continue
        messages.append({"role": role, "content": content[:4000]})
    if not messages or messages[-1].get("role") != "user" or messages[-1].get("content") != question:
        messages.append({"role": "user", "content": question[:4000]})

    text = complete_text(model, messages, temperature=0.4)
    if not text:
        raise RuntimeError(f"{model['code']} 没有返回内容")
    return {"reply": text, "modelCode": model["code"]}


def _resolve(model_code: str | None) -> dict | None:
    if model_code and model_code not in ("auto", ""):
        found = get_model(model_code)
        if found and model_callable(found):
            return found
    for code in CHAT_ORDER:
        found = get_model(code)
        if found and model_callable(found):
            return found
    return get_model("quant-rules")
