#!/usr/bin/env python3
"""
Multi-engine TTS HTTP service
Supports: kokoro-onnx (offline) | edge-tts (cloud, Microsoft)
Runs on 127.0.0.1:8083 inside the container.
"""

import os
import uuid
import asyncio
import logging
from pathlib import Path

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

MODEL_DIR = Path(os.environ.get("TTS_MODEL_DIR", "/app/tts-models"))
UPLOAD_DIR = Path(os.environ.get("TTS_OUTPUT_DIR", "/app/data/uploads"))
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)

app = FastAPI(title="TTS Service", docs_url=None, redoc_url=None)

# ── kokoro engine ──────────────────────────────────────────────────────────────
_kokoro = None
_kokoro_available = False

def _init_kokoro():
    global _kokoro, _kokoro_available
    try:
        from kokoro_onnx import Kokoro
        model_path = MODEL_DIR / "kokoro-v1.0.onnx"
        voices_path = MODEL_DIR / "voices-v1.0.bin"
        if not model_path.exists() or not voices_path.exists():
            logger.warning(f"[kokoro] model files not found in {MODEL_DIR}, skipping")
            return
        _kokoro = Kokoro(str(model_path), str(voices_path))
        _kokoro_available = True
        logger.info("[kokoro] model loaded successfully")
    except Exception as e:
        logger.warning(f"[kokoro] init failed: {e}")

# edge-tts voice → kokoro voice
KOKORO_VOICE_MAP = {
    "zh-CN-XiaoxiaoNeural": "zh_f_xiaobei",
    "zh-CN-XiaoyiNeural":   "zh_f_xiaoni",
    "zh-CN-YunxiNeural":    "zh_m_yunxi",
    "zh-CN-YunyangNeural":  "zh_m_yunjian",
    "zh-CN-XiaomouNeural":  "zh_f_xiaobei",
}
KOKORO_DEFAULT_VOICE = "zh_f_xiaobei"

def _synth_kokoro(text: str, voice: str) -> Path:
    import soundfile as sf
    kokoro_voice = KOKORO_VOICE_MAP.get(voice, KOKORO_DEFAULT_VOICE)
    samples, sample_rate = _kokoro.create(text, voice=kokoro_voice, speed=1.0, lang="zh")
    filename = f"tts_{uuid.uuid4().hex[:16]}.wav"
    out = UPLOAD_DIR / filename
    sf.write(str(out), samples, sample_rate)
    return out

# ── edge-tts engine ────────────────────────────────────────────────────────────
_edge_tts_available = False

def _init_edge_tts():
    global _edge_tts_available
    try:
        import edge_tts  # noqa
        _edge_tts_available = True
        logger.info("[edge-tts] available")
    except ImportError:
        logger.warning("[edge-tts] not installed")

async def _synth_edge_tts(text: str, voice: str) -> Path:
    import edge_tts
    filename = f"tts_{uuid.uuid4().hex[:16]}.mp3"
    out = UPLOAD_DIR / filename
    communicate = edge_tts.Communicate(text, voice)
    await communicate.save(str(out))
    return out

# ── API ────────────────────────────────────────────────────────────────────────
class TTSRequest(BaseModel):
    text: str
    voice: str = "zh-CN-XiaoxiaoNeural"
    engine: str = "auto"   # "auto" | "kokoro" | "edge-tts"


@app.post("/tts")
async def synthesize(req: TTSRequest):
    text = req.text.strip()
    if not text:
        raise HTTPException(status_code=400, detail="text is empty")
    if len(text) > 500:
        text = text[:500]

    engine = req.engine
    # auto: 优先 kokoro（离线），fallback edge-tts
    if engine == "auto":
        engine = "kokoro" if _kokoro_available else "edge-tts"

    out: Path | None = None

    if engine == "kokoro":
        if not _kokoro_available:
            raise HTTPException(status_code=503, detail="kokoro engine not available (model files missing)")
        try:
            out = _synth_kokoro(text, req.voice)
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"kokoro synthesis failed: {e}")

    elif engine == "edge-tts":
        if not _edge_tts_available:
            raise HTTPException(status_code=503, detail="edge-tts engine not available (not installed)")
        try:
            out = await _synth_edge_tts(text, req.voice)
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"edge-tts synthesis failed: {e}")

    else:
        raise HTTPException(status_code=400, detail=f"unknown engine: {engine}")

    if not out or not out.exists() or out.stat().st_size == 0:
        raise HTTPException(status_code=500, detail="TTS produced no audio")

    return {"url": f"/api/chat/uploads/{out.name}", "engine": engine}


@app.get("/health")
def health():
    return {
        "status": "ok",
        "engines": {
            "kokoro": _kokoro_available,
            "edge-tts": _edge_tts_available,
        }
    }


if __name__ == "__main__":
    import uvicorn
    _init_kokoro()
    _init_edge_tts()
    uvicorn.run(app, host="127.0.0.1", port=8083, log_level="warning")
