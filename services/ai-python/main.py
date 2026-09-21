from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from agent.daily_note import daily_note
from agent.research import analyze_stock
from llm.config import list_models, list_profiles
from prompts.loader import list_agents
from quant.algorithms.registry import list_algorithms
from quant.screening.engine import run_screening

app = FastAPI(title="AI Stock Quant", version="0.3.0")


class ScreenRequest(BaseModel):
    detail: int = Field(default=80, ge=20, le=220)
    limit: int = Field(default=30, ge=5, le=60)
    algorithms: list[str] | None = None
    weights: list[dict] | None = None


@app.get("/health")
def health():
    return {"ok": True, "service": "ai-python"}


@app.get("/v1/algorithms")
def algorithms():
    return {"items": list_algorithms()}


@app.post("/v1/screening/run")
def screening(req: ScreenRequest):
    return run_screening(req.model_dump())


class AnalyzeRequest(BaseModel):
    symbol: str
    profileCode: str | None = None
    modelCode: str | None = None


@app.get("/v1/llm/models")
def llm_models():
    return {"items": list_models()}


@app.get("/v1/llm/profiles")
def llm_profiles():
    return {"items": list_profiles()}


@app.get("/v1/agents")
def agents():
    return {"items": list_agents()}


@app.post("/v1/ai/analyze")
def analyze(req: AnalyzeRequest):
    try:
        return analyze_stock(req.symbol, req.profileCode, req.modelCode)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@app.get("/v1/ai/daily-note")
def note(symbol: str, force: bool = False):
    try:
        return daily_note(symbol, force)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc
