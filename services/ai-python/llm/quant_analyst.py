from __future__ import annotations


def analyze_quant(context: dict) -> dict:
    stock = context.get("stock") or {}
    ind = context.get("indicators") or {}
    hits = [h for h in context.get("algorithmHits") or [] if h.get("pass")]
    change = stock.get("changePercent") or 0
    bias5 = ind.get("bias5")
    rsi6 = ind.get("rsi6")
    score = 50
    if hits:
        score += min(30, len(hits) * 8)
        score += min(12, sum(h.get("score") or 0 for h in hits) / 20)
    if ind.get("bullAlign") and ind.get("aboveMa5"):
        score += 8
    if bias5 is not None and bias5 > 8:
        score -= 10
    if rsi6 is not None and rsi6 > 80:
        score -= 6
    if change < -5:
        score -= 8
    score = max(15, min(92, round(score)))

    if score >= 68 and hits:
        direction = "bullish"
        action = "观察回踩买入"
    elif score <= 42:
        direction = "bearish"
        action = "回避或减仓"
    else:
        direction = "neutral"
        action = "观望"

    risk = "high" if (bias5 is not None and bias5 > 8) or change <= -7 else "mid" if score < 55 else "low"
    names = "、".join(h.get("short") or h.get("name") or h.get("algorithmCode") for h in hits) or "未命中五套算法"
    attr = context.get("attribution") or (context.get("sector") or {}).get("attribution") or {}
    attr_bit = f"涨跌主因偏{attr.get('primaryLabel')}。" if attr.get("primaryLabel") else ""
    summary = (
        f"{stock.get('name') or stock.get('symbol')} 现价 {stock.get('price')}，"
        f"涨跌 {change:.2f}%。{attr_bit}"
        f"量化命中：{names}。"
        f"五日线{'向上且多头' if ind.get('bullAlign') and ind.get('ma5Rising') else '尚未完整共振'}，"
        f"建议{action}。"
    )
    cards = [
        {
            "cardType": "technical",
            "title": "技术面",
            "score": score,
            "items": [
                {"name": "方向", "value": {"bullish": "偏多", "bearish": "偏空", "neutral": "中性"}[direction]},
                {"name": "MA", "value": "多头排列" if ind.get("bullAlign") else "未多头"},
                {"name": "RSI6", "value": f"{rsi6:.1f}" if rsi6 is not None else "-"},
                {"name": "乖离MA5", "value": f"{bias5:.1f}%" if bias5 is not None else "-"},
            ],
        },
        {
            "cardType": "algorithm",
            "title": "算法命中",
            "score": score,
            "items": [{"name": h.get("short") or h.get("name") or h.get("algorithmCode"), "value": h.get("reason") or "命中"} for h in hits]
            or [{"name": "算法", "value": "今日五套算法均未命中"}],
        },
        {
            "cardType": "risk",
            "title": "风险",
            "score": 40 if risk == "high" else 60 if risk == "mid" else 75,
            "items": [
                {"name": "等级", "value": {"high": "偏高", "mid": "中性", "low": "可控"}[risk]},
                {"name": "动作", "value": action},
            ],
        },
    ]
    sector = context.get("sector") or {}
    ib = sector.get("industryBoard") or {}
    sector_items = []
    if ib:
        sector_items.append(
            {"name": "行业板块", "value": f"{ib.get('name')} {float(ib.get('changePercent') or 0):+.2f}%"}
        )
    if stock.get("industry"):
        sector_items.append({"name": "行业", "value": str(stock.get("industry"))})
    if attr.get("primaryLabel"):
        sector_items.append({"name": "涨跌主因", "value": attr["primaryLabel"]})
    for d in (attr.get("drivers") or [])[:2]:
        sector_items.append({"name": d.get("label") or "因素", "value": (d.get("detail") or "")[:80]})
    if sector_items:
        cards.insert(
            0,
            {
                "cardType": "sector",
                "title": "板块与涨跌归因",
                "score": score,
                "items": sector_items,
            },
        )
    return {
        "modelCode": "quant-rules",
        "direction": direction,
        "score": score,
        "risk": risk,
        "action": action,
        "summary": summary,
        "cards": cards,
        "attribution": attr or None,
    }
