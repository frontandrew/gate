package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/jwt"
)

// contextKey - тип для ключей контекста
type contextKey string

const (
	// UserClaimsKey - ключ для сохранения claims пользователя в контексте
	UserClaimsKey contextKey = "user_claims"
)

// AuthMiddleware проверяет наличие и валидность JWT токена
func AuthMiddleware(tokenService *jwt.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Извлекаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// Проверяем формат: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Валидируем токен
			claims, err := tokenService.ValidateToken(tokenString)
			if err != nil {
				if err == domain.ErrTokenExpired {
					respondError(w, http.StatusUnauthorized, "Token expired")
					return
				}
				respondError(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			// Добавляем claims в контекст
			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole проверяет, что пользователь имеет одну из указанных ролей
func RequireRole(roles ...domain.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем claims из контекста
			claims, ok := r.Context().Value(UserClaimsKey).(*jwt.Claims)
			if !ok {
				respondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Проверяем роль
			hasRole := false
			for _, role := range roles {
				if claims.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserClaims извлекает claims пользователя из контекста
func GetUserClaims(ctx context.Context) (*jwt.Claims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(*jwt.Claims)
	return claims, ok
}

// respondError отправляет JSON ответ с ошибкой
func respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
