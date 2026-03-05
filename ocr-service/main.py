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
import cv2
from pyzbar.pyzbar import decode as decode_qr

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


class QRCodeResult(BaseModel):
    data: str
    type: str
    rect: dict


class OCRResponse(BaseModel):
    text: str
    lines: list[OCRLine]
    qr_codes: list[QRCodeResult] = []


def detect_qr_codes(pil_image: Image.Image) -> list[dict]:
    """使用 OpenCV 和 pyzbar 检测图片中的二维码"""
    qr_codes = []

    # 转换为 OpenCV 格式
    cv_image = cv2.cvtColor(np.array(pil_image), cv2.COLOR_RGB2BGR)

    # 使用 pyzbar 检测二维码
    try:
        decoded_objects = decode_qr(cv_image)
        for obj in decoded_objects:
            qr_codes.append({
                "data": obj.data.decode('utf-8') if isinstance(obj.data, bytes) else str(obj.data),
                "type": obj.type,
                "rect": {
                    "left": int(obj.rect.left),
                    "top": int(obj.rect.top),
                    "width": int(obj.rect.width),
                    "height": int(obj.rect.height)
                }
            })
    except Exception as e:
        logger.warning(f"pyzbar detection failed: {e}")

    # 如果 pyzbar 没检测到，尝试 OpenCV 的 QRCodeDetector
    if not qr_codes:
        try:
            detector = cv2.QRCodeDetector()
            retval, decoded_info, points, straight_qrcode = detector.detectAndDecodeMulti(cv_image)
            if retval and decoded_info:
                for i, data in enumerate(decoded_info):
                    if data:
                        pts = points[i] if points is not None else None
                        rect = {}
                        if pts is not None and len(pts) == 4:
                            xs = [p[0] for p in pts]
                            ys = [p[1] for p in pts]
                            rect = {
                                "left": int(min(xs)),
                                "top": int(min(ys)),
                                "width": int(max(xs) - min(xs)),
                                "height": int(max(ys) - min(ys))
                            }
                        qr_codes.append({
                            "data": data,
                            "type": "QRCODE",
                            "rect": rect
                        })
        except Exception as e:
            logger.warning(f"OpenCV QR detection failed: {e}")

    return qr_codes


@app.post("/ocr", response_model=OCRResponse)
async def ocr(request: OCRRequest):
    try:
        image_b64 = request.image
        if "," in image_b64:
            image_b64 = image_b64.split(",", 1)[1]

        image_data = base64.b64decode(image_b64)
        image = Image.open(io.BytesIO(image_data)).convert("RGB")

        # 检测二维码
        qr_codes = detect_qr_codes(image)

        # OCR 文字识别
        result, _ = engine(image)

        if not result and not qr_codes:
            return OCRResponse(text="", lines=[], qr_codes=[])

        lines = [OCRLine(text=item[1], confidence=float(item[2])) for item in result] if result else []
        full_text = "\n".join(line.text for line in lines)

        # 如果检测到二维码，添加到结果中
        qr_data_list = [qr["data"] for qr in qr_codes]

        return OCRResponse(
            text=full_text,
            lines=lines,
            qr_codes=qr_codes
        )

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
