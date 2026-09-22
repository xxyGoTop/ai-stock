from __future__ import annotations

from collections.abc import Iterator
from typing import Any

from llm.router import run_profile
from quant.algorithms.registry import REGISTRY, run_all
from quant.indicators import compute_snapshot
from quant.screening.engine import build_rps_maps, rps_of
from quant.screening.market import fetch_klines, fetch_market_returns, fetch_quote
from quant.sector_context import build_sector_context, sector_lines


def analyze_stock(
    symbol: str,
    profile_code: str | None = None,
    model_code: str | None = None,
    extras: dict | None = None,
) -> dict:
    result = None
    for ev in analyze_stock_events(symbol, profile_code, model_code, extras):
        if ev.get("event") == "result":
            result = ev.get("data")
    if not result:
        raise RuntimeError("分析未返回结果")
    return result


def analyze_stock_events(
    symbol: str,
    profile_code: str | None = None,
    model_code: str | None = None,
    extras: dict | None = None,
) -> Iterator[dict[str, Any]]:
    """逐步产出 progress / result，供流式进度展示。"""
    symbol = "".join(ch for ch in str(symbol) if ch.isdigit()).zfill(6)
    extras = extras or {}

    def progress(step: str, title: str, status: str = "running", **extra: Any) -> dict:
        return {"event": "progress", "step": step, "title": title, "status": status, **extra}

    yield progress("klines", "拉取 K 线与行情", "running")
    klines = fetch_klines(symbol, 260)
    if len(klines) < 60:
        raise RuntimeError("K线不足，无法分析")
    stock = _resolve_stock(symbol, extras.get("quote"), klines)
    yield progress(
        "klines",
        "拉取 K 线与行情",
        "done",
        summary=f"{stock.get('name') or symbol} · {stock.get('price')} · {float(stock.get('changePercent') or 0):+.2f}%",
    )

    yield progress("rps", "计算市场相对强弱", "running")
    try:
        returns = fetch_market_returns(pages=6)
    except Exception:
        returns = []
    rps_maps = build_rps_maps(returns)
    indicators = compute_snapshot(klines)
    yield progress("rps", "计算市场相对强弱", "done")

    yield progress("algo", "运行五套选股算法", "running")
    ctx = {
        "stock": stock,
        "klines": klines,
        "rps": rps_of(rps_maps, symbol),
        "indicators": indicators,
        "indexCanBuy": True,
        "inFirstPage": False,
    }
    hits = run_all(ctx)
    hit_n = sum(1 for h in hits if h.get("pass"))
    yield progress("algo", "运行五套选股算法", "done", summary=f"命中 {hit_n}/{len(hits)}")

    yield progress("sector", "板块与涨跌归因", "running")
    try:
        sector = build_sector_context(
            stock,
            news_titles=list(extras.get("newsTitles") or []),
            notice_titles=list(extras.get("noticeTitles") or []),
            hot_topics=list(extras.get("hotTopics") or []),
        )
    except Exception:
        sector = {
            "industry": stock.get("industry") or "",
            "concepts": stock.get("concepts") or [],
            "industryBoard": None,
            "conceptBoards": [],
            "matchedTopics": [],
            "newsTitles": list(extras.get("newsTitles") or []),
            "noticeTitles": list(extras.get("noticeTitles") or []),
            "attribution": {
                "primary": "unknown",
                "primaryLabel": "信号不足",
                "move": "震荡",
                "stockChangePercent": float(stock.get("changePercent") or 0),
                "drivers": [],
                "explanation": "板块归因暂不可用。",
            },
        }
    attr_label = ((sector.get("attribution") or {}).get("primaryLabel")) or ""
    yield progress("sector", "板块与涨跌归因", "done", summary=attr_label or (stock.get("industry") or ""))

    pack = {
        "stock": {
            k: stock.get(k)
            for k in (
                "symbol",
                "name",
                "market",
                "price",
                "changePercent",
                "turnover",
                "volumeRatio",
                "industry",
                "concepts",
                "region",
                "mainNetInflow",
                "mainNetInflowPct",
            )
        },
        "indicators": {
            k: indicators.get(k)
            for k in (
                "ma5",
                "ma10",
                "ma20",
                "bias5",
                "rsi6",
                "bullAlign",
                "aboveMa5",
                "ma5Rising",
                "macdGolden",
                "hist",
            )
        },
        "rps": ctx["rps"],
        "algorithmHits": [
            {
                "algorithmCode": h["algorithmCode"],
                "short": REGISTRY[h["algorithmCode"]].short if h["algorithmCode"] in REGISTRY else h["algorithmCode"],
                "name": REGISTRY[h["algorithmCode"]].name if h["algorithmCode"] in REGISTRY else h["algorithmCode"],
                "pass": h["pass"],
                "score": h["score"],
                "reason": h["reason"],
            }
            for h in hits
        ],
        "sector": sector,
        "sector_lines": sector_lines(sector),
        "attribution": sector.get("attribution") or {},
    }

    yield progress("llm", "大模型解读中", "running", summary="火山方舟 / 规则回退")
    routed = run_profile(pack, profile_code, model_code)
    if not routed.get("attribution"):
        routed["attribution"] = sector.get("attribution")
    if not any((c or {}).get("cardType") == "sector" for c in (routed.get("cards") or [])):
        sector_card = _sector_card(sector)
        if sector_card:
            routed["cards"] = [sector_card, *(routed.get("cards") or [])]
    used = " / ".join(routed.get("usedModels") or ["quant-rules"])
    yield progress("llm", "大模型解读中", "done", summary=used)

    result = {
        "type": "stock_analysis",
        "symbol": symbol,
        "name": stock.get("name"),
        "algorithmHits": pack["algorithmHits"],
        "sector": sector,
        **routed,
    }
    yield {"event": "result", "data": result}


