package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RecognitionResult содержит результат распознавания номера
type RecognitionResult struct {
	Success         bool         `json:"success"`
	LicensePlate    string       `json:"license_plate"`
	Confidence      float64      `json:"confidence"`
	BoundingBox     *BoundingBox `json:"bounding_box,omitempty"`
	ProcessingTime  float64      `json:"processing_time_ms"`
	Error           string       `json:"error,omitempty"`
}

// BoundingBox содержит координаты распознанного номера на изображении
type BoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RecognitionRequest содержит запрос на распознавание
type recognitionRequest struct {
	ImageBase64   string  `json:"image_base64"`
	MinConfidence float64 `json:"min_confidence"`
}

// Client - интерфейс для работы с ML сервисом
type Client interface {
	// RecognizePlate распознает номер автомобиля на изображении
	RecognizePlate(ctx context.Context, imageBase64 string, minConfidence float64) (*RecognitionResult, error)

	// Health проверяет доступность ML сервиса
	Health(ctx context.Context) error
}

// httpClient - HTTP реализация ML клиента
type httpClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewHTTPClient создает новый HTTP клиент для ML сервиса
func NewHTTPClient(baseURL string, timeout time.Duration) Client {
	return &httpClient{
		baseURL: baseURL,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// RecognizePlate отправляет запрос на распознавание номера
func (c *httpClient) RecognizePlate(ctx context.Context, imageBase64 string, minConfidence float64) (*RecognitionResult, error) {
	// Формируем запрос
	reqBody := recognitionRequest{
		ImageBase64:   imageBase64,
		MinConfidence: minConfidence,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Создаем HTTP запрос
	url := fmt.Sprintf("%s/api/v1/recognize", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос с retry логикой
	var result *RecognitionResult
	var lastErr error

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Экспоненциальная задержка между попытками
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		result, lastErr = c.doRequest(req)
		if lastErr == nil {
			return result, nil
		}

		// Если это не временная ошибка, не повторяем
		if !isRetryable(lastErr) {
			break
		}
	}

	return nil, fmt.Errorf("recognition failed after %d attempts: %w", maxRetries, lastErr)
}

// doRequest выполняет HTTP запрос и обрабатывает ответ
func (c *httpClient) doRequest(req *http.Request) (*RecognitionResult, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Парсим ответ
	var result RecognitionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// Health проверяет доступность ML сервиса
func (c *httpClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// isRetryable определяет, можно ли повторить запрос при данной ошибке
func isRetryable(err error) bool {
	// Можно добавить более сложную логику определения
	// временных ошибок (network timeout, connection refused и т.д.)
	return true
}
