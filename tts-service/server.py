#!/usr/bin/env python3
"""
edge-tts HTTP service (fully cloud-based, Microsoft Neural TTS)
Runs on 127.0.0.1:8083 inside the container.

Supported Chinese voices (zh-CN):
  zh-CN-XiaoxiaoNeural   - 晓晓 (女, 温柔)
  zh-CN-XiaoyiNeural     - 晓伊 (女, 活泼)
  zh-CN-XiaohanNeural    - 晓涵 (女, 知性)
  zh-CN-XiaomouNeural    - 晓墨 (女, 戏剧)
  zh-CN-XiaoruiNeural    - 晓睿 (女, 老年)
  zh-CN-XiaoshuangNeural - 晓双 (女/童, 儿童)
  zh-CN-XiaoxuanNeural   - 晓萱 (女, 轻松)
  zh-CN-XiaoyanNeural    - 晓颜 (女)
  zh-CN-XiaozhenNeural   - 晓甄 (女, 激情)
  zh-CN-YunxiNeural      - 云希 (男, 阳光)
  zh-CN-YunjianNeural    - 云健 (男, 运动)
  zh-CN-YunyangNeural    - 云扬 (男, 新闻)
  zh-CN-YunzeNeural      - 云泽 (男, 老年)
"""

import os
import uuid
import asyncio
from pathlib import Path

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

try:
    import edge_tts
    EDGE_TTS_AVAILABLE = True
except ImportError:
    EDGE_TTS_AVAILABLE = False

UPLOAD_DIR = Path(os.environ.get("TTS_OUTPUT_DIR", "/app/data/uploads"))
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)

app = FastAPI(title="TTS Service", docs_url=None, redoc_url=None)

# 完整中文音色列表（角色 → voice name 映射用）
ZH_VOICES = [
    {"id": "zh-CN-XiaoxiaoNeural",   "name": "晓晓", "gender": "female", "style": "温柔"},
    {"id": "zh-CN-XiaoyiNeural",     "name": "晓伊", "gender": "female", "style": "活泼"},
    {"id": "zh-CN-XiaohanNeural",    "name": "晓涵", "gender": "female", "style": "知性"},
    {"id": "zh-CN-XiaomouNeural",    "name": "晓墨", "gender": "female", "style": "戏剧"},
    {"id": "zh-CN-XiaoruiNeural",    "name": "晓睿", "gender": "female", "style": "老年"},
    {"id": "zh-CN-XiaoshuangNeural", "name": "晓双", "gender": "female", "style": "儿童"},
    {"id": "zh-CN-XiaoxuanNeural",   "name": "晓萱", "gender": "female", "style": "轻松"},
    {"id": "zh-CN-XiaoyanNeural",    "name": "晓颜", "gender": "female", "style": ""},
    {"id": "zh-CN-XiaozhenNeural",   "name": "晓甄", "gender": "female", "style": "激情"},
    {"id": "zh-CN-YunxiNeural",      "name": "云希", "gender": "male",   "style": "阳光"},
    {"id": "zh-CN-YunjianNeural",    "name": "云健", "gender": "male",   "style": "运动"},
    {"id": "zh-CN-YunyangNeural",    "name": "云扬", "gender": "male",   "style": "新闻"},
    {"id": "zh-CN-YunzeNeural",      "name": "云泽", "gender": "male",   "style": "老年"},
]


class TTSRequest(BaseModel):
    text: str
    voice: str = "zh-CN-XiaoxiaoNeural"


@app.post("/tts")
async def synthesize(req: TTSRequest):
    if not EDGE_TTS_AVAILABLE:
        raise HTTPException(status_code=503, detail="edge-tts not installed")

    text = req.text.strip()
    if not text:
        raise HTTPException(status_code=400, detail="text is empty")
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


@app.get("/voices")
def list_voices():
    return {"voices": ZH_VOICES}


@app.get("/health")
def health():
    return {"status": "ok", "edge_tts": EDGE_TTS_AVAILABLE}


if __name__ == "__main__":
    import uvicorn
    print(f"[tts] edge-tts available: {EDGE_TTS_AVAILABLE}")
    uvicorn.run(app, host="127.0.0.1", port=8083, log_level="warning")
