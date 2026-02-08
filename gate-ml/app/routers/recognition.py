"""
Recognition API Router
"""
import base64
import logging
import time
from typing import Optional

from fastapi import APIRouter, HTTPException, status, Depends
from pydantic import BaseModel, Field, field_validator

from app.main import get_anpr_model
from app.models.anpr import ANPRModel
from app.config import settings

logger = logging.getLogger(__name__)

router = APIRouter()


class RecognitionRequest(BaseModel):
    """Запрос на распознавание номера"""
    image_base64: str = Field(..., description="Base64 encoded image")
    min_confidence: Optional[float] = Field(0.7, ge=0.0, le=1.0, description="Minimum confidence threshold")

    @field_validator('image_base64')
    def validate_image_base64(cls, v):
        """Валидация base64 строки"""
        try:
            decoded = base64.b64decode(v)
            if len(decoded) > settings.MAX_IMAGE_SIZE:
                raise ValueError(f"Image size exceeds maximum allowed size ({settings.MAX_IMAGE_SIZE} bytes)")
            return v
        except Exception as e:
            raise ValueError(f"Invalid base64 image: {str(e)}")


class BoundingBox(BaseModel):
    """Координаты bounding box"""
    x: int
    y: int
    width: int
    height: int


class RecognitionResponse(BaseModel):
    """Ответ на запрос распознавания"""
    success: bool
    license_plate: Optional[str] = None
    confidence: Optional[float] = None
    bounding_box: Optional[BoundingBox] = None
    processing_time_ms: float
    error: Optional[str] = None


@router.post("/recognize", response_model=RecognitionResponse, status_code=status.HTTP_200_OK)
async def recognize_plate(
    request: RecognitionRequest,
    anpr_model: ANPRModel = Depends(get_anpr_model)
):
    """
    Распознавание номера автомобиля на изображении

    Args:
        request: Запрос с изображением в base64
        anpr_model: Модель ANPR (внедряется через DI)

    Returns:
        Результат распознавания с номером, confidence и координатами
    """
    start_time = time.time()

    try:
        # Декодируем base64
        image_data = base64.b64decode(request.image_base64)

        # Распознаем номер
        result = anpr_model.recognize(image_data)

        processing_time = (time.time() - start_time) * 1000

        if result is None:
            return RecognitionResponse(
                success=False,
                processing_time_ms=processing_time,
                error="No license plate detected or confidence too low"
            )

        license_plate, confidence, bbox = result

        # Проверяем минимальный порог confidence
        if confidence < request.min_confidence:
            return RecognitionResponse(
                success=False,
                processing_time_ms=processing_time,
                error=f"Confidence {confidence:.2f} below threshold {request.min_confidence}"
            )

        return RecognitionResponse(
            success=True,
            license_plate=license_plate,
            confidence=round(confidence, 4),
            bounding_box=BoundingBox(**bbox) if bbox else None,
            processing_time_ms=round(processing_time, 2)
        )

    except base64.binascii.Error as e:
        logger.error(f"Base64 decode error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid base64 image data"
        )
    except Exception as e:
        logger.error(f"Recognition error: {str(e)}", exc_info=True)
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Recognition failed: {str(e)}"
        )


@router.get("/status", status_code=status.HTTP_200_OK)
async def get_status():
    """Статус ML сервиса"""
    return {
        "status": "running",
        "model": "EasyOCR",
        "languages": ["en", "ru"],
        "gpu": settings.USE_GPU
    }
