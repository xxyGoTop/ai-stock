from __future__ import annotations

from llm.router import run_profile
from quant.algorithms.registry import REGISTRY, run_all
from quant.indicators import compute_snapshot
from quant.screening.engine import build_rps_maps, rps_of
from quant.screening.market import fetch_klines, fetch_market_returns, fetch_quote


def analyze_stock(symbol: str, profile_code: str | None = None, model_code: str | None = None) -> dict:
    symbol = "".join(ch for ch in str(symbol) if ch.isdigit()).zfill(6)
    stock = _quote(symbol)
    klines = fetch_klines(symbol, 260)
    if len(klines) < 60:
        raise RuntimeError("K线不足，无法分析")
    try:
        returns = fetch_market_returns(pages=12)
    except Exception:
        returns = []
    rps_maps = build_rps_maps(returns)
    indicators = compute_snapshot(klines)
    ctx = {
        "stock": stock,
        "klines": klines,
        "rps": rps_of(rps_maps, symbol),
        "indicators": indicators,
        "indexCanBuy": True,
        "inFirstPage": False,
    }
    hits = run_all(ctx)
    pack = {
        "stock": {k: stock.get(k) for k in ("symbol", "name", "market", "price", "changePercent", "turnover", "volumeRatio", "industry")},
        "indicators": {k: indicators.get(k) for k in ("ma5", "ma10", "ma20", "bias5", "rsi6", "bullAlign", "aboveMa5", "ma5Rising", "macdGolden", "hist")},
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
    }
    routed = run_profile(pack, profile_code, model_code)
    return {
        "type": "stock_analysis",
        "symbol": symbol,
        "name": stock.get("name"),
        "algorithmHits": pack["algorithmHits"],
        **routed,
    }


def _quote(symbol: str) -> dict:
    hit = fetch_quote(symbol)
    if hit and hit.get("price"):
        return hit
    klines = fetch_klines(symbol, 5)
    last = klines[-1] if klines else {}
    return {
        "symbol": symbol,
        "name": symbol,
        "market": "SH" if symbol.startswith(("6", "9", "5")) else "SZ",
        "price": last.get("close") or 0,
        "changePercent": last.get("changePercent") or 0,
        "turnover": last.get("turnover") or 0,
        "volumeRatio": 0,
        "industry": "",
    }
