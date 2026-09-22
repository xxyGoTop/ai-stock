from __future__ import annotations

import json
import os
import re
import time
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed

UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36"
EM_HOSTS = [
    "https://push2.eastmoney.com",
    "https://82.push2.eastmoney.com",
    "https://push2delay.eastmoney.com",
]
FS = "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23,m:0+t:81+s:2048"
JSONP = re.compile(r"^[a-zA-Z0-9_]+\(")


def _get(url: str, referer: str, retries: int = 3) -> bytes:
    last: Exception | None = None
    for i in range(max(1, retries)):
        try:
            req = urllib.request.Request(
                url,
                headers={
                    "User-Agent": UA,
                    "Referer": referer,
                    "Accept": "application/json,text/plain,*/*",
                    "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
                    "Connection": "close",
                },
            )
            with urllib.request.urlopen(req, timeout=10) as res:
                return res.read()
        except Exception as exc:
            last = exc
            time.sleep(0.25 * (i + 1))
    raise last or RuntimeError(f"GET failed: {url}")


def _json(url: str, referer: str):
    raw = _get(url, referer).decode("utf-8", "ignore").strip()
    if JSONP.match(raw):
        raw = raw[raw.find("(") + 1 :]
        raw = raw.rstrip(";").rstrip(")")
    return json.loads(raw)


def _num(v):
    try:
        n = float(v)
        return n if n == n else 0.0
    except (TypeError, ValueError):
        return 0.0


def guess_market(code: str) -> str:
    c = str(code).zfill(6)
    return "SH" if c[:1] in "695" else "SZ"


def market_symbol(code: str) -> str:
    c = str(code).zfill(6)
    return ("sh" if guess_market(c) == "SH" else "sz") + c


def fetch_active_stocks(pages=4, page_size=100, fid="f6") -> list[dict]:
    fields = "f12,f13,f14,f2,f3,f4,f5,f6,f8,f10,f100"
    out = []
    for pn in range(1, pages + 1):
        query = (
            f"pn={pn}&pz={page_size}&po=1&np=1&fltt=2&invt=2&fid={fid}"
            f"&fs={urllib.parse.quote(FS)}&fields={fields}"
        )
        data = None
        for host in EM_HOSTS:
            try:
                data = _json(f"{host}/api/qt/clist/get?{query}", "https://quote.eastmoney.com/")
                break
            except Exception:
                continue
        rows = ((data or {}).get("data") or {}).get("diff") or []
        if not rows:
            break
        for item in rows:
            code = str(item.get("f12") or "").zfill(6)
            name = str(item.get("f14") or "").strip()
            if not code or not name:
                continue
            out.append(
                {
                    "symbol": code,
                    "name": name,
                    "market": "SH" if _num(item.get("f13")) == 1 else "SZ",
                    "price": _num(item.get("f2")),
                    "changePercent": _num(item.get("f3")),
                    "change": _num(item.get("f4")),
                    "volume": _num(item.get("f5")),
                    "amount": _num(item.get("f6")),
                    "turnover": _num(item.get("f8")),
                    "volumeRatio": _num(item.get("f10")),
                    "industry": str(item.get("f100") or "").strip(),
                }
            )
    uniq = {}
    for s in out:
        uniq[s["symbol"]] = s
    return list(uniq.values())


def fetch_market_returns(pages=12) -> list[dict]:
    fields = "f12,f14,f109,f24,f25"
    rows = []
    for pn in range(1, pages + 1):
        query = (
            f"pn={pn}&pz=100&po=1&np=1&fltt=2&invt=2&fid=f12"
            f"&fs={urllib.parse.quote(FS)}&fields={fields}"
        )
        data = None
        for host in EM_HOSTS:
            try:
                data = _json(f"{host}/api/qt/clist/get?{query}", "https://quote.eastmoney.com/")
                break
            except Exception:
                continue
        diff = ((data or {}).get("data") or {}).get("diff") or []
        if not diff:
            break
        for item in diff:
            code = str(item.get("f12") or "").zfill(6)
            name = str(item.get("f14") or "")
            if not code or "ST" in name.upper():
                continue
            rows.append(
                {
                    "code": code,
                    "change5": _num(item.get("f109")),
                    "change60": _num(item.get("f24")),
                    "changeYtd": _num(item.get("f25")),
                }
            )
        if len(diff) < 100:
            break
    return rows


