"""ASR 服务配置加载器。

优先级: 环境变量 > config.yaml > 内置默认值
路径: 默认读取 ./config.yaml,可通过 ASR_CONFIG_PATH 覆盖。
若文件不存在,只读环境变量 + 默认值,启动不报错。
"""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass, field
from typing import Any

logger = logging.getLogger(__name__)

CONFIG_FILE_ENV = "ASR_CONFIG_PATH"
DEFAULT_CONFIG_PATH = "config.yaml"


def _env_bool(name: str, default: bool) -> bool:
    raw = os.getenv(name)
    if raw is None:
        return default
    return raw.strip().lower() in ("1", "true", "yes", "on")


def _env_int(name: str, default: int) -> int:
    raw = os.getenv(name)
    if not raw or not raw.strip():
        return default
    try:
        return int(raw.strip())
    except ValueError:
        logger.warning("环境变量 %s 不是合法整数(%s),使用默认 %s", name, raw, default)
        return default


def _env_str(name: str, default: str) -> str:
    raw = os.getenv(name)
    if raw is None:
        return default
    return raw.strip()


@dataclass
class ASRConfig:
    model: str = "small"
    device: str = "cpu"
    compute_type: str = "int8"
    language: str = "zh"
    beam_size: int = 5
    vad_filter: bool = True


@dataclass
class DiarizeConfig:
    enabled: bool = False
    model: str = "pyannote/speaker-diarization-3.1"
    hf_token: str = ""
    min_speakers: int = 0
    max_speakers: int = 0
    fallback_on_no_gpu: bool = True


@dataclass
class ServiceConfig:
    asr: ASRConfig = field(default_factory=ASRConfig)
    diarize: DiarizeConfig = field(default_factory=DiarizeConfig)


def _load_yaml(path: str) -> dict[str, Any]:
    if not os.path.isfile(path):
        logger.info("配置文件 %s 不存在,仅使用环境变量与默认配置", path)
        return {}
    try:
        import yaml  # type: ignore
    except ImportError:
        logger.warning("PyYAML 未安装,跳过读取 %s", path)
        return {}
    try:
        with open(path, "r", encoding="utf-8") as fh:
            data = yaml.safe_load(fh) or {}
        if not isinstance(data, dict):
            logger.warning("配置文件 %s 顶层不是 mapping,已忽略", path)
            return {}
        return data
    except Exception as exc:
        logger.exception("读取配置文件 %s 失败: %s", path, exc)
        return {}


def _pick_str(yaml_val: Any, env_name: str, default: str) -> str:
    if yaml_val is not None and str(yaml_val).strip() != "":
        return str(yaml_val).strip()
    return _env_str(env_name, default)


def _pick_int(yaml_val: Any, env_name: str, default: int) -> int:
    if yaml_val is not None and yaml_val != "":
        try:
            return int(yaml_val)
        except (TypeError, ValueError):
            logger.warning("配置项 %s=%r 不是合法整数,回落环境/默认", env_name, yaml_val)
    return _env_int(env_name, default)


def _pick_bool(yaml_val: Any, env_name: str, default: bool) -> bool:
    if yaml_val is not None and yaml_val != "":
        if isinstance(yaml_val, bool):
            return yaml_val
        if isinstance(yaml_val, str):
            return yaml_val.strip().lower() in ("1", "true", "yes", "on")
        return bool(yaml_val)
    return _env_bool(env_name, default)


def load_config() -> ServiceConfig:
    """加载配置: yaml(若存在)→ env → 默认。"""
    path = _env_str(CONFIG_FILE_ENV, DEFAULT_CONFIG_PATH)
    raw = _load_yaml(path)
    asr_raw = raw.get("asr", {}) if isinstance(raw.get("asr"), dict) else {}
    diar_raw = raw.get("diarize", {}) if isinstance(raw.get("diarize"), dict) else {}

    asr = ASRConfig(
        model=_pick_str(asr_raw.get("model"), "ASR_MODEL", "small"),
        device=_pick_str(asr_raw.get("device"), "ASR_DEVICE", "cpu").lower(),
        compute_type=_pick_str(asr_raw.get("compute_type"), "ASR_COMPUTE_TYPE", "int8"),
        language=_pick_str(asr_raw.get("language"), "ASR_LANGUAGE", "zh"),
        beam_size=_pick_int(asr_raw.get("beam_size"), "ASR_BEAM_SIZE", 5),
        vad_filter=_pick_bool(asr_raw.get("vad_filter"), "ASR_VAD_FILTER", True),
    )

    diarize = DiarizeConfig(
        enabled=_pick_bool(diar_raw.get("enabled"), "ASR_DIARIZE_ENABLED", False),
        model=_pick_str(
            diar_raw.get("model"),
            "ASR_DIARIZE_MODEL",
            "pyannote/speaker-diarization-3.1",
        ),
        hf_token=_env_str("HF_TOKEN", str(diar_raw.get("hf_token") or "").strip()),
        min_speakers=_pick_int(diar_raw.get("min_speakers"), "ASR_DIARIZE_MIN_SPEAKERS", 0),
        max_speakers=_pick_int(diar_raw.get("max_speakers"), "ASR_DIARIZE_MAX_SPEAKERS", 0),
        fallback_on_no_gpu=_pick_bool(
            diar_raw.get("fallback_on_no_gpu"),
            "ASR_DIARIZE_FALLBACK_NO_GPU",
            True,
        ),
    )

    return ServiceConfig(asr=asr, diarize=diarize)
