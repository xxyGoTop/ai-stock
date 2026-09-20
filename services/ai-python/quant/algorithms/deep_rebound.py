from .base import result
from ..tao import count_bases, eval_tao_pick_241005, sma


class DeepReboundAlgorithm:
    code = "deep_rebound"
    name = "高RPS深调回升"
    short = "深调回升"
    category = "screening"
    base_score = 40

    def run(self, ctx: dict) -> dict:
        stock = ctx["stock"]
        klines = ctx["klines"]
        pick = eval_tao_pick_241005(klines, ctx.get("rps") or {}, stock.get("turnover"), ctx.get("indexCanBuy", True))
        via = pick.get("via") or []
        passed = bool(pick["hit"] and "深调高RPS回升" in via)
        bases = count_bases(klines)
        bonus = 0
        if bases["baseCount"] == 1:
            bonus += 8
        elif bases["baseCount"] == 2:
            bonus += 5
        elif bases["baseCount"] == 3:
            bonus -= 6
        elif bases["baseCount"] > 3:
            bonus -= 12
        detail = [f"第{bases['baseCount']}个基底" if bases["baseCount"] else "未形成基底"]
        ratio = pick["detail"].get("priceRatio")
        if ratio is not None:
            detail.append(f"现价为年内最高的 {ratio * 100:.0f}%")
            if ratio >= 0.85:
                bonus += 6
            elif ratio < 0.6:
                bonus -= 12
        closes = [k["close"] for k in klines]
        ma20 = sma(closes, 20)[-1]
        ma50 = sma(closes, 50)[-1] if len(closes) >= 50 else None
        price = stock.get("price") or closes[-1]
        if ma20 and price > ma20 and ma50 and price > ma50:
            bonus += 5
            detail.append("已站上20/50日线")
        elif not ma20 or price <= ma20:
            bonus -= 8
            detail.append("尚未站上20日线")
        reason = "；".join(detail) if passed else pick["reason"]
        metrics = {**pick.get("detail", {}), **bases}
        return result(self.code, stock["symbol"], passed, self.base_score + bonus, reason, metrics)
