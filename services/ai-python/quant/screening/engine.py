from __future__ import annotations

from ..algorithms.registry import DEFAULT_PROFILE, REGISTRY, run_all
from ..indicators import compute_snapshot
from ..tao import extras, mix_rps, rank_rps
from . import market


def build_rps_maps(returns: list[dict]) -> dict[str, dict]:
    r5 = rank_rps({r["code"]: r.get("change5") for r in returns})
    r60 = rank_rps({r["code"]: r.get("change60") for r in returns})
    rytd = rank_rps({r["code"]: r.get("changeYtd") for r in returns})
    return {
        "rps20": mix_rps(r5, r60, 0.4),
        "rps50": r60,
        "rps120": mix_rps(r60, rytd, 0.5),
        "rps250": rytd,
    }


def rps_of(maps: dict, code: str) -> dict:
    return {
        "rps20": maps["rps20"].get(code, 0),
        "rps50": maps["rps50"].get(code, 0),
        "rps120": maps["rps120"].get(code, 0),
        "rps250": maps["rps250"].get(code, 0),
    }


def run_screening(payload: dict | None = None) -> dict:
    payload = payload or {}
    detail = min(max(int(payload.get("detail") or 80), 20), 220)
    limit = min(max(int(payload.get("limit") or 30), 5), 60)
    enabled = payload.get("algorithms") or [a["code"] for a in DEFAULT_PROFILE["algorithms"]]
    weights = {a["code"]: float(a.get("weight") or 1) for a in DEFAULT_PROFILE["algorithms"]}
    for item in payload.get("weights") or []:
        if item.get("code"):
            weights[item["code"]] = float(item.get("weight") or 1)

    symbols = [str(x).zfill(6) for x in (payload.get("symbols") or []) if str(x).strip()]
    industry = str(payload.get("industry") or "").strip()
    board = str(payload.get("board") or "").strip()

    if symbols:
        pool = market.fetch_quotes_by_symbols(symbols)
        detail = min(max(len(pool), 20), 220)
    else:
        pool = market.fetch_active_stocks(pages=max(2, detail // 80 + 2), fid="f6")
        pool = [s for s in pool if "ST" not in (s.get("name") or "").upper()]
        if industry:
            keyed = industry
            pool = [
                s
                for s in pool
                if keyed in (s.get("industry") or "") or keyed in (s.get("name") or "")
            ]
        pool.sort(key=lambda s: s.get("amount") or 0, reverse=True)

    pool = [s for s in pool if "ST" not in (s.get("name") or "").upper()]
    scan = pool[:detail]

    gainers = sorted(pool, key=lambda s: s.get("changePercent") or 0, reverse=True)[:29]
    first_page = {s["symbol"]: i + 1 for i, s in enumerate(gainers)}

    returns = market.fetch_market_returns(pages=20)
    rps_maps = build_rps_maps(returns)
    klines_map = market.fetch_klines_many([s["symbol"] for s in scan], limit=260)
    for n, key in ((20, "rps20"), (50, "rps50"), (120, "rps120"), (250, "rps250")):
        ranked = rank_rps({code: extras(kl, n) for code, kl in klines_map.items()})
        for code, value in ranked.items():
            if not rps_maps[key].get(code):
                rps_maps[key][code] = value

    picks = []
    strategy_count = {c: 0 for c in enabled}
    for stock in scan:
        bars = klines_map.get(stock["symbol"])
        if not bars:
            continue
        ctx = {
            "stock": stock,
            "klines": bars,
            "rps": rps_of(rps_maps, stock["symbol"]),
            "indicators": compute_snapshot(bars),
            "indexCanBuy": True,
            "inFirstPage": stock["symbol"] in first_page,
            "observeRank": first_page.get(stock["symbol"]),
        }
        hits = [h for h in run_all(ctx, enabled) if h["pass"]]
        if not hits:
            continue
        if (stock.get("changePercent") or 0) < -7:
            continue
        for h in hits:
            strategy_count[h["algorithmCode"]] = strategy_count.get(h["algorithmCode"], 0) + 1
        score = sum(h["score"] * weights.get(h["algorithmCode"], 1) for h in hits)
        primary = max(hits, key=lambda h: REGISTRY[h["algorithmCode"]].base_score)
        picks.append(
            {
                "symbol": stock["symbol"],
                "name": stock["name"],
                "market": stock["market"],
                "industry": stock.get("industry") or board or "",
                "price": stock.get("price"),
                "changePercent": stock.get("changePercent"),
                "score": round(score, 2),
                "primary": primary["algorithmCode"],
                "primaryName": REGISTRY[primary["algorithmCode"]].name,
                "strategies": [
                    {
                        "code": h["algorithmCode"],
                        "name": REGISTRY[h["algorithmCode"]].name,
                        "short": REGISTRY[h["algorithmCode"]].short,
                        "score": h["score"],
                        "reason": h["reason"],
                    }
                    for h in sorted(hits, key=lambda x: -REGISTRY[x["algorithmCode"]].base_score)
                ],
                "inFirstPage": stock["symbol"] in first_page,
                "rps": ctx["rps"],
            }
        )

    picks.sort(key=lambda x: x["score"], reverse=True)
    return {
        "profileCode": "screening_default",
        "board": board or None,
        "scanned": len(klines_map),
        "qualified": len(picks),
        "strategyCount": strategy_count,
        "picks": picks[:limit],
    }
