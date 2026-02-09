package http

import (
	"context"
	"testing"
	"time"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/jwt"
	"github.com/frontandrew/gate/internal/usecase/access"
	"github.com/frontandrew/gate/internal/usecase/auth"
	"github.com/frontandrew/gate/internal/usecase/pass"
	"github.com/frontandrew/gate/internal/usecase/vehicle"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// Ключ для хранения user_id в контексте
type contextKey string

const userIDContextKey contextKey = "user_id"

// ============================================================================
// Mock Services
// ============================================================================

// MockAuthService мок для auth.Service
type MockAuthService struct {
	mock.Mock
}

var _ interface{} = &MockAuthService{}

func (m *MockAuthService) Register(ctx context.Context, req *auth.RegisterRequest) (*domain.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.LoginResponse), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, req *auth.LogoutRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest) (*auth.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.LoginResponse), args.Error(1)
}

func (m *MockAuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// MockVehicleService мок для vehicle.Service
type MockVehicleService struct {
	mock.Mock
}

func (m *MockVehicleService) CreateVehicle(ctx context.Context, req *vehicle.CreateVehicleRequest) (*domain.Vehicle, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *MockVehicleService) GetVehiclesByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Vehicle, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Vehicle), args.Error(1)
}

func (m *MockVehicleService) GetVehicleByID(ctx context.Context, vehicleID uuid.UUID) (*domain.Vehicle, error) {
	args := m.Called(ctx, vehicleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

// MockPassService мок для pass.Service
type MockPassService struct {
	mock.Mock
}

func (m *MockPassService) CreatePass(ctx context.Context, req *pass.CreatePassRequest) (*domain.Pass, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pass), args.Error(1)
}

func (m *MockPassService) GetPassesByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Pass, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Pass), args.Error(1)
}

func (m *MockPassService) GetPassByID(ctx context.Context, passID uuid.UUID) (*domain.Pass, error) {
	args := m.Called(ctx, passID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pass), args.Error(1)
}

func (m *MockPassService) RevokePass(ctx context.Context, passID, revokedBy uuid.UUID, reason string) error {
	args := m.Called(ctx, passID, revokedBy, reason)
	return args.Error(0)
}

// MockAccessService мок для access.Service
type MockAccessService struct {
	mock.Mock
}

func (m *MockAccessService) CheckAccess(ctx context.Context, req *access.CheckAccessRequest) (*access.CheckAccessResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*access.CheckAccessResponse), args.Error(1)
}

func (m *MockAccessService) GetAccessLogs(ctx context.Context, userID *uuid.UUID, limit, offset int) ([]*domain.AccessLog, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AccessLog), args.Error(1)
}

func (m *MockAccessService) GetAccessLogsByVehicle(ctx context.Context, vehicleID uuid.UUID, limit, offset int) ([]*domain.AccessLog, error) {
	args := m.Called(ctx, vehicleID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AccessLog), args.Error(1)
}

// ============================================================================
// Test Data Factories
// ============================================================================

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
func CreateTestPass(id, userID, vehicleID uuid.UUID, passType domain.PassType) *domain.Pass {
	now := time.Now()
	return &domain.Pass{
		ID:        id,
		UserID:    userID,
		PassType:  passType,
		ValidFrom: now,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateTestAccessLog создает тестовый access log
func CreateTestAccessLog(id, vehicleID uuid.UUID, licensePlate string, accessGranted bool, reason string) *domain.AccessLog {
	userID := uuid.New()
	return &domain.AccessLog{
		ID:            id,
		UserID:        &userID,
		VehicleID:     &vehicleID,
		LicensePlate:  licensePlate,
		AccessGranted: accessGranted,
		AccessReason:  reason,
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
