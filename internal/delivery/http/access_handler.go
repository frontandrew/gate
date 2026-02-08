package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/frontandrew/gate/internal/delivery/http/middleware"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/usecase/access"
	"github.com/google/uuid"
)

// AccessHandler обрабатывает запросы связанные с проверкой доступа
type AccessHandler struct {
	accessService *access.Service
	logger        logger.Logger
}

// NewAccessHandler создает новый handler
func NewAccessHandler(accessService *access.Service, logger logger.Logger) *AccessHandler {
	return &AccessHandler{
		accessService: accessService,
		logger:        logger,
	}
}

// CheckAccess обрабатывает запрос на проверку доступа и распознавание номера
// POST /api/v1/access/check
func (h *AccessHandler) CheckAccess(w http.ResponseWriter, r *http.Request) {
	var req access.CheckAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Проверяем доступ
	response, err := h.accessService.CheckAccess(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to check access", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to check access")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// GetAccessLogs возвращает историю проездов
// GET /api/v1/access/logs
func (h *AccessHandler) GetAccessLogs(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры пагинации
	limit, offset := getPaginationParams(r)

	// Получаем user_id из query params (опционально)
	var userID *uuid.UUID
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid user_id")
			return
		}
		userID = &parsedID
	}

	// Получаем логи
	logs, err := h.accessService.GetAccessLogs(r.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get access logs", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to get access logs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    logs,
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetVehicleAccessLogs возвращает историю проездов автомобиля
// GET /api/v1/access/logs/vehicle/:id
func (h *AccessHandler) GetVehicleAccessLogs(w http.ResponseWriter, r *http.Request) {
	// Извлекаем vehicle_id из URL
	vehicleIDStr := getPathParam(r, "id")
	vehicleID, err := uuid.Parse(vehicleIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid vehicle ID")
		return
	}

	limit, offset := getPaginationParams(r)

	logs, err := h.accessService.GetAccessLogsByVehicle(r.Context(), vehicleID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get vehicle access logs", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to get vehicle access logs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    logs,
	})
}

// GetMyAccessLogs возвращает историю проездов текущего пользователя
// GET /api/v1/access/me/logs
func (h *AccessHandler) GetMyAccessLogs(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit, offset := getPaginationParams(r)

	logs, err := h.accessService.GetAccessLogs(r.Context(), &claims.UserID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user access logs", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to get access logs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    logs,
	})
}

// getPaginationParams извлекает параметры пагинации из query string
func getPaginationParams(r *http.Request) (limit, offset int) {
	limit = 50 // по умолчанию
	offset = 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 100 {
				limit = 100 // максимум 100
			}
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	return limit, offset
}
