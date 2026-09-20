from __future__ import annotations


def fmt_money(n, signed=True) -> str:
    try:
        v = float(n)
    except (TypeError, ValueError):
        return "-"
    yi = v / 1e8
    if abs(yi) >= 0.01:
        return f"{yi:+.2f}亿" if signed else f"{abs(yi):.2f}亿"
    wan = v / 1e4
    return f"{wan:+.0f}万" if signed else f"{abs(wan):.0f}万"


def fmt_price(n) -> str:
    try:
        return f"{float(n):.2f}"
    except (TypeError, ValueError):
        return "-"


def describe_fund(main: float | None, pct: float | None, known: bool) -> dict:
    if not known or main is None:
        return {
            "status": "资金数据缺失",
            "level": "unknown",
            "text": "资金：接口未返回",
            "mainNetInflow": None,
            "mainNetInflowPct": None,
        }
    pct = float(pct or 0)
    if main > 5e7 or pct >= 5:
        status, level = "主力大幅流入", "strong_in"
    elif main > 1e7 or pct >= 2:
        status, level = "主力净流入", "in"
    elif main < -5e7 or pct <= -5:
        status, level = "主力大幅流出", "strong_out"
    elif main < -1e7 or pct <= -2:
        status, level = "主力净流出", "out"
    else:
        status, level = "资金博弈平衡", "neutral"
    return {
        "status": status,
        "level": level,
        "text": f"{status} {fmt_money(main)}（占成交 {pct:+.2f}%）",
        "mainNetInflow": main,
        "mainNetInflowPct": round(pct, 2),
    }


def classify_capital(stock: dict) -> dict:
    turn = _f(stock.get("turnover"))
    circ = _f(stock.get("circMV"))
    amp = _f(stock.get("amplitude"))
    vr = _f(stock.get("volumeRatio"))
    super_in = _f(stock.get("superNetInflow"))
    big_in = _f(stock.get("bigNetInflow"))
    main = _f(stock.get("mainNetInflow"))
    super_pct = _f(stock.get("superNetInflowPct"))
    limit = int(stock.get("limitUpStreak") or 0)
    inst = hot = 0
    hints = []
    if circ is not None and circ >= 200e8:
        inst += 2
        hints.append("流通市值偏大")
    elif circ is not None and circ >= 80e8:
        inst += 1
    elif circ is not None and 0 < circ < 40e8:
        hot += 2
        hints.append("流通盘偏小")
    elif circ is not None and circ < 80e8:
        hot += 1
    if turn is not None:
        if turn <= 4:
            inst += 2
            hints.append(f"换手{turn:.1f}%偏低")
        elif turn <= 8:
            inst += 1
        elif turn >= 18:
            hot += 2
            hints.append(f"换手{turn:.1f}%偏高")
        elif turn >= 12:
            hot += 1
    if amp is not None and amp >= 9:
        hot += 1
        hints.append(f"振幅{amp:.1f}%")
    if vr is not None and vr >= 2.8:
        hot += 1
    if limit >= 1:
        hot += 2
        hints.append("涨停博弈")
    if super_in is not None and super_in > 2e6:
        inst += 1
        hints.append("超大单净买")
    elif super_in is not None and super_in < -2e6 and (turn is None or turn >= 8):
        hot += 1
    if inst >= hot + 1 and inst >= 2:
        kind, label = "inst", "机构股"
    elif hot >= inst + 1 and hot >= 2:
        kind, label = "hot", "游资博弈股"
    else:
        kind, label = "mixed", "机构/游资混合"
    if super_in is None:
        inst_text = "机构净买入：超大单未取到"
    elif super_in >= 0:
        pct = f"，占成交 {super_pct:+.2f}%" if super_pct is not None else ""
        inst_text = f"机构净买入 {fmt_money(super_in)}（超大单{pct}）"
    else:
        pct = f"，占成交 {super_pct:+.2f}%" if super_pct is not None else ""
        inst_text = f"机构净卖出 {fmt_money(abs(super_in), signed=False)}（超大单{pct}）"
    extras = []
    if main is not None:
        extras.append(f"主力{'净买' if main >= 0 else '净卖'} {fmt_money(abs(main), signed=False)}")
    if big_in is not None:
        extras.append(f"大单{'净买' if big_in >= 0 else '净卖'} {fmt_money(abs(big_in), signed=False)}")
    extras.extend(hints[:2])
    return {
        "kind": kind,
        "kindLabel": label,
        "instText": inst_text,
        "text": f"{label} · {inst_text}" + (f"｜{'，'.join(extras[:3])}" if extras else ""),
        "note": "，".join(hints[:3]),
    }


