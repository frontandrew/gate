package http

import (
	"encoding/json"
	"net/http"

	"github.com/frontandrew/gate/internal/delivery/http/middleware"
	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/usecase/auth"
)

// AuthHandler обрабатывает запросы аутентификации
type AuthHandler struct {
	authService *auth.Service
	logger      logger.Logger
}

// NewAuthHandler создает новый handler
func NewAuthHandler(authService *auth.Service, logger logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register обрабатывает регистрацию нового пользователя
// POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		if err == domain.ErrUserAlreadyExists {
			respondError(w, http.StatusConflict, "User already exists")
			return
		}
		h.logger.Error("Failed to register user", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to register user")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    user,
	})
}

// Login обрабатывает вход пользователя
// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	response, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		if err == domain.ErrInvalidCredentials {
			respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		if err == domain.ErrUserInactive {
			respondError(w, http.StatusForbidden, "User account is inactive")
			return
		}
		h.logger.Error("Failed to login user", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to login")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// GetMe возвращает информацию о текущем пользователе
// GET /api/v1/auth/me
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста (добавлен middleware)
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		h.logger.Error("Failed to get user", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    user,
	})
}
