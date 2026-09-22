from __future__ import annotations

import urllib.parse
from typing import Any

from quant.screening.market import EM_HOSTS, _json, _num

_NOISE = (
    "连板", "涨停", "跌停", "昨日", "次新", "ST", "风险警示", "退市", "融资融券",
    "标准普尔", "富时", "MSCI", "沪股通", "深股通", "中字头", "破净",
)


def build_sector_context(
    stock: dict,
    *,
    news_titles: list[str] | None = None,
    notice_titles: list[str] | None = None,
    hot_topics: list[str] | None = None,
) -> dict[str, Any]:
    """行业/概念板块快照 + 规则化涨跌归因（数字均来自行情，不编造）。"""
    industry = str(stock.get("industry") or "").strip()
    concepts = [str(c).strip() for c in (stock.get("concepts") or []) if str(c).strip()][:8]
    change = float(stock.get("changePercent") or 0)

    industry_board = find_board(industry, board_type=3) if industry else None
    concept_boards: list[dict] = []
    for name in concepts:
        hit = find_board(name, board_type=2)
        if hit:
            concept_boards.append(hit)

    # 与当日热门概念重叠（按板块涨幅榜）
    hot_concept = []
    try:
        hot_list = list_boards(2, 40)
    except Exception:
        hot_list = []
    concept_set = {c.lower() for c in concepts}
    for b in hot_list:
        nm = str(b.get("name") or "")
        if not nm:
            continue
        if nm.lower() in concept_set or any(nm in c or c in nm for c in concepts):
            hot_concept.append(b)
        if len(hot_concept) >= 5:
            break

    topics = [t for t in (hot_topics or []) if t][:8]
    matched_topics = []
    blob = " ".join(concepts + ([industry] if industry else []))
    for t in topics:
        if t and t in blob:
            matched_topics.append(t)

    news = [t for t in (news_titles or []) if t][:5]
    notices = [t for t in (notice_titles or []) if t][:5]

    attribution = rule_attribution(
        change=change,
        industry_board=industry_board,
        concept_boards=concept_boards or hot_concept,
        news_titles=news,
        notice_titles=notices,
        matched_topics=matched_topics,
    )

    return {
        "industry": industry,
        "concepts": concepts,
        "industryBoard": industry_board,
        "conceptBoards": (concept_boards or hot_concept)[:5],
        "matchedTopics": matched_topics,
        "newsTitles": news,
        "noticeTitles": notices,
        "attribution": attribution,
    }


def rule_attribution(
    *,
    change: float,
    industry_board: dict | None,
    concept_boards: list[dict],
    news_titles: list[str],
    notice_titles: list[str],
    matched_topics: list[str],
) -> dict[str, Any]:
    drivers: list[dict[str, str]] = []
    scores = {"sector": 0.0, "theme": 0.0, "announcement": 0.0, "idiosyncratic": 0.0}

    board_chg = float((industry_board or {}).get("changePercent") or 0)
    if industry_board and abs(board_chg) >= 0.3:
        same = (change >= 0 and board_chg >= 0) or (change < 0 and board_chg < 0)
        gap = abs(change - board_chg)
        if same and gap <= 4:
            scores["sector"] += 3 + min(3, abs(board_chg))
            direction = "跟涨" if board_chg > 0 else "跟跌"
            drivers.append(
                {
                    "kind": "sector",
                    "label": "板块",
                    "detail": f"所属行业「{industry_board.get('name')}」{board_chg:+.2f}%，个股与板块同向（偏离 {gap:.2f}pct），偏{direction}板块。",
                }
            )
        elif same and abs(board_chg) >= 1.5:
            scores["sector"] += 2
            drivers.append(
                {
                    "kind": "sector",
                    "label": "板块",
                    "detail": f"行业「{industry_board.get('name')}」{board_chg:+.2f}%，与个股同向，有板块共振。",
                }
            )
        elif not same and abs(change) >= 1:
            scores["idiosyncratic"] += 2
            drivers.append(
                {
                    "kind": "idiosyncratic",
                    "label": "个股独立",
                    "detail": f"行业「{industry_board.get('name')}」{board_chg:+.2f}%，个股 {change:+.2f}% 与板块背离，更像个股因素。",
                }
            )

    theme_hits = []
    for b in concept_boards[:5]:
        chg = float(b.get("changePercent") or 0)
        if abs(chg) < 1.2:
            continue
        same = (change >= 0 and chg >= 0) or (change < 0 and chg < 0)
        if same:
            theme_hits.append(b)
            scores["theme"] += 2 + min(2, abs(chg) / 2)
    if theme_hits:
        names = "、".join(f"{b.get('name')}({float(b.get('changePercent') or 0):+.1f}%)" for b in theme_hits[:3])
        drivers.append(
            {
                "kind": "theme",
                "label": "题材",
                "detail": f"关联概念走强/走弱：{names}，个股走势与题材同向。",
            }
        )
    if matched_topics:
        scores["theme"] += 1.5
        drivers.append(
            {
                "kind": "theme",
                "label": "题材",
                "detail": "今日热点题材相关：" + "、".join(matched_topics[:4]),
            }
        )
    if notice_titles:
        scores["announcement"] += 3.5
        drivers.append(
            {
                "kind": "announcement",
                "label": "公告",
                "detail": "近期有公告：" + "；".join(notice_titles[:2]),
            }
        )
    elif news_titles and abs(change) >= 1.5:
        scores["announcement"] += 1.5
        drivers.append(
            {
                "kind": "announcement",
                "label": "资讯",
                "detail": "相关新闻：" + "；".join(news_titles[:2]),
            }
        )

    if abs(change) >= 2 and scores["sector"] < 1.5 and scores["theme"] < 1.5 and scores["announcement"] < 1.5:
        scores["idiosyncratic"] += 2.5
        drivers.append(
            {
                "kind": "idiosyncratic",
                "label": "个股独立",
                "detail": f"涨跌幅 {change:+.2f}% 较明显，但板块/题材/公告信号不强，偏个股资金或情绪。",
            }
        )

    # 选主因
    primary = max(scores, key=scores.get)
    if scores[primary] < 1.2:
        primary = "mixed" if drivers else "unknown"
    elif sum(1 for v in scores.values() if v >= 2.5) >= 2:
        primary = "mixed"

    label_map = {
        "sector": "板块影响",
        "theme": "题材影响",
        "announcement": "公告/资讯",
        "idiosyncratic": "个股独立",
        "mixed": "多因素共振",
        "unknown": "信号不足",
    }
    move = "上涨" if change > 0.3 else "下跌" if change < -0.3 else "震荡"
    explanation = f"今日偏{move}（{change:+.2f}%）。主因判断：{label_map.get(primary, primary)}。"
    if drivers:
        explanation += " " + drivers[0]["detail"]

    return {
        "primary": primary,
        "primaryLabel": label_map.get(primary, primary),
        "move": move,
        "stockChangePercent": round(change, 2),
        "industryChangePercent": round(board_chg, 2) if industry_board else None,
        "drivers": drivers[:5],
        "explanation": explanation,
        "scores": {k: round(v, 2) for k, v in scores.items()},
    }