def estimate_chips(klines: list[dict], lookback=90) -> dict:
    if len(klines) < 20:
        return {"status": "数据不足", "text": "筹码：K线不足", "concentration90": None, "profitRatio": None}
    rows = klines[-min(lookback, len(klines)) :]
    price_min = min(k["low"] for k in rows)
    price_max = max(k["high"] for k in rows)
    if not (price_max > price_min):
        return {"status": "数据不足", "text": "筹码：价格区间无效", "concentration90": None, "profitRatio": None}
    bins = 60
    step = (price_max - price_min) / bins
    chip = [0.0] * bins
    for k in rows:
        turn = min(max((k.get("turnover") or 0) / 100, 0.001), 0.35)
        chip = [x * (1 - turn) for x in chip]
        lo = max(price_min, k["low"])
        hi = min(price_max, k["high"])
        vol = max(k.get("volume") or 0, 1)
        i0 = max(0, int((lo - price_min) / step))
        i1 = min(bins - 1, int((hi - price_min) / step))
        add = vol / max(1, i1 - i0 + 1)
        for i in range(i0, i1 + 1):
            chip[i] += add
    total = sum(chip) or 1
    cum = 0.0
    p5, p95, avg = price_min, price_max, 0.0
    for i, v in enumerate(chip):
        mid = price_min + (i + 0.5) * step
        avg += mid * (v / total)
        prev, cum = cum, cum + v / total
        if prev < 0.05 <= cum:
            p5 = mid
        if prev < 0.95 <= cum:
            p95 = mid
    price = rows[-1]["close"]
    profit = sum(chip[i] for i in range(bins) if price_min + (i + 0.5) * step <= price)
    profit_ratio = round(profit / total * 100, 1)
    conc90 = round((p95 - p5) / avg * 100, 1) if avg else None
    if conc90 is None:
        status = "数据不足"
    elif conc90 <= 8:
        status = "高度集中"
    elif conc90 <= 12:
        status = "较集中"
    elif conc90 <= 18:
        status = "中等集中"
    elif conc90 <= 25:
        status = "偏分散"
    else:
        status = "分散"
    return {
        "status": status,
        "concentration90": conc90,
        "profitRatio": profit_ratio,
        "avgCost": round(avg, 2),
        "cost90": [round(p5, 2), round(p95, 2)],
        "text": f"筹码{status} 集中度90={conc90}%｜获利{profit_ratio}%｜成本区 {p5:.2f}-{p95:.2f}" if conc90 is not None else "筹码：暂无",
    }


def build_expect(stock: dict, fund: dict) -> str:
    concepts = [c for c in (stock.get("concepts") or []) if c][:4]
    industry = (stock.get("industry") or "").strip()
    bits = []
    if concepts:
        bits.append(f"热点题材：{'、'.join(concepts)}")
    if industry:
        bits.append(f"所属行业 {industry}，热度中性")
    if fund.get("text") and fund.get("level") not in (None, "unknown"):
        bits.append(fund["text"])
    return "；".join(bits)


