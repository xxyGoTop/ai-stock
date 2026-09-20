from __future__ import annotations

import json
import os
from functools import lru_cache
from pathlib import Path

from .quota import status as quota_status

CONFIG_PATH = Path(__file__).resolve().parents[1] / "config" / "llm.json"


def load_dotenv() -> None:
    roots = [
        Path(__file__).resolve().parents[3] / ".env",
        Path(__file__).resolve().parents[1] / ".env",
    ]
    for path in roots:
        if not path.exists():
            continue
        for raw in path.read_text(encoding="utf-8").splitlines():
            line = raw.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            key, value = key.strip(), value.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = value


load_dotenv()


@lru_cache(maxsize=1)
def load_config() -> dict:
    return json.loads(CONFIG_PATH.read_text(encoding="utf-8"))


def resolve_key(ref: str) -> str:
    if not ref:
        return ""
    if ref.startswith("env:"):
        for name in ref[4:].split("|"):
            val = os.environ.get(name.strip(), "").strip()
            if val:
                return val
    return ""


def providers() -> list[dict]:
    return list(load_config().get("providers") or [])


def models() -> list[dict]:
    return list(load_config().get("models") or [])


def profiles() -> list[dict]:
    return [p for p in load_config().get("profiles") or [] if p.get("enabled", True)]


def get_provider(code: str) -> dict | None:
    return next((p for p in providers() if p["code"] == code), None)


def get_model(code: str) -> dict | None:
    return next((m for m in models() if m["code"] == code), None)


def get_profile(code: str | None) -> dict:
    items = profiles()
    if code:
        found = next((p for p in items if p["code"] == code), None)
        if found:
            return found
    return next((p for p in items if p["code"] == "stock_analysis_default"), items[0])


def model_callable(model: dict) -> bool:
    from .quota import is_exhausted

    return model_ready(model) and not is_exhausted(model["code"])


def model_ready(model: dict) -> bool:
    if not model.get("enabled", True):
        return False
    provider = get_provider(model["providerCode"])
    if not provider or not provider.get("enabled", True):
        return False
    if provider.get("kind") == "local":
        return True
    return bool(resolve_key(provider.get("apiKeyRef") or ""))


def list_models() -> list[dict]:
    out = []
    for m in models():
        provider = get_provider(m["providerCode"]) or {}
        out.append(
            {
                "code": m["code"],
                "name": m.get("modelName") or m["code"],
                "providerCode": m["providerCode"],
                "roles": m.get("roles") or [],
                "costTier": m.get("costTier"),
                "enabled": bool(m.get("enabled", True) and provider.get("enabled", True)),
                "ready": model_ready(m),
                **quota_status(m["code"]),
            }
        )
    return out


def list_profiles() -> list[dict]:
    out = []
    for p in profiles():
        ready_models = [x["modelCode"] for x in p.get("models") or [] if (m := get_model(x["modelCode"])) and model_ready(m)]
        out.append(
            {
                "code": p["code"],
                "name": p.get("name") or p["code"],
                "task": p.get("task"),
                "mode": p.get("mode"),
                "ready": bool(ready_models),
                "readyModels": ready_models,
            }
        )
    return out
