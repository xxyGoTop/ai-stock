from __future__ import annotations

from .indicators import sma


def hhv(arr: list[float], n: int) -> float:
    return max(arr[-n:]) if arr else 0.0


def hhv_bars_at(arr: list[float], n: int, i: int):
    start = max(0, i - n + 1)
    peak = -1e18
    bars = 0
    for j in range(start, i + 1):
        if arr[j] >= peak:
            peak = arr[j]
            bars = i - j
    return bars, peak


def llv_bars_at(arr: list[float], lookback_bars: int, i: int):
    if lookback_bars <= 0:
        return 0, arr[i]
    start = max(0, i - lookback_bars)
    trough = 1e18
    bars = 0
    for j in range(start, i + 1):
        if arr[j] <= trough:
            trough = arr[j]
            bars = i - j
    return bars, trough


def max_pullback_ok(highs, lows, i, window, max_ratio):
    high_bars, high_price = hhv_bars_at(highs, window, i)
    if not (high_price > 0):
        return False, 1.0
    _, low_price = llv_bars_at(lows, high_bars, i)
    pullback = (high_price - low_price) / high_price
    if pullback > max_ratio:
        return False, pullback
    start = i - high_bars
    peak = highs[start]
    for j in range(start, i + 1):
        if highs[j] >= peak:
            peak = highs[j]
        dd = (peak - lows[j]) / peak if peak > 0 else 0
        if dd > max_ratio:
            return False, dd
    return True, pullback


def count_close_above_ma(closes, ma_arr, lookback, i):
    n = 0
    for d in range(lookback):
        idx = i - d
        if idx < 0 or ma_arr[idx] is None:
            continue
        if closes[idx] > ma_arr[idx]:
            n += 1
    return n


def every_ma_rising(ma_arr, lookback, i):
    for d in range(lookback):
        idx = i - d
        if idx < 1 or ma_arr[idx] is None or ma_arr[idx - 1] is None:
            return False
        if ma_arr[idx] < ma_arr[idx - 1]:
            return False
    return True


def every_ma10_above_ma20(ma10, ma20, lookback, i):
    for d in range(lookback):
        idx = i - d
        if idx < 0 or ma10[idx] is None or ma20[idx] is None:
            return False
        if ma10[idx] < ma20[idx]:
            return False
    return True


def count_bases(klines, min_drop=0.15, min_bars=10, reset_drop=0.4, lookback=300):
    empty = {"baseCount": 0, "inBase": False, "drawdown": 0, "baseDrop": 0, "baseBars": 0, "resetCount": 0, "lastResetBarsAgo": None}
    if not klines:
        return empty
    bars = klines[-lookback:]
    highs = [k["high"] for k in bars]
    lows = [k["low"] for k in bars]
    closes = [k["close"] for k in bars]
    last = len(bars) - 1
    anchor = 0
    for i in range(1, last + 1):
        if lows[i] < lows[anchor]:
            anchor = i
    peak = highs[anchor]
    base_count = 0
    in_base = False
    base_low = 1e18
    base_start = anchor
    last_base_drop = 0.0
    reset_count = 0
    last_reset_idx = None
    for i in range(anchor + 1, last + 1):
        if not in_base:
            if highs[i] > peak:
                peak = highs[i]
            if peak > 0 and (peak - lows[i]) / peak >= min_drop:
                in_base = True
                base_low = lows[i]
                base_start = i
            continue
        if lows[i] < base_low:
            base_low = lows[i]
        dd = (peak - base_low) / peak if peak > 0 else 0
        if dd >= reset_drop:
            base_count = 0
            reset_count += 1
            last_reset_idx = i
            in_base = False
            peak = highs[i]
            base_low = 1e18
            continue
        if closes[i] > peak:
            if i - base_start >= min_bars:
                base_count += 1
                last_base_drop = dd
            in_base = False
            peak = highs[i]
            base_low = 1e18
    base_bars = 0
    if in_base:
        base_bars = last - base_start
        if base_bars >= min_bars:
            base_count += 1
            last_base_drop = (peak - base_low) / peak if peak > 0 else 0
        else:
            in_base = False
    drawdown = (peak - closes[last]) / peak if peak > 0 else 0
    return {
        "baseCount": base_count,
        "inBase": in_base,
        "drawdown": round(drawdown, 3),
        "baseDrop": round(last_base_drop, 3),
        "baseBars": base_bars,
        "resetCount": reset_count,
        "lastResetBarsAgo": None if last_reset_idx is None else last - last_reset_idx,
    }