def fetch_quote(code: str) -> dict | None:
    code = str(code).zfill(6)
    hit = _fetch_quote_ulist(code)
    if hit and hit.get("price"):
        if not hit.get("industry") or not hit.get("concepts"):
            meta = _fetch_quote_stock_get(code) or _fetch_f10_boards(code)
            if meta:
                if not hit.get("industry"):
                    hit["industry"] = meta.get("industry") or ""
                if not hit.get("region"):
                    hit["region"] = meta.get("region") or ""
                if not hit.get("concepts"):
                    hit["concepts"] = meta.get("concepts") or []
        return hit
    hit = _fetch_quote_stock_get(code, with_price=True)
    if hit and hit.get("price"):
        if not hit.get("industry") or not hit.get("concepts"):
            meta = _fetch_f10_boards(code)
            if meta:
                if not hit.get("industry"):
                    hit["industry"] = meta.get("industry") or ""
                if not hit.get("concepts"):
                    hit["concepts"] = meta.get("concepts") or []
        return hit
    via_go = _fetch_quote_via_go(code)
    if via_go and via_go.get("price"):
        if not via_go.get("industry") or not via_go.get("concepts"):
            meta = _fetch_f10_boards(code)
            if meta:
                if not via_go.get("industry"):
                    via_go["industry"] = meta.get("industry") or ""
                if not via_go.get("concepts"):
                    via_go["concepts"] = meta.get("concepts") or []
        return via_go
    return None


def _fetch_quote_ulist(code: str) -> dict | None:
    market = 1 if guess_market(code) == "SH" else 0
    fields = "f12,f13,f14,f2,f3,f4,f5,f6,f7,f8,f10,f21,f62,f66,f69,f72,f75,f100,f102,f103,f184"
    query = f"fltt=2&invt=2&fields={fields}&secids={market}.{code}"
    for host in EM_HOSTS:
        try:
            data = _json(f"{host}/api/qt/ulist.np/get?{query}", "https://quote.eastmoney.com/")
            rows = ((data or {}).get("data") or {}).get("diff") or []
            if not rows:
                continue
            item = rows[0]
            concepts = [x.strip() for x in str(item.get("f103") or "").split(",") if x.strip()][:8]
            return {
                "symbol": str(item.get("f12") or code).zfill(6),
                "name": str(item.get("f14") or "").strip(),
                "market": "SH" if _num(item.get("f13")) == 1 else "SZ",
                "price": _num(item.get("f2")),
                "changePercent": _num(item.get("f3")),
                "change": _num(item.get("f4")),
                "volume": _num(item.get("f5")),
                "amount": _num(item.get("f6")),
                "amplitude": _num(item.get("f7")),
                "turnover": _num(item.get("f8")),
                "volumeRatio": _num(item.get("f10")),
                "circMV": _num(item.get("f21")),
                "mainNetInflow": _num(item.get("f62")),
                "superNetInflow": _num(item.get("f66")),
                "superNetInflowPct": _num(item.get("f69")),
                "bigNetInflow": _num(item.get("f72")),
                "bigNetInflowPct": _num(item.get("f75")),
                "industry": str(item.get("f100") or "").strip(),
                "region": str(item.get("f102") or "").strip(),
                "concepts": concepts,
                "mainNetInflowPct": _num(item.get("f184")),
                "fundKnown": True,
            }
        except Exception:
            continue
    return None


def _fetch_quote_stock_get(code: str, with_price: bool = False) -> dict | None:
    """stock/get：行业在 f127，比 ulist 的 f100 更全。价格字段多为 *100 整数。"""
    market = 1 if guess_market(code) == "SH" else 0
    fields = "f57,f58,f43,f169,f170,f46,f44,f45,f47,f48,f50,f168,f100,f102,f103,f127"
    path = f"/api/qt/stock/get?secid={market}.{code}&fields={fields}"
    for host in EM_HOSTS:
        try:
            data = _json(f"{host}{path}", "https://quote.eastmoney.com/")
            item = (data or {}).get("data") or {}
            if not item:
                continue
            industry = str(item.get("f127") or item.get("f100") or "").strip()
            concepts = [x.strip() for x in str(item.get("f103") or "").split(",") if x.strip()][:8]
            out = {
                "symbol": str(item.get("f57") or code).zfill(6),
                "name": str(item.get("f58") or "").strip(),
                "market": guess_market(code),
                "industry": industry,
                "region": str(item.get("f102") or "").strip(),
                "concepts": concepts,
                "fundKnown": False,
            }
            if with_price:
                # f43 现价通常为「元 * 100」整数
                price = _num(item.get("f43"))
                if price > 1000:  # 启发式：茅台级或普通股放大 100
                    price = price / 100.0
                chg = _num(item.get("f169"))
                if abs(chg) > 30:  # 涨跌额也可能 *100
                    chg = chg / 100.0
                chg_pct = _num(item.get("f170"))
                if abs(chg_pct) > 30 and abs(chg_pct) < 3000:
                    chg_pct = chg_pct / 100.0
                out.update(
                    {
                        "price": price,
                        "change": chg,
                        "changePercent": chg_pct,
                        "open": _num(item.get("f46")) / 100.0 if _num(item.get("f46")) > 1000 else _num(item.get("f46")),
                        "high": _num(item.get("f44")) / 100.0 if _num(item.get("f44")) > 1000 else _num(item.get("f44")),
                        "low": _num(item.get("f45")) / 100.0 if _num(item.get("f45")) > 1000 else _num(item.get("f45")),
                        "volume": _num(item.get("f47")),
                        "amount": _num(item.get("f48")),
                        "volumeRatio": _num(item.get("f50")) / 100.0 if _num(item.get("f50")) > 20 else _num(item.get("f50")),
                        "turnover": _num(item.get("f168")) / 100.0 if _num(item.get("f168")) > 100 else _num(item.get("f168")),
                    }
                )
            return out
        except Exception:
            continue
    return None