def _sector_card(sector: dict) -> dict | None:
    attr = sector.get("attribution") or {}
    items = []
    ib = sector.get("industryBoard") or {}
    if ib:
        items.append(
            {
                "name": "行业板块",
                "value": f"{ib.get('name')} {float(ib.get('changePercent') or 0):+.2f}%",
            }
        )
    if sector.get("concepts"):
        items.append({"name": "概念", "value": "、".join((sector.get("concepts") or [])[:4])})
    if attr.get("primaryLabel"):
        items.append({"name": "涨跌主因", "value": attr["primaryLabel"]})
    for d in (attr.get("drivers") or [])[:2]:
        items.append({"name": d.get("label") or "因素", "value": d.get("detail") or ""})
    if not items:
        return None
    return {
        "cardType": "sector",
        "title": "板块与涨跌归因",
        "score": 60,
        "items": items,
    }


def _resolve_stock(symbol: str, injected: dict | None, klines: list[dict]) -> dict:
    """优先用 Go 注入的行情，再本地拉取，最后用 K 线末根补价。"""
    base = _normalize_quote(injected) if isinstance(injected, dict) else None
    local = fetch_quote(symbol)
    if local and local.get("price"):
        if base and base.get("price"):
            for k in ("industry", "region", "concepts", "mainNetInflow", "mainNetInflowPct"):
                if (not base.get(k)) and local.get(k):
                    base[k] = local[k]
            return base
        return local
    if base and base.get("price"):
        return base

    last = klines[-1] if klines else {}
    prev = klines[-2] if len(klines) > 1 else {}
    price = float(last.get("close") or 0)
    prev_close = float(prev.get("close") or 0)
    chg_pct = float(last.get("changePercent") or 0)
    if chg_pct == 0 and prev_close > 0 and price > 0:
        chg_pct = (price - prev_close) / prev_close * 100
    return {
        "symbol": symbol,
        "name": (base or {}).get("name") or symbol,
        "market": (base or {}).get("market") or ("SH" if symbol.startswith(("6", "9", "5")) else "SZ"),
        "price": price,
        "changePercent": chg_pct,
        "turnover": float(last.get("turnover") or 0),
        "volumeRatio": 0,
        "industry": (base or {}).get("industry") or "",
        "concepts": list((base or {}).get("concepts") or []),
        "region": (base or {}).get("region") or "",
    }


def _normalize_quote(raw: dict) -> dict | None:
    if not raw:
        return None
    price = float(raw.get("price") or 0)
    if price <= 0:
        return None
    concepts = raw.get("concepts") or []
    if isinstance(concepts, str):
        concepts = [x.strip() for x in concepts.split(",") if x.strip()]
    return {
        "symbol": str(raw.get("symbol") or "").zfill(6),
        "name": str(raw.get("name") or "").strip(),
        "market": str(raw.get("market") or ""),
        "price": price,
        "changePercent": float(raw.get("changePercent") or 0),
        "change": float(raw.get("change") or 0),
        "turnover": float(raw.get("turnover") or 0),
        "volumeRatio": float(raw.get("volumeRatio") or 0),
        "volume": float(raw.get("volume") or 0),
        "amount": float(raw.get("amount") or 0),
        "industry": str(raw.get("industry") or "").strip(),
        "region": str(raw.get("region") or "").strip(),
        "concepts": list(concepts)[:8],
        "mainNetInflow": float(raw.get("mainNetInflow") or 0),
        "mainNetInflowPct": float(raw.get("mainNetInflowPct") or 0),
        "superNetInflow": float(raw.get("superNetInflow") or 0),
        "bigNetInflow": float(raw.get("bigNetInflow") or 0),
        "fundKnown": True,
    }


def _quote(symbol: str) -> dict:
    hit = fetch_quote(symbol)
    if hit and hit.get("price"):
        return hit
    klines = fetch_klines(symbol, 5)
    return _resolve_stock(symbol, None, klines)
