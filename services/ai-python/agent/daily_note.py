from __future__ import annotations

import json
from datetime import datetime, timezone, timedelta
from pathlib import Path

from quant.algorithms.registry import REGISTRY, run_all
from quant.indicators import compute_snapshot
from quant.meta import build_expect, build_verdict, classify_capital, describe_fund, estimate_chips
from quant.screening.market import fetch_klines, fetch_quote
from quant.tao import count_bases

CST = timezone(timedelta(hours=8))
NOTE_DIR = Path(__file__).resolve().parents[1] / "data" / "daily_notes"

SHORT = {a.code: a.short for a in REGISTRY.values()}
NAME = {a.code: a.name for a in REGISTRY.values()}
ORDER = ["year_high", "deep_rebound", "forward_train", "daily_observe", "ma5_align"]


def daily_note(symbol: str, force: bool = False) -> dict:
    symbol = "".join(ch for ch in str(symbol) if ch.isdigit()).zfill(6)
    as_of = _trade_date()
    cached = None if force else _load(as_of, symbol)
    if cached and cached.get("verdict") and (cached.get("plan") or {}).get("position") is not None:
        cached["cached"] = True
        return cached
    note = _build(symbol, as_of)
    _save(as_of, symbol, note)
    note["cached"] = False
    return note


def _build(symbol: str, as_of: str) -> dict:
    stock = fetch_quote(symbol) or {}
    klines = fetch_klines(symbol, 260)
    if len(klines) < 60:
        raise RuntimeError("K线不足，无法生成每日笔记")
    as_of = str(klines[-1].get("date") or as_of)[:10]
    indicators = compute_snapshot(klines)
    bases = count_bases(klines)
    ctx = {
        "stock": {
            "symbol": symbol,
            "name": stock.get("name") or symbol,
            "market": stock.get("market") or ("SH" if symbol.startswith(("6", "9", "5")) else "SZ"),
            "price": stock.get("price") or klines[-1]["close"],
            "changePercent": stock.get("changePercent") or klines[-1].get("changePercent") or 0,
            "turnover": stock.get("turnover") or 0,
            "volumeRatio": stock.get("volumeRatio") or 0,
            "amplitude": stock.get("amplitude"),
            "circMV": stock.get("circMV"),
            "industry": stock.get("industry") or "",
            "region": stock.get("region") or "",
            "concepts": stock.get("concepts") or [],
            "mainNetInflow": stock.get("mainNetInflow"),
            "mainNetInflowPct": stock.get("mainNetInflowPct"),
            "superNetInflow": stock.get("superNetInflow"),
            "superNetInflowPct": stock.get("superNetInflowPct"),
            "bigNetInflow": stock.get("bigNetInflow"),
            "bigNetInflowPct": stock.get("bigNetInflowPct"),
            "limitUpStreak": indicators.get("limitUpStreak") or 0,
        },
        "klines": klines,
        "rps": {"rps20": 0, "rps50": 0, "rps120": 0, "rps250": 0},
        "indicators": indicators,
        "indexCanBuy": True,
        "inFirstPage": False,
    }
    hits = run_all(ctx)
    passed = [h for h in hits if h.get("pass")]
    passed.sort(key=lambda h: ORDER.index(h["algorithmCode"]) if h["algorithmCode"] in ORDER else 99)
    primary = passed[0]["algorithmCode"] if passed else ""
    score = round(sum(h.get("score") or 0 for h in passed), 1)
    stance = _base_stance(bases.get("baseCount") or 0)
    plan = _entry_plan(ctx["stock"]["price"], indicators, primary or "ma5_align")
    action = _action(score, plan, ctx["stock"]["changePercent"])
    plan.update(_trade_meta(score, action, plan, indicators, ctx["stock"]))
    reasons = [f"【{NAME.get(h['algorithmCode'], h['algorithmCode'])}】{h.get('reason')}" for h in passed] or [
        "今日五套算法均未命中，仅作观察。"
    ]
    risks = _risks(indicators, ctx["stock"], stance, plan, [h["algorithmCode"] for h in passed])
    fund = describe_fund(
        ctx["stock"].get("mainNetInflow"),
        ctx["stock"].get("mainNetInflowPct"),
        bool(stock.get("fundKnown")),
    )
    capital = classify_capital(ctx["stock"])
    chips = estimate_chips(klines)
    verdict = build_verdict(ctx["stock"], indicators, fund, chips, klines)
    expect = build_expect(ctx["stock"], fund)
    return {
        "type": "daily_note",
        "symbol": symbol,
        "name": ctx["stock"]["name"],
        "industry": ctx["stock"]["industry"] or "未知行业",
        "region": ctx["stock"]["region"],
        "concepts": ctx["stock"]["concepts"],
        "asOf": as_of,
        "price": ctx["stock"]["price"],
        "changePercent": ctx["stock"]["changePercent"],
        "score": score,
        "primary": primary,
        "primaryName": NAME.get(primary, "") or "未命中五套算法",
        "strategies": [
            {
                "code": h["algorithmCode"],
                "short": SHORT.get(h["algorithmCode"], h["algorithmCode"]),
                "name": NAME.get(h["algorithmCode"], h["algorithmCode"]),
                "pass": bool(h.get("pass")),
                "score": h.get("score") or 0,
                "reason": h.get("reason") or "",
            }
            for h in hits
        ],
        "baseLabel": stance["label"],
        "baseLevel": stance["level"],
        "baseCount": bases.get("baseCount") or 0,
        "action": action["action"],
        "actionLevel": action["level"],
        "actionNote": action["note"],
        "plan": plan,
        "reasons": reasons,
        "risks": risks,
        "fund": fund,
        "capital": capital,
        "chips": chips,
        "expect": expect,
        "verdict": verdict,
        "indicators": {
            "ma5": indicators.get("ma5"),
            "ma10": indicators.get("ma10"),
            "ma20": indicators.get("ma20"),
            "bias5": indicators.get("bias5"),
            "rsi6": indicators.get("rsi6"),
            "rsi14": indicators.get("rsi14"),
            "hist": indicators.get("hist"),
            "macdGolden": indicators.get("macdGolden"),
            "bullAlign": indicators.get("bullAlign"),
        },
    }


