from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor, as_completed

from .config import get_model, get_profile, model_ready
from .providers import call_model
from .quant_analyst import analyze_quant

RISK_RANK = {"low": 0, "mid": 1, "high": 2}


def run_profile(context: dict, profile_code: str | None = None) -> dict:
    profile = get_profile(profile_code)
    entries = [e for e in profile.get("models") or [] if (m := get_model(e["modelCode"])) and model_ready(m)]
    if not entries:
        vote = analyze_quant(context)
        return _pack(profile, [vote], [vote])

    mode = profile.get("mode") or "single"
    if mode == "single":
        votes = [_invoke(entries[0], context)]
    elif mode == "fallback":
        votes = []
        last_err = None
        chain = profile.get("fallback") or [e["modelCode"] for e in entries]
        for code in chain:
            entry = next((e for e in entries if e["modelCode"] == code), {"modelCode": code, "temperature": 0.2})
            try:
                votes = [_invoke(entry, context)]
                break
            except Exception as exc:
                last_err = exc
        if not votes:
            raise RuntimeError(f"全部模型失败：{last_err}")
    else:
        votes = []
        with ThreadPoolExecutor(max_workers=min(3, len(entries))) as pool:
            futs = {pool.submit(_invoke, e, context): e["modelCode"] for e in entries}
            for fut in as_completed(futs):
                try:
                    votes.append(fut.result())
                except Exception:
                    continue
        if not votes:
            votes = [analyze_quant(context)]
    return _pack(profile, entries, votes)


def _invoke(entry: dict, context: dict) -> dict:
    model = get_model(entry["modelCode"])
    if not model:
        raise RuntimeError("model missing")
    return call_model(model, context, float(entry.get("temperature") or 0.2))


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
        "mode": profile.get("mode"),
        "usedModels": [v.get("modelCode") for v in votes],
        "votes": [{"modelCode": v.get("modelCode"), "direction": v.get("direction"), "score": v.get("score"), "risk": v.get("risk")} for v in votes],
        "final": {"direction": direction, "score": final_score, "risk": risk, "action": primary.get("action"), "summary": primary.get("summary")},
        "cards": primary.get("cards") or [],
        "summaries": [{"modelCode": v.get("modelCode"), "summary": v.get("summary")} for v in votes],
    }
