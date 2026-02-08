"""
Gate ML Service - License Plate Recognition
FastAPI сервис для распознавания автомобильных номеров
"""
import logging
import time
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware

from app.config import settings
from app.models.anpr import ANPRModel
from app.routers import recognition

# Настройка логирования
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

# Глобальная переменная для модели
anpr_model: ANPRModel = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Управление жизненным циклом приложения"""
    global anpr_model

    # Startup: загрузка модели
    logger.info("Loading ANPR model...")
    start_time = time.time()
    anpr_model = ANPRModel(
        languages=['en', 'ru'],
        gpu=settings.USE_GPU
    )
    load_time = time.time() - start_time
    logger.info(f"ANPR model loaded successfully in {load_time:.2f} seconds")

    yield

    # Shutdown
    logger.info("Shutting down ML service...")
    anpr_model = None


# Создание FastAPI приложения
app = FastAPI(
    title="Gate ML Service",
    description="License Plate Recognition Service",
    version="1.0.0",
    lifespan=lifespan
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # В продакшене указать конкретные origins
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Подключение роутеров
app.include_router(recognition.router, prefix="/api/v1", tags=["recognition"])


@app.get("/", tags=["health"])
async def root():
    """Health check endpoint"""
    return {
        "service": "gate-ml",
        "status": "running",
        "version": "1.0.0"
    }


@app.get("/health", tags=["health"])
async def health():
    """Detailed health check"""
    return {
        "status": "healthy",
        "model_loaded": anpr_model is not None,
        "gpu_available": settings.USE_GPU
    }


def get_anpr_model() -> ANPRModel:
    """Dependency для получения модели"""
    if anpr_model is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="ANPR model not loaded"
        )
    return anpr_model
