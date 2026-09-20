from fastapi import FastAPI

app = FastAPI(title="AI Stock Quant", version="0.1.0")


@app.get("/health")
def health():
    return {"ok": True, "service": "ai-python"}
