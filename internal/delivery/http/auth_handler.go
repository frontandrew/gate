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

// RefreshToken обновляет access token используя refresh token
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req auth.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	response, err := h.authService.RefreshToken(r.Context(), &req)
	if err != nil {
		if err == domain.ErrInvalidToken {
			respondError(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}
		if err == domain.ErrUserNotFound {
			respondError(w, http.StatusUnauthorized, "User not found")
			return
		}
		if err == domain.ErrUserInactive {
			respondError(w, http.StatusForbidden, "User account is inactive")
			return
		}
		h.logger.Error("Failed to refresh token", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to refresh token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// Logout завершает сессию пользователя
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req auth.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.authService.Logout(r.Context(), &req)
	if err != nil {
		if err == domain.ErrInvalidToken {
			respondError(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}
		h.logger.Error("Failed to logout", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, http.StatusInternalServerError, "Failed to logout")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}