def eval_daily_observe(klines, rps, turnover):
    if not klines or len(klines) < 120:
        return {"hit": False, "reason": "K线不足", "detail": {}}
    highs = [k["high"] for k in klines]
    lows = [k["low"] for k in klines]
    closes = [k["close"] for k in klines]
    i = len(klines) - 1
    rps120 = rps.get("rps120") or 0
    rps250 = rps.get("rps250") or 0
    rps50 = rps.get("rps50") or 0
    turn = turnover if turnover is not None else (klines[i].get("turnover") or 0)
    mrgc00 = turn < 25
    win120 = min(120, i + 1)
    ok50, pb50 = max_pullback_ok(highs, lows, i, win120, 0.5)
    hhv_c250 = hhv(closes, min(250, len(closes)))
    hhv_h250 = hhv(highs, min(250, len(highs)))
    mrgc002 = hhv_c250 > 0 and closes[i] / hhv_c250 > 0.7
    mrgc01 = ok50 and mrgc002
    ok35, _ = max_pullback_ok(highs, lows, i, win120, 0.35)
    mrgc_hc = ok35 and hhv_c250 > 0 and closes[i] / hhv_c250 > 0.8
    xg11 = any(hhv_c250 > 0 and closes[i - d] >= hhv_c250 * 0.9999 for d in range(5) if i - d >= 0)
    xg12 = rps120 > 95.99 or rps250 > 95.99
    xg13 = rps120 > 94.99 and rps50 > 94.99
    xg1 = xg11 and (xg12 or xg13)
    xg2 = hhv_h250 > 0 and closes[i] / hhv_h250 >= 0.85 and (rps120 > 96.99 or rps250 > 96.99)
    xg3 = hhv_h250 > 0 and closes[i] / hhv_h250 >= 0.7 and (rps120 > 97.99 or rps250 > 97.99)
    xg4 = mrgc_hc and (rps120 > 94.99 or rps250 > 94.99)
    hit = bool(mrgc00 and mrgc01 and (xg1 or xg2 or xg3 or xg4))
    parts = []
    if xg1:
        parts.append("近5日收盘年高+高RPS")
    if xg2:
        parts.append("价≥年高85%+极高RPS")
    if xg3:
        parts.append("价≥年高70%+超高RPS")
    if xg4:
        parts.append("120日回撤≤35%+高RPS")
    if not mrgc00:
        parts.append("换手≥25%剔除")
    if not mrgc01:
        parts.append("120日回撤/年高位置未达标")
    reason = "；".join([p for p in parts if "剔除" not in p and "未达标" not in p]) if hit else "；".join(parts) or "未满足火车每日观察"
    return {
        "hit": hit,
        "reason": reason or "火车每日观察",
        "detail": {"xg1": xg1, "xg2": xg2, "xg3": xg3, "xg4": xg4, "mrgc00": mrgc00, "mrgc01": mrgc01, "pullback120": round(pb50, 3), "rps50": rps50, "rps120": rps120, "rps250": rps250, "turn": round(float(turn), 2)},
    }


def eval_forward_train(klines, rps, turnover):
    if not klines or len(klines) < 250:
        return {"hit": False, "reason": "K线不足(需≥250日)", "detail": {}}
    highs = [k["high"] for k in klines]
    lows = [k["low"] for k in klines]
    closes = [k["close"] for k in klines]
    i = len(closes) - 1
    turn = turnover if turnover is not None else (klines[i].get("turnover") or 0)
    rps120 = rps.get("rps120") or 0
    rps250 = rps.get("rps250") or 0
    ma10 = sma(closes, 10)
    ma20 = sma(closes, 20)
    ma200 = sma(closes, 200)
    ma250 = sma(closes, 250)
    sxhcg1 = rps120 + rps250 > 185
    sxhcg20 = ma20[i] is not None and closes[i] > ma20[i]
    sxhcg21 = count_close_above_ma(closes, ma250, 30, i) >= 25
    sxhcg22 = count_close_above_ma(closes, ma200, 30, i) >= 25
    sxhcg23 = count_close_above_ma(closes, ma20, 10, i) >= 9
    sxhcg24 = count_close_above_ma(closes, ma10, 4, i) >= 3 and count_close_above_ma(closes, ma20, 4, i) >= 3
    sxhcg2 = sxhcg20 and sxhcg21 and sxhcg22 and (sxhcg23 or sxhcg24)
    ok20, pb20 = max_pullback_ok(highs, lows, i, min(20, i + 1), 0.25)
    hhv_c250 = hhv(closes, min(250, len(closes)))
    sxhcg3 = ok20 and hhv_c250 > 0 and closes[i] / hhv_c250 > 0.8
    sxhcg41 = every_ma_rising(ma20, 5, i) and every_ma10_above_ma20(ma10, ma20, 5, i)
    sxhcg42 = (
        ma10[i] is not None
        and ma10[i - 1] is not None
        and ma20[i] is not None
        and ma20[i - 1] is not None
        and ma10[i] >= ma10[i - 1]
        and ma20[i] >= ma20[i - 1]
        and ma10[i] >= ma20[i]
    )
    sxhcg4 = sxhcg41 or sxhcg42
    sxhcg5 = turn < 15
    ok120, pb120 = max_pullback_ok(highs, lows, i, min(120, i + 1), 0.5)
    hit = bool(sxhcg1 and sxhcg2 and sxhcg3 and sxhcg4 and sxhcg5 and ok120)
    parts = []
    if not sxhcg1:
        parts.append(f"RPS120+250={(rps120 + rps250):.1f}≤185")
    if not sxhcg2:
        parts.append("未站稳中长期均线")
    if not sxhcg3:
        parts.append("20日回撤或年高位置未达标")
    if not sxhcg4:
        parts.append("均线未顺向")
    if not sxhcg5:
        parts.append("换手≥15%")
    if not ok120:
        parts.append("120日回撤>50%")
    below_ma10 = ma10[i] is not None and closes[i] < ma10[i]
    year_high_right = hhv_c250 > 0 and closes[i] >= hhv_c250 * 0.9999
    reason = (
        f"顺向火车轨(RPS合计{(rps120 + rps250):.0f}{'·偏好10日线下买点' if below_ma10 else ''}{'·右侧年高' if year_high_right else ''})"
        if hit
        else "；".join(parts) or "未满足顺向火车轨"
    )
    return {
        "hit": hit,
        "reason": reason,
        "detail": {
            "rps120": rps120,
            "rps250": rps250,
            "rpsSum": round(rps120 + rps250, 2),
            "pullback20": round(pb20, 3),
            "pullback120": round(pb120, 3),
            "belowMa10": below_ma10,
            "yearHighRight": year_high_right,
            "turn": round(float(turn), 2),
        },
    }