def build_verdict(stock: dict, ind: dict, fund: dict, chips: dict, klines: list[dict] | None = None) -> dict:
    klines = klines or []
    price = _f(stock.get("price")) or 0
    change = _f(stock.get("changePercent"))
    turn = _f(stock.get("turnover"))
    ma = _ma(ind, price)
    rsi = _rsi(ind.get("rsi14"))
    macd = _macd(ind)
    fund_block = _fund(stock, fund, change)
    support_px = _key_support(klines)
    hold = _support_held(klines, support_px)
    near_high = _near_high(klines, price)
    chip_block = _chips(chips, klines, near_high)
    support = _support_text(support_px, hold)
    volume = _volume(klines)

    hist = _f(ind.get("hist"))
    expanding = bool(ind.get("macdHistExpanding"))
    if hist is not None and hist > 0:
        macd_tag = f"MACD：红柱{'放大' if expanding else '续增'} {hist:+.4f}"
        macd_hot = expanding
    elif hist is not None and hist < 0:
        macd_tag = f"MACD：绿柱 {hist:+.4f}"
        macd_hot = False
    else:
        macd_tag = "MACD：走平"
        macd_hot = False

    pros, cons = [], []
    if support["tone"] == "good":
        prefix = "放量上涨，" if volume["tone"] == "good" else ""
        pros.append(f"{prefix}回踩不击穿核心支撑 {fmt_price(support_px)}，低点承接有力")
    if ma["tone"] == "good":
        if near_high:
            pros.append("短期均线多头，盘中创出近端新高")
        else:
            pros.append("短期均线多头，股价站在 5/10/20 日线上方")
    if turn is not None and 7 <= turn <= 13:
        pros.append("换手维持在 7~13% 健康活跃区间，没有爆量到 18% 以上疯狂派发")
    elif turn is not None and 3 <= turn < 7:
        pros.append(f"换手 {turn:.1f}%，活跃度一般，未见疯狂派发")
    if rsi["tone"] == "good":
        pros.append("RSI 维持 50 以上且未进超买")
    if chip_block["tone"] == "good" and not chip_block.get("loosening"):
        pros.append("未见连续长阴砸盘，筹码没有快速松动")
    if fund_block["tone"] == "good":
        pros.append(fund_block["text"])

    if rsi["tone"] == "warn" and (rsi.get("value") or 0) > 70:
        cons.append("RSI 进入超买区，短期获利盘丰厚")
    if "逢高兑现" in (fund_block.get("note") or ""):
        cons.append("股价上涨，但主力资金小幅流出，边拉边卖")
    if turn is not None and turn >= 18:
        cons.append(f"换手 {turn:.1f}%，超过 18%，有疯狂派发嫌疑")
    if chip_block.get("loosening"):
        cons.append(chip_block["text"].rstrip("。"))
    if support["tone"] == "bad":
        cons.append(support["text"].rstrip("。"))
    if volume["tone"] == "bad":
        cons.append(volume["text"].rstrip("。"))
    if ma["tone"] == "bad":
        cons.append("短期均线多头破坏，回调不是沿均线的健康回踩")
    if rsi["tone"] == "bad":
        cons.append("RSI 跌破 50，短期动能转弱")
    if not expanding and hist is not None and hist > 0:
        cons.append("红柱未继续大幅拉长，动能有放缓迹象")

    if not pros and ma["tone"] != "bad":
        pros.append(ma["text"].rstrip("。"))
    if not cons:
        cons.append("暂无明显隐患，仍需按计划止损")

    items = [
        {"key": "ma", "title": "均线", "text": ma["text"], "tone": ma["tone"]},
        {"key": "rsi", "title": "RSI", "text": rsi["text"], "hint": rsi.get("note") or "", "tone": rsi["tone"]},
        {"key": "macd", "title": "MACD", "text": macd["text"], "tone": macd["tone"]},
        {"key": "fund", "title": "资金面", "text": fund_block["text"], "hint": fund_block.get("note") or "", "tone": fund_block["tone"]},
        {"key": "support", "title": "支撑", "text": support["text"], "tone": support["tone"]},
        {"key": "volume", "title": "量能", "text": volume["text"], "tone": volume["tone"]},
    ]
    return {
        "macdTag": macd_tag,
        "macdHot": macd_hot,
        "items": items,
        "pros": _uniq(pros)[:5],
        "cons": _uniq(cons)[:5],
    }


def _ma(ind: dict, price: float) -> dict:
    ma5, ma10, ma20 = _f(ind.get("ma5")), _f(ind.get("ma10")), _f(ind.get("ma20"))
    if ma5 is None:
        return {"tone": "muted", "text": "均线数据不足，暂不判断多空排列。"}
    all_up = bool(ind.get("ma5Rising") and ind.get("ma10Rising") and ind.get("ma20Rising"))
    above_all = price >= ma5 and (ma10 is None or price >= ma10) and (ma20 is None or price >= ma20)
    if ind.get("bullAlign") and all_up and above_all:
        return {"tone": "good", "text": "5/10/20 日线全部向上，股价稳稳站在全部短期均线上方，均线多头形态完好。"}
    if above_all and ind.get("bullAlign"):
        return {"tone": "good", "text": "股价站在 5/10/20 日线上方，均线多头排列，部分均线斜率一般，整体仍偏强。"}
    if above_all and not all_up:
        return {"tone": "warn", "text": "股价还在短期均线上方，但 5/10/20 并未全部向上，多头尚未完全确认。"}
    if ind.get("bullAlign") and not above_all:
        return {"tone": "warn", "text": f"均线仍多头排列，但现价 {fmt_price(price)} 已跌破部分均线，回踩是否有效要看 {fmt_price(ma5)} 一线承接。"}
    if ind.get("aboveMa5"):
        return {"tone": "warn", "text": "仅站上五日线，10/20 日线尚未形成多头，只能当短线，跌破五日线就要走。"}
    return {"tone": "bad", "text": "短期均线未形成多头，或股价已跌破关键均线，回调不是健康回踩。"}