def find_board(keyword: str, board_type: int = 3) -> dict | None:
    keyword = str(keyword or "").strip()
    if not keyword:
        return None
    try:
        boards = list_boards(board_type, 120)
    except Exception:
        return None
    key = keyword.lower().replace("板块", "").replace("概念", "")
    best = None
    best_score = 0
    for b in boards:
        name = str(b.get("name") or "")
        if not name or any(n in name for n in _NOISE):
            continue
        nl = name.lower()
        score = 0
        if nl == key or name == keyword:
            score = 100
        elif key in nl or nl in key:
            score = 80
        elif key[:2] and key[:2] in nl:
            score = 40
        if score > best_score:
            best_score = score
            best = b
    return best if best_score >= 40 else None


def list_boards(board_type: int, limit: int = 80) -> list[dict]:
    """board_type: 2=概念 3=行业。"""
    if limit <= 0 or limit > 200:
        limit = 80
    fs = f"m:90+t:{board_type}"
    fields = "f12,f14,f2,f3,f8,f104,f105,f109"
    query = urllib.parse.urlencode(
        {
            "pn": 1,
            "pz": limit,
            "po": 1,
            "np": 1,
            "fltt": 2,
            "invt": 2,
            "fid": "f3",
            "fs": fs,
            "fields": fields,
        }
    )
    last_err: Exception | None = None
    for host in EM_HOSTS:
        try:
            data = _json(f"{host}/api/qt/clist/get?{query}", "https://quote.eastmoney.com/")
            diff = ((data.get("data") or {}).get("diff")) or []
            out = []
            for item in diff:
                name = str(item.get("f14") or "").strip()
                code = str(item.get("f12") or "").strip()
                if not name or not code or any(n in name for n in _NOISE):
                    continue
                out.append(
                    {
                        "code": code,
                        "name": name,
                        "changePercent": _num(item.get("f3")),
                        "change5": _num(item.get("f109")),
                        "leader": str(item.get("f104") or "").strip(),
                        "leaderCode": str(item.get("f105") or "").strip(),
                        "boardType": "concept" if board_type == 2 else "industry",
                    }
                )
            if out:
                return out
        except Exception as exc:
            last_err = exc
            continue
    if last_err:
        raise last_err
    return []


def sector_lines(ctx: dict) -> str:
    lines = []
    ind = ctx.get("industry") or ""
    if ind:
        lines.append(f"所属行业：{ind}")
    ib = ctx.get("industryBoard") or {}
    if ib:
        lines.append(
            f"行业板块：{ib.get('name')} 今日 {float(ib.get('changePercent') or 0):+.2f}% "
            f"（近5日 {float(ib.get('change5') or 0):+.2f}%），领涨 {ib.get('leader') or '-'}"
        )
    concepts = ctx.get("concepts") or []
    if concepts:
        lines.append("概念标签：" + "、".join(concepts[:6]))
    for b in (ctx.get("conceptBoards") or [])[:4]:
        lines.append(f"概念板块：{b.get('name')} {float(b.get('changePercent') or 0):+.2f}%")
    attr = ctx.get("attribution") or {}
    if attr.get("explanation"):
        lines.append("涨跌归因：" + attr["explanation"])
    for d in attr.get("drivers") or []:
        lines.append(f"- [{d.get('label')}] {d.get('detail')}")
    notices = ctx.get("noticeTitles") or []
    if notices:
        lines.append("近期公告：" + "；".join(notices[:3]))
    news = ctx.get("newsTitles") or []
    if news:
        lines.append("相关新闻：" + "；".join(news[:3]))
    return "\n".join(lines) if lines else "暂无板块/归因数据"
