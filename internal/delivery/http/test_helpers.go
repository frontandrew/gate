package http

import (
	"context"
	"testing"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/jwt"
	"github.com/google/uuid"
)

// Ключ для хранения user_id в контексте
type contextKey string

const userIDContextKey contextKey = "user_id"

// CreateTestUser создает тестового пользователя
func CreateTestUser(id uuid.UUID, email, role string) *domain.User {
	return &domain.User{
		ID:       id,
		Email:    email,
		FullName: "Test User",
		Phone:    "+7 999 999 99 99",
		Role:     domain.UserRole(role),
		IsActive: true,
	}
}

// CreateTestVehicle создает тестовый автомобиль
func CreateTestVehicle(id, ownerID uuid.UUID, licensePlate string) *domain.Vehicle {
	return &domain.Vehicle{
		ID:           id,
		OwnerID:      ownerID,
		LicensePlate: licensePlate,
		VehicleType:  "car",
		Model:        "Test Model",
		Color:        "Test Color",
		IsActive:     true,
	}
}

// CreateTestPass создает тестовый пропуск
func CreateTestPass(id, userID uuid.UUID, passType string) *domain.Pass{
	return &domain.Pass{
		ID:       id,
		UserID:   userID,
		PassType: domain.PassType(passType),
		IsActive: true,
	}
}

// CreateTestAccessLog создает тестовый access log
func CreateTestAccessLog(id, userID, vehicleID uuid.UUID) *domain.AccessLog {
	return &domain.AccessLog{
		ID:            id,
		UserID:        &userID,
		VehicleID:     &vehicleID,
		LicensePlate:  "А123ВС777",
		AccessGranted: true,
		AccessReason:  "Valid pass",
		GateID:        "gate_001",
		Direction:     "IN",
	}
}

// CreateAuthContext создает контекст с user_id для тестирования
func CreateAuthContext(t *testing.T, userID uuid.UUID) context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, userIDContextKey, userID)
}

// CreateTestJWTToken создает тестовый JWT токен
func CreateTestJWTToken(user *domain.User, secretKey string) (string, error) {
	tokenService := jwt.NewTokenService(secretKey, 15*60, 7*24*60*60) // 15 min, 7 days
	tokenPair, err := tokenService.GenerateTokenPair(user)
	if err != nil {
		return "", err
	}
	return tokenPair.AccessToken, nil
}

// AssertSuccess проверяет успешный ответ API
func AssertSuccess(t *testing.T, response map[string]interface{}) {
	t.Helper()
	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Errorf("Expected success=true, got %v", response)
	}
}

// AssertError проверяет ошибочный ответ API
func AssertError(t *testing.T, response map[string]interface{}) {
	t.Helper()
	success, ok := response["success"].(bool)
	if !ok || success {
		t.Errorf("Expected success=false, got %v", response)
	}
}
