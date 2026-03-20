#!/usr/bin/env python3
"""
edge-tts HTTP service
Provides a simple REST API for text-to-speech synthesis using edge-tts.
Runs on 127.0.0.1:8083 inside the container.
"""

import os
import uuid
from pathlib import Path

import edge_tts
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

app = FastAPI(title="TTS Service", docs_url=None, redoc_url=None)

UPLOAD_DIR = Path(os.environ.get("TTS_OUTPUT_DIR", "/app/data/uploads"))
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)


class TTSRequest(BaseModel):
    text: str
    voice: str = "zh-CN-XiaoxiaoNeural"


@app.post("/tts")
async def synthesize(req: TTSRequest):
    if not req.text or not req.text.strip():
        raise HTTPException(status_code=400, detail="text is empty")

    text = req.text.strip()
    if len(text) > 500:
        text = text[:500]

    filename = f"tts_{uuid.uuid4().hex[:16]}.mp3"
    output_path = UPLOAD_DIR / filename

    communicate = edge_tts.Communicate(text, req.voice)
    try:
        await communicate.save(str(output_path))
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"TTS synthesis failed: {e}")

    if not output_path.exists() or output_path.stat().st_size == 0:
        raise HTTPException(status_code=500, detail="TTS produced no audio")

    return {"url": f"/api/chat/uploads/{filename}"}


@app.get("/health")
async def health():
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="127.0.0.1", port=8083, log_level="warning")
