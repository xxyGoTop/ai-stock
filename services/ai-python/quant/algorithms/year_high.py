from .base import result
from ..tao import eval_tao_pick_241005


class YearHighAlgorithm:
    code = "year_high"
    name = "率先一年新高"
    short = "年新高"
    category = "screening"
    base_score = 46

    def run(self, ctx: dict) -> dict:
        stock = ctx["stock"]
        pick = eval_tao_pick_241005(ctx["klines"], ctx.get("rps") or {}, stock.get("turnover"), ctx.get("indexCanBuy", True))
        via = pick.get("via") or []
        passed = bool(pick["hit"] and "率先年新高" in via)
        bonus = 0
        detail = []
        if passed:
            ago = pick["detail"].get("yearHighBarsAgo")
            detail.append("今日最高价即250日新高" if ago == 0 else f"{ago}日前创250日新高")
            if ctx.get("inFirstPage"):
                rank = ctx.get("observeRank") or 99
                bonus += 8 if rank <= 10 else 5 if rank <= 20 else 2
                detail.append(f"当日涨幅榜第一版第{rank}名")
            else:
                bonus -= 5
                detail.append("未进当日涨幅榜第一版，强度不足")
            if pick.get("weakBreakout"):
                bonus -= 8
                detail.append("无指数中期信号，需防假突破")
        reason = "；".join(detail) if passed else pick["reason"]
        return result(self.code, stock["symbol"], passed, self.base_score + bonus, reason, pick.get("detail"))
