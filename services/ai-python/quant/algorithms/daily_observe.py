from .base import result
from ..tao import eval_daily_observe


class DailyObserveAlgorithm:
    code = "daily_observe"
    name = "火车每日观察"
    short = "每日观察"
    category = "screening"
    base_score = 30

    def run(self, ctx: dict) -> dict:
        stock = ctx["stock"]
        ev = eval_daily_observe(ctx["klines"], ctx.get("rps") or {}, stock.get("turnover"))
        bonus = 0
        d = ev.get("detail") or {}
        if ev["hit"]:
            if d.get("xg1"):
                bonus += 5
            if d.get("xg2"):
                bonus += 4
            elif d.get("xg3"):
                bonus += 3
        return result(self.code, stock["symbol"], ev["hit"], self.base_score + bonus, ev["reason"], d)
