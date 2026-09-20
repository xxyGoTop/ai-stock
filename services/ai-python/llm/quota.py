from __future__ import annotations

import threading
import time

# 免费额度用尽后先跳过，默认 6 小时后再试
COOLDOWN_SEC = 6 * 3600
_lock = threading.Lock()
_until: dict[str, float] = {}
_reason: dict[str, str] = {}

QUOTA_HINTS = (
    "quota",
    "rate limit",
    "rate_limit",
    "too many requests",
    "insufficient",
    "exceed",
    "overloaded",
    "余额",
    "额度",
    "限流",
    "用量",
    "次数",
    "配额",
    "超出",
)


def mark_exhausted(code: str, reason: str = "额度或限流", seconds: int = COOLDOWN_SEC) -> None:
    with _lock:
        _until[code] = time.time() + max(60, seconds)
        _reason[code] = reason


def is_exhausted(code: str) -> bool:
    with _lock:
        until = _until.get(code) or 0
        if until <= time.time():
            _until.pop(code, None)
            _reason.pop(code, None)
            return False
        return True


def status(code: str) -> dict:
    with _lock:
        until = _until.get(code) or 0
        left = max(0, int(until - time.time()))
        return {
            "exhausted": left > 0,
            "retryInSec": left,
            "reason": _reason.get(code) or "",
        }


def is_quota_error(exc: BaseException) -> bool:
    text = str(exc or "").lower()
    if any(k in text for k in ("429", "402")):
        return True
    return any(k in text for k in QUOTA_HINTS)


class QuotaError(RuntimeError):
    pass
