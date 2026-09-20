from __future__ import annotations


def sma(values: list[float], period: int) -> list[float | None]:
    out: list[float | None] = [None] * len(values)
    if period <= 0:
        return out
    total = 0.0
    for i, v in enumerate(values):
        total += v
        if i >= period:
            total -= values[i - period]
        if i >= period - 1:
            out[i] = total / period
    return out


def ema(values: list[float], period: int) -> list[float | None]:
    out: list[float | None] = [None] * len(values)
    k = 2 / (period + 1)
    prev = None
    total = 0.0
    for i, v in enumerate(values):
        if i < period:
            total += v
            if i == period - 1:
                prev = total / period
                out[i] = prev
            continue
        if prev is None:
            continue
        prev = v * k + prev * (1 - k)
        out[i] = prev
    return out


def macd(closes: list[float], fast=12, slow=26, signal=9):
    ef = ema(closes, fast)
    es = ema(closes, slow)
    dif = [None] * len(closes)
    raw = [0.0] * len(closes)
    for i, _ in enumerate(closes):
        if ef[i] is not None and es[i] is not None:
            dif[i] = ef[i] - es[i]
            raw[i] = dif[i]
    dea = ema(raw, signal)
    hist = [None] * len(closes)
    for i, _ in enumerate(closes):
        if dif[i] is not None and dea[i] is not None:
            hist[i] = (dif[i] - dea[i]) * 2
        else:
            dea[i] = None
    return dif, dea, hist


def rsi(closes: list[float], period=6) -> list[float | None]:
    out: list[float | None] = [None] * len(closes)
    if len(closes) <= period:
        return out
    gain = loss = 0.0
    for i in range(1, period + 1):
        diff = closes[i] - closes[i - 1]
        if diff >= 0:
            gain += diff
        else:
            loss -= diff
    avg_gain = gain / period
    avg_loss = loss / period
    out[period] = 100.0 if avg_loss == 0 else 100 - 100 / (1 + avg_gain / avg_loss)
    for i in range(period + 1, len(closes)):
        diff = closes[i] - closes[i - 1]
        g = diff if diff > 0 else 0.0
        l = -diff if diff < 0 else 0.0
        avg_gain = (avg_gain * (period - 1) + g) / period
        avg_loss = (avg_loss * (period - 1) + l) / period
        out[i] = 100.0 if avg_loss == 0 else 100 - 100 / (1 + avg_gain / avg_loss)
    return out


def last(arr):
    return arr[-1] if arr else None


def compute_snapshot(klines: list[dict]) -> dict:
    closes = [k["close"] for k in klines]
    volumes = [k.get("volume") or 0 for k in klines]
    ma5 = sma(closes, 5)
    ma10 = sma(closes, 10)
    ma20 = sma(closes, 20)
    dif, dea, hist = macd(closes)
    rsi6 = rsi(closes, 6)
    i = len(closes) - 1
    prev = i - 1
    price = closes[i]
    ma5v, ma10v, ma20v = ma5[i], ma10[i], ma20[i]
    bias5 = ((price - ma5v) / ma5v * 100) if ma5v else None
    bull_align = bool(ma5v and ma10v and ma20v and ma5v > ma10v > ma20v)
    ma5_rising = bool(i >= 3 and ma5v and ma5[i - 3] and ma5v > ma5[i - 3])
    above_ma5 = bool(ma5v and price > ma5v)
    macd_golden = False
    macd_hist_expanding = False
    macd_above_zero = bool(dif[i] is not None and dea[i] is not None and dif[i] > 0 and dea[i] > 0)
    if prev >= 0 and dif[prev] is not None and dea[prev] is not None and dif[i] is not None and dea[i] is not None:
        macd_golden = dif[prev] <= dea[prev] and dif[i] > dea[i]
    if prev >= 0 and hist[i] is not None and hist[prev] is not None:
        macd_hist_expanding = hist[i] > 0 and hist[i] > hist[prev]
    last_vol = volumes[i]
    prev_vol = volumes[prev] if prev >= 0 else 0
    gentle = False
    if len(volumes) >= 5:
        recent = volumes[-5:]
        avg = sum(recent) / 5
        gentle = recent[-1] > recent[0] * 1.05 and recent[-1] >= avg * 0.9 and max(recent) <= avg * 2.8
    limit_up = 0
    for k in reversed(klines[-5:]):
        if (k.get("changePercent") or 0) >= 9.5:
            limit_up += 1
        else:
            break
    return {
        "ma5": ma5v,
        "ma10": ma10v,
        "ma20": ma20v,
        "bias5": bias5,
        "bullAlign": bull_align,
        "ma5Rising": ma5_rising,
        "aboveMa5": above_ma5,
        "macdGolden": macd_golden,
        "macdHistExpanding": macd_hist_expanding,
        "macdAboveZero": macd_above_zero,
        "lastVolume": last_vol,
        "prevVolume": prev_vol,
        "gentleVolume": gentle,
        "limitUpStreak": limit_up,
        "lastTurnover": klines[i].get("turnover") or 0,
        "hist": hist[i],
        "rsi6": rsi6[i],
    }
