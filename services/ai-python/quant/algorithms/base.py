from __future__ import annotations

from typing import Any, Protocol


class Algorithm(Protocol):
    code: str
    name: str
    short: str
    category: str
    base_score: float

    def run(self, ctx: dict[str, Any]) -> dict[str, Any]:
        ...


def result(code: str, symbol: str, passed: bool, score: float, reason: str, metrics: dict | None = None) -> dict:
    return {
        "algorithmCode": code,
        "symbol": symbol,
        "pass": passed,
        "score": round(score, 2) if passed else 0,
        "reason": reason,
        "metrics": metrics or {},
    }
