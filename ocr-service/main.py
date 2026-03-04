import base64
import io
import logging
import numpy as np
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from PIL import Image
from rapidocr_onnxruntime import RapidOCR

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

engine: RapidOCR | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    global engine
    logger.info("加载 RapidOCR 模型...")
    engine = RapidOCR()
    # 用一张空白图预热，让 ONNX runtime 完成首次 JIT 编译
    dummy = Image.fromarray(np.full((64, 256, 3), 255, dtype=np.uint8))
    engine(dummy)
    logger.info("RapidOCR 预热完成，服务就绪")
    yield


app = FastAPI(title="OCR Service", version="1.0.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


class OCRRequest(BaseModel):
    image: str  # base64 编码的图片（支持 data:image/xxx;base64,... 前缀）


class OCRLine(BaseModel):
    text: str
    confidence: float


class OCRResponse(BaseModel):
    text: str
    lines: list[OCRLine]


@app.post("/ocr", response_model=OCRResponse)
async def ocr(request: OCRRequest):
    try:
        image_b64 = request.image
        if "," in image_b64:
            image_b64 = image_b64.split(",", 1)[1]

        image_data = base64.b64decode(image_b64)
        image = Image.open(io.BytesIO(image_data)).convert("RGB")

        result, _ = engine(image)

        if not result:
            return OCRResponse(text="", lines=[])

        lines = [OCRLine(text=item[1], confidence=float(item[2])) for item in result]
        full_text = "\n".join(line.text for line in lines)

        return OCRResponse(text=full_text, lines=lines)

    except Exception as e:
        logger.error(f"OCR error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/health")
async def health():
    # 预热完成前不对外声明就绪
    if engine is None:
        raise HTTPException(status_code=503, detail="warming up")
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
