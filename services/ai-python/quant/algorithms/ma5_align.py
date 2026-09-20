from .base import result
from ..indicators import compute_snapshot


class Ma5AlignAlgorithm:
    code = "ma5_align"
    name = "五日线多头共振"
    short = "五日线"
    category = "screening"
    base_score = 24

    def run(self, ctx: dict) -> dict:
        stock = ctx["stock"]
        ind = ctx.get("indicators") or compute_snapshot(ctx["klines"])
        veto = []
        if (ind.get("limitUpStreak") or 0) >= 2:
            veto.append("连续涨停高位")
        if ind.get("bias5") is not None and ind["bias5"] > 10:
            veto.append(f"乖离过大({ind['bias5']:.1f}%)")
        if "ST" in (stock.get("name") or "").upper():
            veto.append("ST 不纳入短线正股池")
        vol_ratio = stock.get("volumeRatio") or 0
        vol_ok = vol_ratio >= 1.3 or (ind.get("gentleVolume") and (ind.get("lastVolume") or 0) > (ind.get("prevVolume") or 0))
        must = ind.get("bullAlign") and ind.get("aboveMa5") and ind.get("ma5Rising") and vol_ok and not veto
        bonus = 0
        detail = []
        if must:
            detail.append("MA5>MA10>MA20 且站上向上五日线")
            bias5 = ind.get("bias5")
            if bias5 is not None and 0 <= bias5 <= 2.5:
                bonus += 4
                detail.append(f"乖离 +{bias5:.1f}%")
            elif bias5 is not None and bias5 > 6:
                bonus -= 4
                detail.append(f"乖离偏大 +{bias5:.1f}%")
            if vol_ratio >= 1.5:
                bonus += 3
            if ind.get("macdAboveZero") and (ind.get("macdGolden") or ind.get("macdHistExpanding")):
                bonus += 2
        reason = "；".join(detail) if must else ("；".join(veto) or "未满足五日线多头共振")
        return result(self.code, stock["symbol"], bool(must), self.base_score + bonus, reason, {"bias5": ind.get("bias5"), "volumeRatio": vol_ratio})
