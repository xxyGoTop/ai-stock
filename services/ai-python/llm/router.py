from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor, as_completed

from .config import get_model, get_profile, model_callable, model_ready
from .providers import call_model
from .quant_analyst import analyze_quant
from .quota import QuotaError, is_exhausted, is_quota_error, mark_exhausted

RISK_RANK = {"low": 0, "mid": 1, "high": 2}


def run_profile(context: dict, profile_code: str | None = None, model_code: str | None = None) -> dict:
    if model_code:
        return _run_named_model(context, model_code)
    profile = get_profile(profile_code)
    entries = [e for e in profile.get("models") or [] if (m := get_model(e["modelCode"])) and model_ready(m)]
    if not entries:
        vote = analyze_quant(context)
        return _pack(profile, [vote], [vote])

    agent_code = profile.get("agentCode")
    mode = profile.get("mode") or "single"
    if mode == "single":
        votes = [_invoke_or_fallback(entries, profile, context, agent_code)]
    elif mode == "fallback":
        votes = [_invoke_or_fallback(entries, profile, context, agent_code)]
    else:
        votes = []
        usable = [e for e in entries if not is_exhausted(e["modelCode"])]
        with ThreadPoolExecutor(max_workers=min(3, len(usable) or 1)) as pool:
            futs = {pool.submit(_invoke, e, context, agent_code): e["modelCode"] for e in usable}
            for fut in as_completed(futs):
                try:
                    vote = fut.result()
                    if _usable(vote):
                        votes.append(vote)
                except QuotaError as exc:
                    mark_exhausted(futs[fut], str(exc))
                except Exception:
                    continue
        if not votes:
            votes = [analyze_quant(context)]
    packed = _pack(profile, entries, votes)
    if not packed.get("cards") or not (packed.get("final") or {}).get("summary"):
        fallback = analyze_quant(context)
        packed["cards"] = fallback.get("cards") or []
        if not (packed.get("final") or {}).get("summary"):
            packed["final"] = {**(packed.get("final") or {}), "action": fallback.get("action"), "summary": fallback.get("summary")}
    return packed


def _run_named_model(context: dict, model_code: str) -> dict:
    model = get_model(model_code)
    profile = {
        "code": f"model_{model_code}",
        "name": (model or {}).get("label") or model_code,
        "task": "stock_analysis",
        "agentCode": "stock_analyst",
        "mode": "fallback",
        "models": [{"modelCode": model_code, "weight": 1, "temperature": 0.2}],
        "fallback": [model_code, "quant-rules"],
        "enabled": True,
    }
    entries = [e for e in profile["models"] if (m := get_model(e["modelCode"])) and model_ready(m)]
    if not entries:
        vote = analyze_quant(context)
        return _pack(profile, [vote], [vote])
    votes = [_invoke_or_fallback(entries, profile, context, profile["agentCode"])]
    return _pack(profile, entries, votes)


def _invoke_or_fallback(entries: list[dict], profile: dict, context: dict, agent_code: str | None = None) -> dict:
    chain = profile.get("fallback") or [e["modelCode"] for e in entries]
    last_err = None
    for code in chain:
        model = get_model(code)
        if not model or not model_callable(model):
            continue
        entry = next((e for e in entries if e["modelCode"] == code), {"modelCode": code, "temperature": 0.2})
        try:
            vote = _invoke(entry, context, agent_code)
            if _usable(vote):
                return vote
            last_err = RuntimeError(f"{code} 返回空结论")
        except QuotaError as exc:
            mark_exhausted(code, str(exc))
            last_err = exc
        except Exception as exc:
            last_err = exc
            if is_quota_error(exc):
                mark_exhausted(code, str(exc))
    if last_err:
        return analyze_quant(context)
    return analyze_quant(context)


def _usable(vote: dict | None) -> bool:
    if not vote:
        return False
    summary = str(vote.get("summary") or "").strip()
    if summary in ("", "{}", "模型未返回结构化结论"):
        return bool(vote.get("cards"))
    return True


def _invoke(entry: dict, context: dict, agent_code: str | None = None) -> dict:
    model = get_model(entry["modelCode"])
    if not model:
        raise RuntimeError("model missing")
    return call_model(model, context, float(entry.get("temperature") or 0.2), agent_code)


def _pack(profile: dict, entries: list[dict], votes: list[dict]) -> dict:
    weight = {e["modelCode"]: float(e.get("weight") or 1) for e in profile.get("models") or []}
    total_w = 0.0
    score_sum = 0.0
    ballot: dict[str, float] = {}
    risk = "low"
    for v in votes:
        w = weight.get(v.get("modelCode") or "", 1)
        total_w += w
        score_sum += (v.get("score") or 50) * w
        d = v.get("direction") or "neutral"
        ballot[d] = ballot.get(d, 0) + w
        if RISK_RANK.get(v.get("risk") or "mid", 1) > RISK_RANK[risk]:
            risk = v.get("risk") or risk
    final_score = round(score_sum / total_w, 1) if total_w else 50
    direction = max(ballot, key=ballot.get) if ballot else "neutral"
    primary = max(votes, key=lambda x: weight.get(x.get("modelCode") or "", 1))
    return {
        "profileCode": profile["code"],
        "profileName": profile.get("name") or profile["code"],
        "agentCode": profile.get("agentCode") or "stock_analyst",
        "mode": profile.get("mode"),
        "usedModels": [v.get("modelCode") for v in votes],
        "votes": [{"modelCode": v.get("modelCode"), "direction": v.get("direction"), "score": v.get("score"), "risk": v.get("risk")} for v in votes],
        "final": {"direction": direction, "score": final_score, "risk": risk, "action": primary.get("action"), "summary": primary.get("summary")},
        "cards": primary.get("cards") or [],
        "summaries": [{"modelCode": v.get("modelCode"), "summary": v.get("summary")} for v in votes],
    }
