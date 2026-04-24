import logging
import os
import tempfile
from contextlib import asynccontextmanager

from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from faster_whisper import WhisperModel
from pydantic import BaseModel

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

model: WhisperModel | None = None
model_name = os.getenv("ASR_MODEL", "small")
device = os.getenv("ASR_DEVICE", "cpu")
compute_type = os.getenv("ASR_COMPUTE_TYPE", "int8")
beam_size = int(os.getenv("ASR_BEAM_SIZE", "5"))
default_language = os.getenv("ASR_LANGUAGE", "zh").strip() or None


class ASRSegment(BaseModel):
    start: float
    end: float
    text: str


class ASRResponse(BaseModel):
    text: str
    language: str | None = None
    status: str = "completed"
    error: str = ""
    segments: list[ASRSegment] = []


@asynccontextmanager
async def lifespan(app: FastAPI):
    global model
    logger.info("加载 ASR 模型: %s, device=%s, compute_type=%s", model_name, device, compute_type)
    model = WhisperModel(model_name, device=device, compute_type=compute_type)
    # 预热模型，避免首次请求长时间阻塞
    with tempfile.NamedTemporaryFile(suffix=".wav") as tmp:
        try:
            model.transcribe(tmp.name, language=default_language, beam_size=1)
        except Exception:
            logger.info("ASR 预热完成")
    yield


app = FastAPI(title="ASR Service", version="1.0.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.post("/transcribe", response_model=ASRResponse)
async def transcribe(
    file: UploadFile = File(...),
    language: str | None = Form(default=None),
):
    if model is None:
        raise HTTPException(status_code=503, detail="model not ready")

    suffix = os.path.splitext(file.filename or "audio.webm")[1] or ".webm"
    temp_path = ""
    try:
        with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
            temp_path = tmp.name
            while True:
                chunk = await file.read(1024 * 1024)
                if not chunk:
                    break
                tmp.write(chunk)

        segments_iter, info = model.transcribe(
            temp_path,
            language=(language or default_language),
            beam_size=beam_size,
            vad_filter=True,
            condition_on_previous_text=False,
        )
        segments = [
            ASRSegment(
                start=float(segment.start),
                end=float(segment.end),
                text=segment.text.strip(),
            )
            for segment in segments_iter
            if segment.text.strip()
        ]
        text = "\n".join(segment.text for segment in segments).strip()
        return ASRResponse(
            text=text,
            language=getattr(info, "language", None),
            segments=segments,
        )
    except Exception as exc:
        logger.exception("ASR 转写失败")
        raise HTTPException(status_code=500, detail=str(exc))
    finally:
        try:
            await file.close()
        except Exception:
            pass
        if temp_path and os.path.exists(temp_path):
            os.remove(temp_path)


@app.get("/health")
async def health():
    if model is None:
        raise HTTPException(status_code=503, detail="warming up")
    return {
        "status": "ok",
        "model": model_name,
        "device": device,
        "compute_type": compute_type,
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=9000)
