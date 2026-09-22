from __future__ import annotations

import json

from prompts.loader import render_messages

from .config import get_model, model_callable
from .providers import complete_text

NL_ORDER = ["doubao-seed-2-1-lite", "doubao-seed-2-1-pro", "deepseek-v4-flash"]


def parse_screening_nl(
    query: str,
    boards: list[dict] | None = None,
    model_code: str | None = None,
) -> dict:
    """把自然语言选股需求解析成算法 + 板块提示。"""
    question = (query or "").strip()
    board_rows = []
    for b in boards or []:
        name = str(b.get("name") or "").strip()
        code = str(b.get("code") or "").strip()
        if not name:
            continue
        board_rows.append({"name": name, "code": code, "changePercent": b.get("changePercent")})

    fallback = {
        "summary": "按默认五算法全市场扫描。",
        "boardHints": [],
        "algorithms": ["year_high", "deep_rebound", "forward_train", "daily_observe", "ma5_align"],
        "limit": 30,
        "detail": 80,
        "filters": "",
        "modelCode": "quant-rules",
    }
    if not question:
        return fallback

    model = _resolve(model_code)
    if not model:
        # 无模型时：用关键词粗映射板块提示
        fallback["summary"] = f"模型未就绪，按关键词理解：{question}"
        fallback["boardHints"] = _keyword_board_hints(question, board_rows)
        return fallback

    context = {
        "query": question,
        "context_json": json.dumps(
            {"candidateBoards": board_rows[:100]},
            ensure_ascii=False,
        )[:7000],
    }
    messages = render_messages("screening_nl", context)
    text = complete_text(model, messages, temperature=0.2, json_mode=True)
    data = _parse_nl_json(text)
    data["modelCode"] = model["code"]
    if not data.get("boardHints"):
        data["boardHints"] = _keyword_board_hints(question, board_rows)
    return data


def _resolve(model_code: str | None) -> dict | None:
    if model_code and model_code not in ("auto", ""):
        found = get_model(model_code)
        if found and model_callable(found):
            return found
    for code in NL_ORDER:
        found = get_model(code)
        if found and model_callable(found):
            return found
    return None


def _parse_nl_json(text: str) -> dict:
    text = (text or "").strip()
    if text.startswith("```"):
        text = text.strip("`")
        if text.lower().startswith("json"):
            text = text[4:].lstrip()
    if not text.startswith("{"):
        start, end = text.find("{"), text.rfind("}")
        if start >= 0 and end > start:
            text = text[start : end + 1]
    try:
        raw = json.loads(text) if text else {}
    except json.JSONDecodeError:
        raw = {}
    if not isinstance(raw, dict):
        raw = {}

    allowed = {"year_high", "deep_rebound", "forward_train", "daily_observe", "ma5_align"}
    algos = [str(a) for a in (raw.get("algorithms") or []) if str(a) in allowed]
    if not algos:
        algos = list(allowed)

    hints = []
    for h in raw.get("boardHints") or []:
        name = str(h).strip()
        if name and name not in hints:
            hints.append(name)

    try:
        limit = int(raw.get("limit") or 30)
    except (TypeError, ValueError):
        limit = 30
    try:
        detail = int(raw.get("detail") or 80)
    except (TypeError, ValueError):
        detail = 80

    return {
        "summary": str(raw.get("summary") or "").strip()[:200] or "已理解选股需求。",
        "boardHints": hints[:5],
        "algorithms": algos,
        "limit": max(10, min(60, limit)),
        "detail": max(20, min(220, detail)),
        "filters": str(raw.get("filters") or "").strip()[:120],
    }


def _keyword_board_hints(query: str, boards: list[dict]) -> list[str]:
    """无模型时的粗匹配：关键词落在板块名里。"""
    q = query
    seeds = []
    for token in (
        "医药", "制药", "中药", "医疗", "创新药", "半导体", "芯片", "新能源", "光伏", "锂电",
        "白酒", "银行", "券商", "证券", "军工", "地产", "有色", "汽车", "AI", "人工智能", "算力",
        "消费", "软件", "游戏", "传媒", "农业", "煤炭", "石油", "电力", "保险",
    ):
        if token.lower() in q.lower() or token in q:
            seeds.append(token)
    if not seeds:
        return []
    hits = []
    for b in boards:
        name = str(b.get("name") or "")
        for s in seeds:
            if s in name or name in s:
                if name and name not in hits:
                    hits.append(name)
                break
    return hits[:5]