def _rsi(rsi14) -> dict:
    v = _f(rsi14)
    if v is None:
        return {"value": None, "tone": "muted", "text": "RSI14 数据不足。", "note": ""}
    if v > 70:
        return {
            "value": round(v, 2),
            "tone": "warn",
            "text": f"RSI14 ≈{v:.2f}",
            "note": "已经进入超买区间（＞70），代表短期多头力量很强，但同时提示：短期有回调风险，不宜追高。",
        }
    if v >= 50:
        return {
            "value": round(v, 2),
            "tone": "good",
            "text": f"RSI14 ≈{v:.2f}",
            "note": "维持在 50 以上，多头占优，尚未进入超买，短线仍可沿均线持有。",
        }
    return {
        "value": round(v, 2),
        "tone": "bad",
        "text": f"RSI14 ≈{v:.2f}",
        "note": "跌破 50，短期动能转弱，需要先观察能否重新站回 50 再谈加仓。",
    }


def _macd(ind: dict) -> dict:
    hist, dif, dea = _f(ind.get("hist")), _f(ind.get("dif")), _f(ind.get("dea"))
    if hist is None or dif is None or dea is None:
        return {"tone": "muted", "text": "MACD 数据不足。"}
    above = dif > dea
    if hist > 0 and above:
        if ind.get("macdHistExpanding"):
            return {"tone": "good", "text": "红柱继续拉长，DIF 在 DEA 上方，多头趋势，动能仍在增强。"}
        return {"tone": "warn", "text": "红柱继续，DIF 在 DEA 上方，多头趋势，但红柱没有继续大幅拉长，多头动能有放缓迹象。"}
    if hist < 0 and ind.get("macdGreenShrinking"):
        return {"tone": "warn", "text": "仍处绿柱但绿柱缩短，DIF 向 DEA 收敛，空头动能减弱。"}
    if ind.get("macdGolden"):
        return {"tone": "good", "text": "MACD 金叉确认，DIF 上穿 DEA，多头刚切换。"}
    if hist < 0:
        return {"tone": "bad", "text": "绿柱运行，DIF 在 DEA 下方，空头仍占优。"}
    return {"tone": "warn", "text": "红柱刚翻正或贴近零轴，多头刚起步，还要看能否延续。"}


def _fund(stock: dict, fund: dict, change: float | None) -> dict:
    inflow = _f(stock.get("mainNetInflow") if stock.get("mainNetInflow") is not None else fund.get("mainNetInflow"))
    if inflow is None:
        return {"tone": "muted", "text": "资金数据不足。", "note": ""}
    text = f"当日主力资金{'净流入' if inflow >= 0 else '净流出'} {fmt_money(abs(inflow), signed=False)}"
    if change is not None and change >= 2 and inflow < -1e6:
        return {
            "tone": "warn",
            "text": text,
            "note": "股价大涨，但主力小幅流出，属于拉涨过程中有资金逢高兑现，属于要留意的小隐患，不是纯粹资金单边进攻行情。",
        }
    if change is not None and change >= 1 and inflow > 1e6:
        return {"tone": "good", "text": text, "note": "价涨且主力净流入，资金与走势同向，比单纯拉涨更健康。"}
    if change is not None and change <= -1 and inflow < -1e6:
        return {"tone": "bad", "text": text, "note": "股价下跌且主力净流出，资金与走势同向走弱。"}
    if change is not None and change <= -0.5 and inflow > 1e6:
        return {"tone": "warn", "text": text, "note": "股价回落但主力净流入，更像逢低吸筹，要看回踩是否站稳。"}
    pct = fund.get("mainNetInflowPct")
    note = f"{fund.get('status') or ''}，占成交 {pct}%。" if fund.get("status") else ""
    return {"tone": "good" if inflow >= 0 else "warn", "text": text, "note": note}


