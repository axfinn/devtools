"""ASR 服务: faster-whisper 转写 + 可选 pyannote 说话人识别。

向后兼容: 当 diarize 关闭或 GPU 不可用时,响应结构与 v1.0 一致
(仅 text / language / segments),仅在启用角色识别时新增 diarize_status 字段。
"""

from __future__ import annotations

import logging
import os
import tempfile
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from faster_whisper import WhisperModel
from pydantic import BaseModel

from config_loader import DiarizeConfig, ServiceConfig, load_config

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# ── 全局状态(由 lifespan 初始化) ─────────────────────────
config: ServiceConfig = ServiceConfig()
whisper_model: WhisperModel | None = None
diarize_pipeline: Any = None
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
    speaker: str | None = None


class ASRResponse(BaseModel):
    text: str
    language: str | None = None
    status: str = "completed"
    error: str = ""
    segments: list[ASRSegment] = []
    # 角色识别状态(向后兼容: 旧调用方忽略此字段)
    # 取值: "disabled" | "ok" | "skipped_no_gpu" | "failed"
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


# ── Diarization 懒加载 ────────────────────────────────────


def _load_diarize_pipeline(diar_cfg: DiarizeConfig):
    """懒加载 pyannote pipeline。失败抛出异常,调用方决定降级还是报错。"""
    if not diar_cfg.hf_token:
        raise RuntimeError(
            "diarize.enabled=true 但未配置 HF_TOKEN;请在 .env 设置 HF_TOKEN=hf_xxx"
        )
    from pyannote.audio import Pipeline  # type: ignore

    logger.info("加载 pyannote diarization pipeline: %s", diar_cfg.model)
    pipeline = Pipeline.from_pretrained(
        diar_cfg.model,
        use_auth_token=diar_cfg.hf_token,
    )
    try:
        import torch  # type: ignore

        if torch.cuda.is_available():
            pipeline.to(torch.device("cuda"))
            logger.info("pyannote pipeline 已迁移到 GPU")
    except Exception as exc:
        logger.warning("pyannote 迁移到 GPU 失败,将使用默认设备: %s", exc)
    return pipeline


# ── Speaker 合并 ─────────────────────────────────────────


def _assign_speakers(whisper_segs, diarize_annotation) -> list[ASRSegment]:
    """把 pyannote 的 (turn, _, speaker) 与 whisper segments 按中点时间对齐。"""
    if diarize_annotation is None:
        return [
            ASRSegment(start=s.start, end=s.end, text=s.text.strip())
            for s in whisper_segs
        ]
    out: list[ASRSegment] = []
    for seg in whisper_segs:
        mid = (seg.start + seg.end) / 2.0
        speaker: str | None = None
        try:
            for turn, _, label in diarize_annotation.itertracks(yield_label=True):
                if turn.start <= mid <= turn.end:
                    speaker = label
                    break
        except Exception:
            speaker = None
        out.append(
            ASRSegment(
                start=float(seg.start),
                end=float(seg.end),
                text=seg.text.strip(),
                speaker=speaker,
            )
        )
    return out


# ── 启动 / 关闭 ──────────────────────────────────────────


@asynccontextmanager
async def lifespan(app: FastAPI):
    global config, whisper_model, diarize_pipeline, gpu_available, effective_device

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

    # Diarization(可选,懒加载;失败不阻塞启动,首次请求时报错或降级)
    if config.diarize.enabled:
        try:
            diarize_pipeline = _load_diarize_pipeline(config.diarize)
        except Exception as exc:
            logger.error("Diarization pipeline 加载失败: %s", exc)
            diarize_pipeline = None

    logger.info(
        "ASR 服务就绪: device=%s, gpu_available=%s, diarize_enabled=%s, diarize_loaded=%s",
        effective_device,
        gpu_available,
        config.diarize.enabled,
        diarize_pipeline is not None,
    )
    yield
    whisper_model = None
    diarize_pipeline = None


app = FastAPI(title="ASR Service", version="2.0.0", lifespan=lifespan)

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
        "diarize_enabled": config.diarize.enabled,
        "diarize_loaded": diarize_pipeline is not None,
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

        # Diarization 分支
        diarize_status = "disabled"
        segments: list[ASRSegment]
        if not config.diarize.enabled:
            segments = [
                ASRSegment(start=float(s.start), end=float(s.end), text=s.text.strip())
                for s in whisper_segs
            ]
        elif diarize_pipeline is None:
            # 启用了但加载失败 → 走只转写
            logger.warning("diarize_enabled 但 pipeline 未加载,只返回转写")
            segments = [
                ASRSegment(start=float(s.start), end=float(s.end), text=s.text.strip())
                for s in whisper_segs
            ]
            diarize_status = "failed"
        else:
            # 启用了 + pipeline 在
            if not gpu_available and config.diarize.fallback_on_no_gpu:
                logger.warning("diarize 启用但无 GPU,自动降级为只转写")
                segments = [
                    ASRSegment(start=float(s.start), end=float(s.end), text=s.text.strip())
                    for s in whisper_segs
                ]
                diarize_status = "skipped_no_gpu"
            elif not gpu_available and not config.diarize.fallback_on_no_gpu:
                raise HTTPException(
                    status_code=503,
                    detail="diarize 启用但当前环境无 GPU,且 fallback_on_no_gpu=false",
                )
            else:
                # 实际跑 pyannote
                try:
                    diar_kwargs: dict[str, Any] = {}
                    if config.diarize.min_speakers > 0:
                        diar_kwargs["min_speakers"] = config.diarize.min_speakers
                    if config.diarize.max_speakers > 0:
                        diar_kwargs["max_speakers"] = config.diarize.max_speakers
                    annotation = diarize_pipeline(temp_path, **diar_kwargs)
                    segments = _assign_speakers(whisper_segs, annotation)
                    diarize_status = "ok"
                except Exception as exc:
                    logger.exception("diarize 推理失败,降级为只转写")
                    segments = [
                        ASRSegment(
                            start=float(s.start), end=float(s.end), text=s.text.strip()
                        )
                        for s in whisper_segs
                    ]
                    diarize_status = "failed"

        text = "\n".join(s.text for s in segments).strip()
        return ASRResponse(
            text=text,
            language=getattr(info, "language", None),
            segments=segments,
            diarize_status=diarize_status,
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
