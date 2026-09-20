from .base import result
from ..tao import eval_forward_train


class ForwardTrainAlgorithm:
    code = "forward_train"
    name = "顺向火车轨"
    short = "火车轨"
    category = "screening"
    base_score = 36

    def run(self, ctx: dict) -> dict:
        stock = ctx["stock"]
        ev = eval_forward_train(ctx["klines"], ctx.get("rps") or {}, stock.get("turnover"))
        bonus = 0
        d = ev.get("detail") or {}
        if ev["hit"]:
            if (d.get("rps250") or 0) >= 98:
                bonus += 7
            elif (d.get("rps250") or 0) >= 95:
                bonus += 4
            if d.get("yearHighRight"):
                bonus += 6
            if d.get("belowMa10"):
                bonus += 6
            if (d.get("pullback20") or 1) <= 0.12:
                bonus += 3
        return result(self.code, stock["symbol"], ev["hit"], self.base_score + bonus, ev["reason"], d)
