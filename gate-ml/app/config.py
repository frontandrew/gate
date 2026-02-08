"""
Configuration for ML Service
"""
import os
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Настройки ML сервиса"""

    # Server settings
    HOST: str = "0.0.0.0"
    PORT: int = 8001
    WORKERS: int = 1

    # ML Model settings
    USE_GPU: bool = False
    MODEL_PATH: str = "/app/models"
    MIN_CONFIDENCE: float = 0.7

    # Image processing
    MAX_IMAGE_SIZE: int = 10 * 1024 * 1024  # 10 MB
    ALLOWED_EXTENSIONS: set = {".jpg", ".jpeg", ".png", ".bmp"}

    # Logging
    LOG_LEVEL: str = "INFO"

    class Config:
        env_file = ".env"
        case_sensitive = True


# Создаем экземпляр настроек
settings = Settings()
