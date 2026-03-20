#!/usr/bin/env python3
"""
Kokoro-ONNX TTS HTTP service (fully offline, no external API needed)
Runs on 127.0.0.1:8083 inside the container.

Voices mapping (kokoro v1.0 Chinese voices):
  zh_f_xiaobei  - 小北 (female, young)
  zh_f_xiaoni   - 小妮 (female, warm)
  zh_m_yunjian  - 云见 (male, deep)
  zh_m_yunxi    - 云希 (male, young)
"""

import os
import uuid
from pathlib import Path

import numpy as np
import soundfile as sf
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from kokoro_onnx import Kokoro

MODEL_DIR = Path(os.environ.get("TTS_MODEL_DIR", "/app/tts-models"))
UPLOAD_DIR = Path(os.environ.get("TTS_OUTPUT_DIR", "/app/data/uploads"))
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)

app = FastAPI(title="TTS Service", docs_url=None, redoc_url=None)

# 全局模型实例，启动时加载一次
_kokoro: Kokoro | None = None


def get_kokoro() -> Kokoro:
    global _kokoro
    if _kokoro is None:
        model_path = MODEL_DIR / "kokoro-v1.0.onnx"
        voices_path = MODEL_DIR / "voices-v1.0.bin"
        if not model_path.exists() or not voices_path.exists():
            raise RuntimeError(f"TTS model files not found in {MODEL_DIR}")
        _kokoro = Kokoro(str(model_path), str(voices_path))
    return _kokoro


# edge-tts voice name → kokoro voice name 映射
VOICE_MAP = {
    "zh-CN-XiaoxiaoNeural": "zh_f_xiaobei",
    "zh-CN-XiaoyiNeural":   "zh_f_xiaoni",
    "zh-CN-YunxiNeural":    "zh_m_yunxi",
    "zh-CN-YunyangNeural":  "zh_m_yunjian",
    "zh-CN-XiaomouNeural":  "zh_f_xiaobei",  # fallback
}

DEFAULT_VOICE = "zh_f_xiaobei"


class TTSRequest(BaseModel):
    text: str
    voice: str = "zh-CN-XiaoxiaoNeural"


@app.post("/tts")
def synthesize(req: TTSRequest):
    if not req.text or not req.text.strip():
        raise HTTPException(status_code=400, detail="text is empty")

    text = req.text.strip()
    if len(text) > 500:
        text = text[:500]

    # 映射 edge-tts voice name 到 kokoro voice name
    voice = VOICE_MAP.get(req.voice, DEFAULT_VOICE)

    try:
        kokoro = get_kokoro()
        samples, sample_rate = kokoro.create(text, voice=voice, speed=1.0, lang="zh")
    except RuntimeError as e:
        raise HTTPException(status_code=503, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"TTS synthesis failed: {e}")

    filename = f"tts_{uuid.uuid4().hex[:16]}.wav"
    output_path = UPLOAD_DIR / filename
    sf.write(str(output_path), samples, sample_rate)

    if not output_path.exists() or output_path.stat().st_size == 0:
        raise HTTPException(status_code=500, detail="TTS produced no audio")

    return {"url": f"/api/chat/uploads/{filename}"}


@app.get("/health")
def health():
    try:
        get_kokoro()
        return {"status": "ok"}
    except RuntimeError as e:
        raise HTTPException(status_code=503, detail=str(e))


if __name__ == "__main__":
    import uvicorn
    # 预加载模型
    try:
        get_kokoro()
        print("[tts] Kokoro model loaded successfully")
    except Exception as e:
        print(f"[tts] WARNING: model load failed: {e}")
    uvicorn.run(app, host="127.0.0.1", port=8083, log_level="warning")