def eval_tao_pick_241005(klines, rps, turnover, index_can_buy=True):
    if not klines or len(klines) < 120:
        return {"hit": False, "reason": "K线不足", "via": [], "weakBreakout": False, "detail": {}}
    highs = [k["high"] for k in klines]
    closes = [k["close"] for k in klines]
    i = len(klines) - 1
    rps120 = rps.get("rps120") or 0
    rps250 = rps.get("rps250") or 0
    turn = turnover if turnover is not None else (klines[i].get("turnover") or 0)
    rps_base = max(rps120, rps250)
    premise_ok = rps_base >= 95
    hhv_h250 = hhv(highs, min(250, len(highs)))
    xg11 = False
    year_high_bars_ago = None
    for d in range(5):
        if i - d >= 0 and hhv_h250 > 0 and highs[i - d] >= hhv_h250 * 0.9999:
            xg11 = True
            year_high_bars_ago = d
            break
    xg1 = xg11 and (rps120 > 96 or rps250 > 96)
    price_ratio = closes[i] / hhv_h250 if hhv_h250 > 0 else 0
    xg2 = price_ratio >= 0.5 and (rps120 > 98 or rps250 > 98)
    xg3 = turn < 20
    bases = count_bases(klines)
    via = []
    if xg1:
        via.append("率先年新高")
    if xg2 and not xg1:
        via.append("深调高RPS回升")
    hit = bool(premise_ok and (xg1 or xg2) and xg3)
    weak = xg1 and not xg2 and not index_can_buy
    notes = []
    if not premise_ok:
        notes.append(f"RPS120/250 最高仅{rps_base:.0f}，未达95前提")
    if not xg3:
        notes.append(f"换手{float(turn):.1f}%≥20%，游资票剔除")
    if hit and bases["baseCount"] >= 3:
        notes.append(f"已是第{bases['baseCount']}个基底，文章建议谨慎")
    if weak:
        notes.append("无指数中期信号，年新高需防假突破")
    if hit:
        extra = f"·第{bases['baseCount']}基底" if bases["baseCount"] else ""
        reason = f"{'+'.join(via)}（RPS120={rps120:.0f}/RPS250={rps250:.0f}{extra}）"
        if notes:
            reason += "｜" + "；".join(notes)
    else:
        reason = "；".join(notes) or "未满足241005择股思路"
    return {
        "hit": hit,
        "via": via,
        "reason": reason,
        "weakBreakout": weak,
        "detail": {
            "premiseOk": premise_ok,
            "xg1": xg1,
            "xg2": xg2,
            "xg3": xg3,
            "rps120": rps120,
            "rps250": rps250,
            "rpsBase": round(rps_base, 2),
            "priceRatio": round(price_ratio, 3),
            "yearHighBarsAgo": year_high_bars_ago,
            "baseCount": bases["baseCount"],
            "baseDrop": bases["baseDrop"],
            "drawdown": bases["drawdown"],
            "turn": round(float(turn), 2),
        },
    }


def rank_rps(extra_by_code: dict[str, float | None]) -> dict[str, float]:
    entries = [(c, v) for c, v in extra_by_code.items() if v is not None]
    entries.sort(key=lambda x: x[1])
    n = len(entries)
    out = {}
    for i, (code, _) in enumerate(entries):
        out[code] = 100.0 if n == 1 else round(i / (n - 1) * 100, 2)
    return out


def extras(klines, n):
    if not klines or len(klines) <= n:
        return None
    c = klines[-1]["close"]
    ref = klines[-1 - n]["close"]
    if not ref:
        return None
    return (c - ref) / ref


def mix_rps(a: dict, b: dict, wa: float) -> dict:
    out = {}
    for code, vb in b.items():
        if code in a:
            out[code] = round(a[code] * wa + vb * (1 - wa), 2)
    return out
