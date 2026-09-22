from fastapi import FastAPI, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field

from agent.daily_note import daily_note
from agent.research import analyze_stock, analyze_stock_events
from llm.chat import chat_reply
from llm.config import list_models, list_profiles
from llm.screening_nl import parse_screening_nl
from prompts.loader import list_agents
from quant.algorithms.registry import list_algorithms
from quant.screening.engine import run_screening

import json

app = FastAPI(title="AI Stock Quant", version="0.3.0")


class ScreenRequest(BaseModel):
    detail: int = Field(default=80, ge=20, le=220)
    limit: int = Field(default=30, ge=5, le=60)
    algorithms: list[str] | None = None
    weights: list[dict] | None = None
    symbols: list[str] | None = None
    board: str | None = None
    industry: str | None = None


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
    newsTitles: list[str] | None = None
    noticeTitles: list[str] | None = None
    hotTopics: list[str] | None = None
    quote: dict | None = None


class ChatTurn(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    message: str
    messages: list[ChatTurn] | None = None
    modelCode: str | None = None
    memoryContext: str | None = None


class ScreeningNLRequest(BaseModel):
    query: str
    boards: list[dict] | None = None
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
        extras = {
            "newsTitles": req.newsTitles or [],
            "noticeTitles": req.noticeTitles or [],
            "hotTopics": req.hotTopics or [],
            "quote": req.quote,
        }
        return analyze_stock(req.symbol, req.profileCode, req.modelCode, extras)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@app.post("/v1/ai/analyze/stream")
def analyze_stream(req: AnalyzeRequest):
    """NDJSON 进度流：progress... → result。"""

    def gen():
        extras = {
            "newsTitles": req.newsTitles or [],
            "noticeTitles": req.noticeTitles or [],
            "hotTopics": req.hotTopics or [],
            "quote": req.quote,
        }
        try:
            for ev in analyze_stock_events(req.symbol, req.profileCode, req.modelCode, extras):
                yield json.dumps(ev, ensure_ascii=False) + "\n"
        except Exception as exc:
            yield json.dumps({"event": "error", "message": str(exc)}, ensure_ascii=False) + "\n"

    return StreamingResponse(gen(), media_type="application/x-ndjson")


@app.post("/v1/ai/chat")
def chat(req: ChatRequest):
    try:
        history = [t.model_dump() for t in req.messages or []]
        return chat_reply(req.message, history, req.modelCode, req.memoryContext)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@app.post("/v1/ai/screening-nl")
def screening_nl(req: ScreeningNLRequest):
    try:
        return parse_screening_nl(req.query, req.boards, req.modelCode)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@app.get("/v1/ai/daily-note")
def note(symbol: str, force: bool = False):
    try:
        return daily_note(symbol, force)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc
