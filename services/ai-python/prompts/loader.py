from __future__ import annotations

import json
import re
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path

ROOT = Path(__file__).resolve().parent
PLACEHOLDER = re.compile(r"\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}")


@dataclass
class AgentPrompt:
    code: str
    name: str
    task: str
    role: str
    version: str
    enabled: bool
    description: str
    system: str
    user: str

    def meta(self) -> dict:
        return {
            "code": self.code,
            "name": self.name,
            "task": self.task,
            "role": self.role,
            "version": self.version,
            "enabled": self.enabled,
            "description": self.description,
        }


def list_agents() -> list[dict]:
    return [load_agent(code).meta() for code in _agent_codes()]


def get_agent(code: str | None) -> AgentPrompt:
    codes = _agent_codes()
    if not codes:
        raise RuntimeError("prompts 目录下没有可用 Agent")
    if code and code in codes:
        return load_agent(code)
    if "stock_analyst" in codes:
        return load_agent("stock_analyst")
    return load_agent(codes[0])


@lru_cache(maxsize=32)
def load_agent(code: str) -> AgentPrompt:
    folder = ROOT / code
    meta_path = folder / "agent.json"
    if not meta_path.exists():
        raise RuntimeError(f"Agent 不存在：{code}")
    meta = json.loads(meta_path.read_text(encoding="utf-8"))
    system = (folder / "system.md").read_text(encoding="utf-8").strip()
    user = (folder / "user.md").read_text(encoding="utf-8").strip()
    return AgentPrompt(
        code=meta.get("code") or code,
        name=meta.get("name") or code,
        task=meta.get("task") or "",
        role=meta.get("role") or "analyst",
        version=str(meta.get("version") or "1.0"),
        enabled=bool(meta.get("enabled", True)),
        description=meta.get("description") or "",
        system=system,
        user=user,
    )


def render_messages(agent_code: str | None, context: dict) -> list[dict]:
    agent = get_agent(agent_code)
    vars_ = context_vars(context)
    return [
        {"role": "system", "content": render(agent.system, vars_)},
        {"role": "user", "content": render(agent.user, vars_)[:8000]},
    ]


def render(template: str, vars_: dict) -> str:
    def repl(match: re.Match[str]) -> str:
        return _lookup(vars_, match.group(1))

    return PLACEHOLDER.sub(repl, template)


def context_vars(context: dict) -> dict:
    stock = context.get("stock") or {}
    indicators = context.get("indicators") or {}
    hits = context.get("algorithmHits") or []
    lines = []
    for hit in hits:
        flag = "命中" if hit.get("pass") else "未命中"
        name = hit.get("short") or hit.get("name") or hit.get("algorithmCode")
        lines.append(f"- {name} {flag}：{hit.get('reason') or ''}")
    slim_keys = ("stock", "indicators", "rps", "algorithmHits", "sector", "attribution")
    slim = {key: context.get(key) for key in slim_keys if key in context}
    out = {
        "stock": stock,
        "indicators": indicators,
        "rps": context.get("rps") or {},
        "algorithm_lines": "\n".join(lines) or "- 无算法结果",
        "sector_lines": str(context.get("sector_lines") or "暂无板块/归因数据"),
        "context_json": json.dumps(slim, ensure_ascii=False)[:6000],
        "query": str(context.get("query") or ""),
    }
    # 选股理解等场景可直接传入 context_json / query
    if context.get("context_json"):
        out["context_json"] = str(context["context_json"])[:8000]
    if context.get("query"):
        out["query"] = str(context["query"])
    return out


def _agent_codes() -> list[str]:
    codes: list[str] = []
    for path in sorted(ROOT.iterdir()):
        if not path.is_dir() or not (path / "agent.json").exists() or not (path / "system.md").exists():
            continue
        meta = json.loads((path / "agent.json").read_text(encoding="utf-8"))
        if meta.get("enabled", True):
            codes.append(str(meta.get("code") or path.name))
    return codes


def _lookup(vars_: dict, path: str) -> str:
    current: object = vars_
    for part in path.split("."):
        if isinstance(current, dict):
            current = current.get(part)
        else:
            return "-"
    if current is None or current == "":
        return "-"
    if isinstance(current, bool):
        return "是" if current else "否"
    return str(current)
