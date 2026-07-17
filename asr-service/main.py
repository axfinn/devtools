"""ASR 服务: faster-whisper 转写。

说话人识别(diarization)已拆出到独立服务,见同级 ../diarize-service/。
本服务只做转写,diarize_status 字段恒为 "disabled",保持调用方解析逻辑不变。
"""

from __future__ import annotations

import logging
import os
import tempfile
from contextlib import asynccontextmanager

from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from faster_whisper import WhisperModel
from pydantic import BaseModel

from config_loader import ASRConfig, ServiceConfig, load_config

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# ── 全局状态(由 lifespan 初始化) ─────────────────────────
config: ServiceConfig = ServiceConfig()
whisper_model: WhisperModel | None = None
gpu_available: bool = False
effective_device: str = "cpu"  # 启动时若 cuda 不可用会被降级


# ── GPU 可用性检测 ────────────────────────────────────────


def _detect_gpu() -> bool:
    """检测 GPU 是否可用于 ASR/CTranslate2 推理。

    优先用 ctranslate2(ASR 实际依赖)的 CUDA 检测;若 ctranslate2 不在
    (极少见),回退到 torch。返回 True 表示有可用 GPU。
    """
    try:
        import ctranslate2  # type: ignore

        return bool(ctranslate2.contains_cuda("cuda")) or bool(
            ctranslate2.contains_cuda("cuda-int8")
        )
    except Exception:
        pass
    try:
        import torch  # type: ignore

        return bool(torch.cuda.is_available())
    except Exception:
        return False


# ── Pydantic 模型 ─────────────────────────────────────────


class ASRSegment(BaseModel):
    start: float
    end: float
    text: str
    # speaker 字段保留但不再由本服务填充;若调用方启用了 diarize-service,
    # 会在 Go 层按中点合并后填入。
    speaker: str | None = None


class ASRResponse(BaseModel):
    text: str
    language: str | None = None
    status: str = "completed"
    error: str = ""
    segments: list[ASRSegment] = []
    # 恒为 "disabled";diarization 已拆到独立服务(diarize-service),
    # 由 devtools 后端按 DIARIZE_SERVICE_URL 配置决定是否调。
    diarize_status: str = "disabled"


# ── Whisper 模型加载 ──────────────────────────────────────


def _load_whisper(cfg: ServiceConfig, device: str) -> WhisperModel:
    logger.info(
        "加载 Whisper 模型: %s, device=%s, compute_type=%s",
        cfg.asr.model,
        device,
        cfg.asr.compute_type,
    )
    return WhisperModel(cfg.asr.model, device=device, compute_type=cfg.asr.compute_type)


# ── 启动 / 关闭 ──────────────────────────────────────────


@asynccontextmanager
async def lifespan(app: FastAPI):
    global config, whisper_model, gpu_available, effective_device

    config = load_config()

    # GPU 检测
    gpu_available = _detect_gpu()
    effective_device = config.asr.device
    if effective_device == "cuda" and not gpu_available:
        logger.warning("配置要求 device=cuda 但未检测到可用 GPU,降级为 cpu")
        effective_device = "cpu"

    # Whisper
    try:
        whisper_model = _load_whisper(config, effective_device)
    except Exception:
        logger.exception("Whisper 模型加载失败")
        raise

    # 预热(只跑一次很短的转写,避免首次请求长时间阻塞)
    with tempfile.NamedTemporaryFile(suffix=".wav") as tmp:
        try:
            whisper_model.transcribe(
                tmp.name,
                language=(config.asr.language or None),
                beam_size=1,
            )
            logger.info("ASR 预热完成")
        except Exception:
            logger.info("ASR 预热跳过(临时文件不可用)")

    logger.info(
        "ASR 服务就绪: device=%s, gpu_available=%s",
        effective_device,
        gpu_available,
    )
    yield
    whisper_model = None


app = FastAPI(title="ASR Service", version="3.0.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


# ── 路由 ──────────────────────────────────────────────────


@app.get("/health")
async def health():
    if whisper_model is None:
        raise HTTPException(status_code=503, detail="warming up")
    return {
        "status": "ok",
        "model": config.asr.model,
        "device": effective_device,
        "requested_device": config.asr.device,
        "compute_type": config.asr.compute_type,
        "gpu_available": gpu_available,
        "diarize_enabled": False,
        "diarize_service_url": os.getenv("DIARIZE_SERVICE_URL", ""),
    }


@app.post("/transcribe", response_model=ASRResponse)
async def transcribe(
    file: UploadFile = File(...),
    language: str | None = Form(default=None),
):
    if whisper_model is None:
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

        lang = (language or config.asr.language or None)
        segs_iter, info = whisper_model.transcribe(
            temp_path,
            language=lang,
            beam_size=config.asr.beam_size,
            vad_filter=config.asr.vad_filter,
            condition_on_previous_text=False,
        )
        whisper_segs = [s for s in segs_iter if s.text.strip()]

        segments = [
            ASRSegment(start=float(s.start), end=float(s.end), text=s.text.strip())
            for s in whisper_segs
        ]

        text = "\n".join(s.text for s in segments).strip()
        return ASRResponse(
            text=text,
            language=getattr(info, "language", None),
            segments=segments,
            diarize_status="disabled",
        )
    except HTTPException:
        raise
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


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=9000)
