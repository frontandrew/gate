package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/frontandrew/gate/internal/delivery/http/middleware"
	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/usecase/vehicle"
	"github.com/google/uuid"
)

// VehicleService определяет интерфейс для сервиса автомобилей
type VehicleService interface {
	CreateVehicle(ctx context.Context, req *vehicle.CreateVehicleRequest) (*domain.Vehicle, error)
	GetVehiclesByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Vehicle, error)
	GetVehicleByID(ctx context.Context, vehicleID uuid.UUID) (*domain.Vehicle, error)
}

// VehicleHandler обрабатывает запросы связанные с автомобилями
type VehicleHandler struct {
	vehicleService VehicleService
	logger         logger.Logger
}

// NewVehicleHandler создает новый handler
func NewVehicleHandler(vehicleService VehicleService, logger logger.Logger) *VehicleHandler {
	return &VehicleHandler{
		vehicleService: vehicleService,
		logger:         logger,
	}
}

// CreateVehicle создает новый автомобиль
// POST /api/v1/vehicles
func (h *VehicleHandler) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	var req vehicle.CreateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Получаем текущего пользователя
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Пользователь может создавать автомобили только для себя (если не админ)
	if req.OwnerID != claims.UserID && claims.Role != domain.RoleAdmin {
		respondError(w, http.StatusForbidden, "Cannot create vehicle for another user")
		return
	}

	v, err := h.vehicleService.CreateVehicle(r.Context(), &req)
	if err != nil {
		if err == domain.ErrVehicleAlreadyExists {
			respondError(w, http.StatusConflict, "Vehicle already exists")
			return
		}
		h.logger.Error("Failed to create vehicle", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to create vehicle")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    v,
	})
}

// GetMyVehicles возвращает все автомобили текущего пользователя
// GET /api/v1/vehicles/me
func (h *VehicleHandler) GetMyVehicles(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	vehicles, err := h.vehicleService.GetVehiclesByOwner(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("Failed to get user vehicles", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to get vehicles")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    vehicles,
	})
}

// GetVehicleByID возвращает автомобиль по ID
// GET /api/v1/vehicles/:id
func (h *VehicleHandler) GetVehicleByID(w http.ResponseWriter, r *http.Request) {
	vehicleIDStr := getPathParam(r, "id")
	vehicleID, err := uuid.Parse(vehicleIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid vehicle ID")
		return
	}

	v, err := h.vehicleService.GetVehicleByID(r.Context(), vehicleID)
	if err != nil {
		if err == domain.ErrVehicleNotFound {
			respondError(w, http.StatusNotFound, "Vehicle not found")
			return
		}
		h.logger.Error("Failed to get vehicle", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to get vehicle")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    v,
	})
}
