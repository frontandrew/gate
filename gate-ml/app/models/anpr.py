"""
Automatic Number Plate Recognition (ANPR) Model
Using EasyOCR for license plate recognition
"""
import logging
import re
from typing import Optional, Tuple

import cv2
import easyocr
import numpy as np
from PIL import Image

logger = logging.getLogger(__name__)


class ANPRModel:
    """Модель для распознавания автомобильных номеров"""

    def __init__(self, languages: list = None, gpu: bool = False):
        """
        Инициализация модели EasyOCR

        Args:
            languages: Список языков для распознавания (по умолчанию: ['en', 'ru'])
            gpu: Использовать ли GPU для обработки
        """
        if languages is None:
            languages = ['en', 'ru']

        logger.info(f"Initializing EasyOCR with languages: {languages}, GPU: {gpu}")
        self.reader = easyocr.Reader(languages, gpu=gpu)
        self.min_confidence = 0.5

        # Паттерны для российских номеров
        # Формат: А123ВС777 (буква, 3 цифры, 2 буквы, 2-3 цифры)
        self.russian_pattern = re.compile(r'[АВЕКМНОРСТУХ]\d{3}[АВЕКМНОРСТУХ]{2}\d{2,3}')

        # Паттерны для международных номеров
        self.international_pattern = re.compile(r'[A-Z0-9]{4,10}')

    def preprocess_image(self, image: np.ndarray) -> np.ndarray:
        """
        Предобработка изображения для улучшения распознавания

        Args:
            image: Исходное изображение (numpy array)

        Returns:
            Обработанное изображение
        """
        # Конвертация в grayscale
        if len(image.shape) == 3:
            gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        else:
            gray = image

        # Увеличение контраста
        clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8, 8))
        enhanced = clahe.apply(gray)

        # Удаление шумов
        denoised = cv2.fastNlMeansDenoising(enhanced, None, 10, 7, 21)

        # Бинаризация (Otsu's thresholding)
        _, binary = cv2.threshold(denoised, 0, 255, cv2.THRESH_BINARY + cv2.THRESH_OTSU)

        return binary

    def normalize_plate(self, text: str) -> str:
        """
        Нормализация распознанного текста номера

        Args:
            text: Распознанный текст

        Returns:
            Нормализованный номер
        """
        # Убираем пробелы и приводим к верхнему регистру
        normalized = text.upper().replace(" ", "").replace("-", "")

        # Замена похожих символов (OCR может путать)
        replacements = {
            'O': '0',
            'I': '1',
            'Z': '2',
            'S': '5',
            'B': '8',
        }

        for old, new in replacements.items():
            # Заменяем только если это имеет смысл в контексте номера
            if old in normalized:
                # Применяем эвристику для определения, нужна ли замена
                pass

        return normalized

    def validate_plate(self, text: str) -> bool:
        """
        Проверка, является ли распознанный текст номером автомобиля

        Args:
            text: Распознанный текст

        Returns:
            True если текст похож на номер
        """
        normalized = self.normalize_plate(text)

        # Проверка по российскому паттерну
        if self.russian_pattern.match(normalized):
            return True

        # Проверка по международному паттерну
        if self.international_pattern.match(normalized):
            return True

        return False

    def recognize(self, image_data: bytes) -> Optional[Tuple[str, float, dict]]:
        """
        Распознавание номера автомобиля на изображении

        Args:
            image_data: Байты изображения

        Returns:
            Tuple[license_plate, confidence, bounding_box] или None если номер не найден
        """
        try:
            # Декодируем изображение
            nparr = np.frombuffer(image_data, np.uint8)
            image = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

            if image is None:
                logger.error("Failed to decode image")
                return None

            # Предобработка
            processed = self.preprocess_image(image)

            # Распознавание с помощью EasyOCR
            results = self.reader.readtext(processed)

            if not results:
                logger.info("No text detected in image")
                return None

            # Поиск наиболее вероятного номера
            best_plate = None
            best_confidence = 0
            best_bbox = None

            for (bbox, text, confidence) in results:
                # Нормализуем текст
                normalized_text = self.normalize_plate(text)

                # Проверяем, похож ли текст на номер автомобиля
                if self.validate_plate(normalized_text):
                    if confidence > best_confidence:
                        best_plate = normalized_text
                        best_confidence = confidence
                        # bbox: [[x1,y1], [x2,y2], [x3,y3], [x4,y4]]
                        best_bbox = {
                            "x": int(bbox[0][0]),
                            "y": int(bbox[0][1]),
                            "width": int(bbox[2][0] - bbox[0][0]),
                            "height": int(bbox[2][1] - bbox[0][1])
                        }

            if best_plate and best_confidence >= self.min_confidence:
                logger.info(f"Recognized plate: {best_plate} (confidence: {best_confidence:.2f})")
                return best_plate, best_confidence, best_bbox
            else:
                logger.info(f"No valid plate found (best confidence: {best_confidence:.2f})")
                return None

        except Exception as e:
            logger.error(f"Error during recognition: {str(e)}", exc_info=True)
            return None