def _fetch_quote_via_go(code: str) -> dict | None:
    """Python 直连东财失败时，回退到本机 Go API（Go 客户端更稳）。"""
    base = (os.getenv("GO_API_URL") or os.getenv("API_GO_URL") or "http://127.0.0.1:18080").rstrip("/")
    try:
        raw = _get(f"{base}/api/v1/stocks/{code}", base, retries=2)
        payload = json.loads(raw.decode("utf-8", "ignore"))
        data = payload.get("data") if isinstance(payload, dict) else None
        if not isinstance(data, dict) or not data.get("price"):
            return None
        return {
            "symbol": str(data.get("symbol") or code).zfill(6),
            "name": str(data.get("name") or "").strip(),
            "market": str(data.get("market") or guess_market(code)),
            "price": _num(data.get("price")),
            "changePercent": _num(data.get("changePercent")),
            "change": _num(data.get("change")),
            "volume": _num(data.get("volume")),
            "amount": _num(data.get("amount")),
            "turnover": _num(data.get("turnover")),
            "volumeRatio": _num(data.get("volumeRatio")),
            "amplitude": _num(data.get("amplitude")),
            "industry": str(data.get("industry") or "").strip(),
            "region": str(data.get("region") or "").strip(),
            "concepts": list(data.get("concepts") or [])[:8],
            "mainNetInflow": _num(data.get("mainNetInflow")),
            "mainNetInflowPct": _num(data.get("mainNetInflowPct")),
            "superNetInflow": _num(data.get("superNetInflow")),
            "bigNetInflow": _num(data.get("bigNetInflow")),
            "fundKnown": True,
        }
    except Exception:
        return None


def _fetch_f10_boards(code: str) -> dict:
    """F10 核心题材：ssbk 第一项作行业，其余作概念。"""
    code = str(code).zfill(6)
    prefix = "SH" if guess_market(code) == "SH" else "SZ"
    url = f"https://emweb.securities.eastmoney.com/PC_HSF10/CoreConception/PageAjax?code={prefix}{code}"
    try:
        data = _json(url, "https://emweb.securities.eastmoney.com/")
    except Exception:
        return {}
    rows = data.get("ssbk") or []
    names = []
    seen = set()
    for row in rows:
        name = str((row or {}).get("BOARD_NAME") or "").strip()
        if not name or name in seen:
            continue
        seen.add(name)
        names.append(name)
    if not names:
        return {}
    return {"industry": names[0], "concepts": names[1:9]}


def fetch_klines_tencent(code: str, limit=260) -> list[dict]:
    ms = market_symbol(code)
    urls = [
        f"https://web.ifzq.gtimg.cn/appstock/app/newfqkline/get?param={ms},day,,,{limit},qfq",
        f"https://proxy.finance.qq.com/ifzqgtimg/appstock/app/fqkline/get?param={ms},day,,,{limit},qfq",
        f"https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param={ms},day,,,{limit},qfq",
    ]
    data = None
    last = None
    for url in urls:
        try:
            data = _json(url, "https://gu.qq.com/")
            break
        except Exception as exc:
            last = exc
    if data is None:
        raise last or RuntimeError("tencent kline empty")
    node = ((data.get("data") or {}).get(ms) or {})
    raw = node.get("qfqday") or node.get("day") or []
    out = []
    prev = 0.0
    for row in raw:
        if not isinstance(row, (list, tuple)) or len(row) < 6:
            continue
        open_, close, high, low, volume = map(_num, row[1:6])
        chg = ((close - prev) / prev * 100) if prev else 0
        out.append(
            {
                "date": str(row[0])[:10],
                "open": open_,
                "close": close,
                "high": high,
                "low": low,
                "volume": volume,
                "changePercent": chg,
                "turnover": 0,
            }
        )
        prev = close
    return out