def _entry_plan(price: float, ind: dict, primary: str) -> dict:
    price = float(price or 0)
    ma5 = float(ind.get("ma5") or price)
    ma10 = float(ind.get("ma10") or ma5)
    ma20 = float(ind.get("ma20") or ma10)
    if primary == "forward_train":
        entry, low, high, stop, extra = "10日线下低吸", ma10 * 0.985, ma10 * 1.004, ma20 * 0.985, f"回到10日线({_r(ma10)})下方，跌破20日线离场"
        t1, t2 = 0.06, 0.12
    elif primary == "year_high":
        entry, low, high, stop, extra = "年高右侧·回踩5日线", ma5, max(ma5 * 1.01, min(price, ma5 * 1.03)), max(ma10 * 0.985, ma5 * 0.93), f"回踩站稳5日线({_r(ma5)})再进"
        t1, t2 = 0.06, 0.12
    elif primary == "deep_rebound":
        entry, low, high, stop, extra = "回升确认·贴10日线", min(ma10, ma5) * 0.995, max(ma10, ma5) * 1.005, ma20 * 0.97, f"10日线({_r(ma10)})附近分批"
        t1, t2 = 0.06, 0.12
    elif primary == "daily_observe":
        entry, low, high, stop, extra = "观察池·等回踩", min(ma10, ma5) * 0.995, max(ma10, ma5) * 1.005, ma20 * 0.97, "等回踩到区间再分批，不追"
        t1, t2 = 0.06, 0.12
    else:
        entry, low, high, stop, extra = "五日线短线", ma5 * 0.997, ma5 * 1.008, ma5 * 0.985, "硬止损破了当天走"
        t1, t2 = 0.03, 0.06
    plan = _normalize(price, low, high, stop, t1, t2)
    plan.update(
        {
            "entryType": entry,
            "note": f"{_stance_note(plan, price)}。涨到 {plan['target1']:.2f} 先减半，余仓看到 {plan['target2']:.2f}。{extra}",
        }
    )
    return plan


def _normalize(price, zone_low, zone_high, structural, t1, t2):
    low, high = min(zone_low, zone_high), max(zone_low, zone_high)
    if not (low > 0 and high > 0):
        low, high = price * 0.99, price * 1.01
    if (high - low) / low > 0.03:
        low = high / 1.03
    stop = min(structural or low * 0.985, low * 0.985)
    stop = max(stop, high * 0.93)
    target1 = high * (1 + t1)
    target2 = max(high * (1 + t2), target1 * 1.04)
    stance = "in"
    if price > high * 1.01:
        stance = "above"
    elif price < low * 0.99:
        stance = "below"
    return {
        "buyLow": _r(low),
        "buyHigh": _r(high),
        "stop": _r(stop),
        "target1": _r(target1),
        "target2": _r(target2),
        "stance": stance,
    }


