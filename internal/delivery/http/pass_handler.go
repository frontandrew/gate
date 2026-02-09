package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/frontandrew/gate/internal/delivery/http/middleware"
	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/usecase/pass"
	"github.com/google/uuid"
)

// PassService определяет интерфейс для сервиса пропусков
type PassService interface {
	CreatePass(ctx context.Context, req *pass.CreatePassRequest) (*domain.Pass, error)
	GetPassesByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Pass, error)
	GetPassByID(ctx context.Context, passID uuid.UUID) (*domain.Pass, error)
	RevokePass(ctx context.Context, passID, revokedBy uuid.UUID, reason string) error
}

// PassHandler обрабатывает запросы связанные с пропусками
type PassHandler struct {
	passService PassService
	logger      logger.Logger
}

// NewPassHandler создает новый handler
func NewPassHandler(passService PassService, logger logger.Logger) *PassHandler {
	return &PassHandler{
		passService: passService,
		logger:      logger,
	}
}

// CreatePass создает новый пропуск (только для админов и охранников)
// POST /api/v1/passes
func (h *PassHandler) CreatePass(w http.ResponseWriter, r *http.Request) {
	var req pass.CreatePassRequest
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

	// Устанавливаем created_by
	req.CreatedBy = claims.UserID

	p, err := h.passService.CreatePass(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create pass", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to create pass")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    p,
	})
}

// GetMyPasses возвращает все пропуска текущего пользователя
// GET /api/v1/passes/me
func (h *PassHandler) GetMyPasses(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	passes, err := h.passService.GetPassesByUser(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("Failed to get user passes", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to get passes")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    passes,
	})
}

// GetPassByID возвращает пропуск по ID
// GET /api/v1/passes/:id
func (h *PassHandler) GetPassByID(w http.ResponseWriter, r *http.Request) {
	passIDStr := getPathParam(r, "id")
	passID, err := uuid.Parse(passIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid pass ID")
		return
	}

	p, err := h.passService.GetPassByID(r.Context(), passID)
	if err != nil {
		if err == domain.ErrPassNotFound {
			respondError(w, http.StatusNotFound, "Pass not found")
			return
		}
		h.logger.Error("Failed to get pass", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to get pass")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    p,
	})
}

// RevokePass отзывает пропуск (только для админов и охранников)
// DELETE /api/v1/passes/:id/revoke
func (h *PassHandler) RevokePass(w http.ResponseWriter, r *http.Request) {
	passIDStr := getPathParam(r, "id")
	passID, err := uuid.Parse(passIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid pass ID")
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.passService.RevokePass(r.Context(), passID, claims.UserID, body.Reason); err != nil {
		if err == domain.ErrPassNotFound {
			respondError(w, http.StatusNotFound, "Pass not found")
			return
		}
		h.logger.Error("Failed to revoke pass", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to revoke pass")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Pass revoked successfully",
	})
}