def fetch_klines_sina(code: str, limit=260) -> list[dict]:
    ms = market_symbol(code)
    url = (
        "https://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData"
        f"?symbol={ms}&scale=240&ma=5&datalen={limit}"
    )
    raw = _json(url, "https://finance.sina.com.cn/")
    out = []
    prev = 0.0
    for row in raw or []:
        if not isinstance(row, dict):
            continue
        close = _num(row.get("close"))
        chg = ((close - prev) / prev * 100) if prev else 0
        out.append(
            {
                "date": str(row.get("day") or "")[:10],
                "open": _num(row.get("open")),
                "close": close,
                "high": _num(row.get("high")),
                "low": _num(row.get("low")),
                "volume": _num(row.get("volume")),
                "changePercent": chg,
                "turnover": 0,
            }
        )
        prev = close
    return out


def fetch_klines_eastmoney(code: str, limit=260) -> list[dict]:
    market = 1 if guess_market(code) == "SH" else 0
    path = (
        f"/api/qt/stock/kline/get?secid={market}.{str(code).zfill(6)}"
        "&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61"
        f"&klt=101&fqt=1&end=20500101&lmt={limit}"
    )
    hosts = [
        "https://push2his.eastmoney.com",
        "https://push2delay.eastmoney.com",
        "https://82.push2.eastmoney.com",
    ]
    for host in hosts:
        try:
            data = _json(host + path, "https://quote.eastmoney.com/")
        except Exception:
            continue
        raw = ((data or {}).get("data") or {}).get("klines") or []
        out = []
        for line in raw:
            parts = str(line).split(",")
            if len(parts) < 7:
                continue
            out.append(
                {
                    "date": parts[0][:10],
                    "open": _num(parts[1]),
                    "close": _num(parts[2]),
                    "high": _num(parts[3]),
                    "low": _num(parts[4]),
                    "volume": _num(parts[5]),
                    "changePercent": _num(parts[8]) if len(parts) > 8 else 0,
                    "turnover": _num(parts[9]) if len(parts) > 9 else 0,
                }
            )
        if out:
            return out
    return []


def fetch_quotes_by_symbols(symbols: list[str]) -> list[dict]:
    """按给定代码批量拉行情快照，供板块内选股限定候选池。"""
    codes = []
    seen = set()
    for raw in symbols:
        c = str(raw).strip().zfill(6)
        if not c or c in seen:
            continue
        seen.add(c)
        codes.append(c)
    out: list[dict] = []
    for i in range(0, len(codes), 40):
        chunk = codes[i : i + 40]
        secids = []
        for c in chunk:
            m = 1 if guess_market(c) == "SH" else 0
            secids.append(f"{m}.{c}")
        query = (
            "fltt=2&invt=2&fields=f12,f14,f2,f3,f4,f5,f6,f8,f10,f100"
            f"&secids={urllib.parse.quote(','.join(secids), safe=',.')}"
        )
        data = None
        for host in EM_HOSTS:
            try:
                data = _json(f"{host}/api/qt/ulist.np/get?{query}", "https://quote.eastmoney.com/")
                break
            except Exception:
                continue
        rows = ((data or {}).get("data") or {}).get("diff") or []
        if isinstance(rows, dict):
            rows = list(rows.values())
        for item in rows:
            code = str(item.get("f12") or "").zfill(6)
            name = str(item.get("f14") or "").strip()
            if not code or not name:
                continue
            out.append(
                {
                    "symbol": code,
                    "name": name,
                    "market": guess_market(code),
                    "price": _num(item.get("f2")),
                    "changePercent": _num(item.get("f3")),
                    "change": _num(item.get("f4")),
                    "volume": _num(item.get("f5")),
                    "amount": _num(item.get("f6")),
                    "turnover": _num(item.get("f8")),
                    "volumeRatio": _num(item.get("f10")),
                    "industry": str(item.get("f100") or "").strip(),
                }
            )
    # 保序：按入参 symbols 顺序
    by_code = {s["symbol"]: s for s in out}
    ordered = [by_code[c] for c in codes if c in by_code]
    return ordered


def fetch_klines(code: str, limit=260) -> list[dict]:
    for fn in (fetch_klines_tencent, fetch_klines_sina, fetch_klines_eastmoney):
        try:
            bars = fn(code, limit)
            if len(bars) >= 10:
                return bars
        except Exception:
            continue
    return []


def fetch_klines_many(codes: list[str], limit=260, workers=10) -> dict[str, list]:
    result = {}
    with ThreadPoolExecutor(max_workers=workers) as pool:
        futs = {pool.submit(fetch_klines, c, limit): c for c in codes}
        for fut in as_completed(futs):
            code = futs[fut]
            try:
                bars = fut.result()
                if len(bars) >= 60:
                    result[code] = bars
            except Exception:
                continue
    return result
