import base64
import io
import logging
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from PIL import Image
from rapidocr_onnxruntime import RapidOCR

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="OCR Service", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

engine = RapidOCR()


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
        # 去掉 data URL 前缀（如 data:image/png;base64,）
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
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