def _trade_meta(score: float, action: dict, plan: dict, ind: dict, stock: dict) -> dict:
    level = action.get("level") or "watch"
    if level == "buy":
        pos, risk, direction = (0.2 if score >= 40 else 0.15), "medium", "buy"
    elif level == "wait":
        pos, risk, direction = 0.1, "medium", "watch"
    else:
        pos, risk, direction = 0.05, "high", "watch"
    price = float(stock.get("price") or plan.get("buyHigh") or 0)
    qty = 0
    if price > 0:
        qty = int(1_000_000 * pos / price / 100) * 100
        if pos >= 0.1:
            qty = max(100, qty)
    invalid = [f"跌破止损 {plan['stop']:.2f}"]
    if ind.get("ma20"):
        invalid.append(f"跌破20日线 {float(ind['ma20']):.2f}")
    if plan.get("stance") == "above":
        invalid.append("现价未回踩买入区间，追高失效")
    elif plan.get("stance") == "below":
        invalid.append("现价未站回买入区间")
    if (ind.get("rsi14") or 0) > 70:
        invalid.append("RSI14 维持超买且拐头向下")
    return {
        "direction": direction,
        "position": pos,
        "suggestedQty": qty,
        "riskLevel": risk,
        "invalidConditions": invalid[:4],
    }


def _stance_note(plan: dict, price: float) -> str:
    if plan["stance"] == "above":
        return f"现价 {_r(price)} 已高出买入上限，不追，挂单等回踩"
    if plan["stance"] == "below":
        return f"现价 {_r(price)} 低于买入下限，等站回区间再动手"
    return f"现价 {_r(price)} 正在买入区间内，可按计划分批"


def _action(score: float, plan: dict, change: float) -> dict:
    if change <= -7:
        return {"action": "仅观察", "level": "watch", "note": "当日大跌，先回避"}
    if score < 24:
        return {"action": "仅观察", "level": "watch", "note": "评分偏低，信号不够密集"}
    if plan["stance"] == "in":
        return {"action": "可买入", "level": "buy", "note": "现价在买入区间内"}
    if plan["stance"] == "above":
        return {"action": "等回踩", "level": "wait", "note": f"等回到 {plan['buyHigh']} 以下"}
    return {"action": "等站回", "level": "wait", "note": f"等站回 {plan['buyLow']} 之上"}


def _risks(ind: dict, stock: dict, stance: dict, plan: dict, hit_keys: list[str]) -> list[str]:
    out = []
    if stance["level"] in ("caution", "risky"):
        out.append(f"{stance['label']}，第3个基底起成功率下降，仓位要降")
    if plan["stance"] == "above":
        out.append(f"现价高于买入区间，直接买等于追高")
    bias = ind.get("bias5")
    if bias is not None and bias > 8:
        out.append(f"已高出五日线 {bias:.1f}%，短期透支")
    if (stock.get("turnover") or 0) >= 20:
        out.append(f"换手 {stock['turnover']:.1f}%，波动会很大")
    if hit_keys == ["ma5_align"]:
        out.append("仅五日线共振，只能做短线，止损必须严格")
    return out


def _base_stance(count: int) -> dict:
    if not count:
        return {"label": "未形成基底", "level": "unknown"}
    if count == 1:
        return {"label": "第1个基底", "level": "good"}
    if count == 2:
        return {"label": "第2个基底", "level": "good"}
    if count == 3:
        return {"label": "第3个基底", "level": "caution"}
    return {"label": f"第{count}个基底", "level": "risky"}


def _trade_date() -> str:
    return datetime.now(CST).strftime("%Y-%m-%d")


def _path(as_of: str, symbol: str) -> Path:
    return NOTE_DIR / as_of / f"{symbol}.json"


def _load(as_of: str, symbol: str) -> dict | None:
    p = _path(as_of, symbol)
    if not p.exists():
        return None
    try:
        return json.loads(p.read_text(encoding="utf-8"))
    except Exception:
        return None


def _save(as_of: str, symbol: str, note: dict) -> None:
    p = _path(as_of, symbol)
    p.parent.mkdir(parents=True, exist_ok=True)
    payload = {k: v for k, v in note.items() if k != "cached"}
    p.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")


def _r(n) -> float:
    try:
        return round(float(n), 2)
    except (TypeError, ValueError):
        return 0.0
