package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/jwt"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/usecase/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService - мок для auth service
type MockAuthService struct {
	mock.Mock
}

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

func (m *MockAuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockAuthService) ValidateToken(tokenString string) (*jwt.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.Claims), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest) (*auth.LoginResponse, error) {
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

// TestAuthHandler_Register тестирует регистрацию пользователя
func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешная регистрация",
			requestBody: auth.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				FullName: "Test User",
				Phone:    "+7 999 999 99 99",
				Role:     domain.RoleUser,
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, mock.AnythingOfType("*auth.RegisterRequest")).
					Return(&domain.User{
						ID:       uuid.New(),
						Email:    "test@example.com",
						FullName: "Test User",
						Role:     domain.RoleUser,
						IsActive: true,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "test@example.com", data["email"])
				assert.Equal(t, "Test User", data["full_name"])
			},
		},
		{
			name: "пользователь уже существует",
			requestBody: auth.RegisterRequest{
				Email:    "existing@example.com",
				Password: "password123",
				FullName: "Existing User",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, mock.AnythingOfType("*auth.RegisterRequest")).
					Return(nil, domain.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
				assert.Contains(t, resp["error"].(string), "already exists")
			},
		},
		{
			name:           "невалидный JSON",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем мок сервиса
			mockService := new(MockAuthService)
			tt.mockSetup(mockService)

			// Создаем handler
			log := logger.NewDevelopment()
			handler := NewAuthHandler(mockService, log)

			// Создаем запрос
			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Выполняем запрос
			handler.Register(w, req)

			// Проверяем результат
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestAuthHandler_Login тестирует вход пользователя
func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешный вход",
			requestBody: auth.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, mock.AnythingOfType("*auth.LoginRequest")).
					Return(&auth.LoginResponse{
						User: &domain.User{
							ID:       uuid.New(),
							Email:    "test@example.com",
							FullName: "Test User",
							Role:     domain.RoleUser,
						},
						AccessToken:  "access_token_here",
						RefreshToken: "refresh_token_here",
						ExpiresAt:    "2026-02-10T00:00:00Z",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].(map[string]interface{})
				assert.NotEmpty(t, data["access_token"])
				assert.NotEmpty(t, data["refresh_token"])
			},
		},
		{
			name: "неверные учетные данные",
			requestBody: auth.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, mock.AnythingOfType("*auth.LoginRequest")).
					Return(nil, domain.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
		{
			name: "неактивный пользователь",
			requestBody: auth.LoginRequest{
				Email:    "inactive@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, mock.AnythingOfType("*auth.LoginRequest")).
					Return(nil, domain.ErrUserInactive)
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			tt.mockSetup(mockService)

			log := logger.NewDevelopment()
			handler := NewAuthHandler(mockService, log)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Login(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestAuthHandler_Logout тестирует выход пользователя
func TestAuthHandler_Logout(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешный выход",
			requestBody: auth.LogoutRequest{
				RefreshToken: "valid_refresh_token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Logout", mock.Anything, mock.AnythingOfType("*auth.LogoutRequest")).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				assert.Equal(t, "Logged out successfully", resp["message"])
			},
		},
		{
			name: "невалидный refresh token",
			requestBody: auth.LogoutRequest{
				RefreshToken: "invalid_token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Logout", mock.Anything, mock.AnythingOfType("*auth.LogoutRequest")).
					Return(domain.ErrInvalidToken)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			tt.mockSetup(mockService)

			log := logger.NewDevelopment()
			handler := NewAuthHandler(mockService, log)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Logout(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestAuthHandler_RefreshToken тестирует обновление токена
func TestAuthHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешное обновление токена",
			requestBody: auth.RefreshTokenRequest{
				RefreshToken: "valid_refresh_token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("RefreshToken", mock.Anything, mock.AnythingOfType("*auth.RefreshTokenRequest")).
					Return(&auth.LoginResponse{
						User: &domain.User{
							ID:    uuid.New(),
							Email: "test@example.com",
							Role:  domain.RoleUser,
						},
						AccessToken:  "new_access_token",
						RefreshToken: "new_refresh_token",
						ExpiresAt:    "2026-02-10T00:00:00Z",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].(map[string]interface{})
				assert.NotEmpty(t, data["access_token"])
				assert.NotEmpty(t, data["refresh_token"])
			},
		},
		{
			name: "невалидный refresh token",
			requestBody: auth.RefreshTokenRequest{
				RefreshToken: "invalid_token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("RefreshToken", mock.Anything, mock.AnythingOfType("*auth.RefreshTokenRequest")).
					Return(nil, domain.ErrInvalidToken)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
		{
			name: "пользователь не найден",
			requestBody: auth.RefreshTokenRequest{
				RefreshToken: "orphaned_token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("RefreshToken", mock.Anything, mock.AnythingOfType("*auth.RefreshTokenRequest")).
					Return(nil, domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			tt.mockSetup(mockService)

			log := logger.NewDevelopment()
			handler := NewAuthHandler(mockService, log)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.RefreshToken(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}