def _chips(chips: dict, klines: list[dict], near_high: bool) -> dict:
    bears = _long_bears(klines)
    status = chips.get("status") or ""
    if bears >= 2 and near_high:
        return {"tone": "bad", "text": f"高位出现连续 {bears} 根长阴砸盘，筹码有快速松动迹象。", "loosening": True}
    if bears >= 2:
        return {"tone": "warn", "text": f"近端连续 {bears} 根长阴，要注意筹码是否开始松动。", "loosening": True}
    if near_high:
        extra = chips.get("text") or "高位震荡时筹码未快速大面积松动，没有连续长阴砸盘。"
        return {"tone": "good", "text": f"{extra}。高位震荡未见连续长阴砸盘，筹码没有快速大面积松动。" if chips.get("text") else extra, "loosening": False}
    return {"tone": "good" if "集中" in status else "muted", "text": chips.get("text") or "筹码：暂无足够K线。", "loosening": False}


def _support_text(support: float | None, hold: dict) -> dict:
    if support is None:
        return {"tone": "muted", "text": "支撑：K线不足，暂不判断回踩是否有效。"}
    if hold.get("broke"):
        return {"tone": "bad", "text": f"回踩已有效跌破核心支撑 {fmt_price(support)}，低点承接失败。"}
    if hold.get("held"):
        return {"tone": "good", "text": f"股价站稳关键支撑 {fmt_price(support)}，回踩不有效跌破，低点承接有力。"}
    if hold.get("wickBroke"):
        return {"tone": "warn", "text": f"盘中刺破支撑 {fmt_price(support)} 但收盘收回，暂算回踩未有效跌破，仍要盯下一根。"}
    return {"tone": "warn", "text": f"核心支撑看 {fmt_price(support)}，当前尚未确认是否站稳。"}


def _volume(klines: list[dict]) -> dict:
    recent = klines[-8:]
    up, down = [], []
    for k in recent:
        vol = _f(k.get("volume"))
        if not vol:
            continue
        if (k.get("close") or 0) >= (k.get("open") or 0):
            up.append(vol)
        else:
            down.append(vol)
    avg_up = sum(up) / len(up) if up else None
    avg_down = sum(down) / len(down) if down else None
    shrink = avg_up is not None and avg_down is not None and avg_down < avg_up * 0.88
    expand = avg_up is not None and avg_down is not None and avg_up > avg_down * 1.08
    if shrink and expand:
        return {"tone": "good", "text": "回调缩量，拉升放量，量价配合健康。"}
    if expand and not shrink:
        return {"tone": "warn", "text": "拉升放量，但回调并未明显缩量，回撤时抛压仍在。"}
    if shrink:
        return {"tone": "warn", "text": "回调缩量，抛压不大，但拉升放量还不充分。"}
    if avg_up is not None and avg_down is not None and avg_down > avg_up * 1.15:
        return {"tone": "bad", "text": "下跌放量、上涨缩量，量价背离，不宜当强势看待。"}
    return {"tone": "muted", "text": "量能：近端涨跌日成交对比不足。"}


def _key_support(klines: list[dict]) -> float | None:
    rows = klines[-21:-1] if len(klines) > 6 else klines[:-1]
    lows = [_f(k.get("low")) for k in rows if _f(k.get("low")) is not None]
    return min(lows) if lows else None


def _support_held(klines: list[dict], support: float | None) -> dict:
    if not support or not klines:
        return {"held": None, "broke": False, "wickBroke": False}
    wick = close_broke = False
    for k in klines[-10:]:
        if _f(k.get("close")) is not None and k["close"] < support * 0.997:
            close_broke = True
        if _f(k.get("low")) is not None and k["low"] < support * 0.99:
            wick = True
    last = klines[-1]
    last_held = _f(last.get("close")) is not None and last["close"] >= support * 0.997
    return {"held": last_held and not close_broke, "broke": close_broke, "wickBroke": wick}


def _near_high(klines: list[dict], price: float) -> bool:
    highs = [_f(k.get("high")) or 0 for k in klines[-60:]]
    high60 = max(highs) if highs else 0
    return bool(high60 > 0 and price >= high60 * 0.985)


def _long_bears(klines: list[dict]) -> int:
    streak = max_streak = 0
    for k in klines[-12:]:
        open_, close = _f(k.get("open")), _f(k.get("close"))
        if not open_ or close is None:
            streak = 0
            continue
        body = (open_ - close) / open_
        if close < open_ and body >= 0.03:
            streak += 1
            max_streak = max(max_streak, streak)
        else:
            streak = 0
    return max_streak


def _uniq(items: list[str]) -> list[str]:
    seen, out = set(), []
    for item in items:
        if item and item not in seen:
            seen.add(item)
            out.append(item)
    return out


def _f(v):
    try:
        if v is None or v == "" or v == "-":
            return None
        return float(v)
    except (TypeError, ValueError):
        return None
