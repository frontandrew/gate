package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/usecase/pass"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPassHandler_CreatePass(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupContext   func() context.Context
		mockSetup      func(*MockPassService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешное создание пропуска",
			requestBody: pass.CreatePassRequest{
				UserID:     uuid.New(),
				VehicleIDs: []uuid.UUID{uuid.New()},
				PassType:   domain.PassTypePermanent,
			},
			setupContext: func() context.Context {
				return CreateAuthContext(t, uuid.New(), "admin@test.com", domain.RoleAdmin)
			},
			mockSetup: func(m *MockPassService) {
				m.On("CreatePass", mock.Anything, mock.AnythingOfType("*pass.CreatePassRequest")).
					Return(CreateTestPass(uuid.New(), uuid.New(), uuid.New(), domain.PassTypePermanent), nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				assert.NotNil(t, resp["data"])
			},
		},
		{
			name: "отсутствие авторизации",
			requestBody: pass.CreatePassRequest{
				UserID:     uuid.New(),
				VehicleIDs: []uuid.UUID{uuid.New()},
				PassType:   domain.PassTypePermanent,
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			mockSetup: func(m *MockPassService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
		{
			name:        "невалидный JSON",
			requestBody: "invalid json",
			setupContext: func() context.Context {
				return CreateAuthContext(t, uuid.New(), "admin@test.com", domain.RoleAdmin)
			},
			mockSetup: func(m *MockPassService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPassService)
			tt.mockSetup(mockService)

			log := logger.NewNoop()
			handler := NewPassHandler(mockService, log)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/passes", bytes.NewReader(body))
			req = req.WithContext(tt.setupContext())
			w := httptest.NewRecorder()

			handler.CreatePass(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

func TestPassHandler_GetMyPasses(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func() context.Context
		mockSetup      func(*MockPassService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешное получение пропусков",
			setupContext: func() context.Context {
				return CreateAuthContext(t, uuid.New(), "user@test.com", domain.RoleUser)
			},
			mockSetup: func(m *MockPassService) {
				passes := []*domain.Pass{
					CreateTestPass(uuid.New(), uuid.New(), uuid.New(), domain.PassTypePermanent),
					CreateTestPass(uuid.New(), uuid.New(), uuid.New(), domain.PassTypeTemporary),
				}
				m.On("GetPassesByUser", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(passes, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				data := resp["data"].([]interface{})
				assert.Len(t, data, 2)
			},
		},
		{
			name: "пустой список пропусков",
			setupContext: func() context.Context {
				return CreateAuthContext(t, uuid.New(), "user@test.com", domain.RoleUser)
			},
			mockSetup: func(m *MockPassService) {
				m.On("GetPassesByUser", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return([]*domain.Pass{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				data := resp["data"].([]interface{})
				assert.Empty(t, data)
			},
		},
		{
			name: "отсутствие авторизации",
			setupContext: func() context.Context {
				return context.Background()
			},
			mockSetup: func(m *MockPassService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPassService)
			tt.mockSetup(mockService)

			log := logger.NewNoop()
			handler := NewPassHandler(mockService, log)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/passes/me", nil)
			req = req.WithContext(tt.setupContext())
			w := httptest.NewRecorder()

			handler.GetMyPasses(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

func TestPassHandler_GetPassByID(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name           string
		passID         string
		mockSetup      func(*MockPassService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "успешное получение пропуска",
			passID: validID.String(),
			mockSetup: func(m *MockPassService) {
				p := CreateTestPass(validID, uuid.New(), uuid.New(), domain.PassTypePermanent)
				m.On("GetPassByID", mock.Anything, validID).Return(p, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				assert.NotNil(t, resp["data"])
			},
		},
		{
			name:   "пропуск не найден",
			passID: validID.String(),
			mockSetup: func(m *MockPassService) {
				m.On("GetPassByID", mock.Anything, validID).Return(nil, domain.ErrPassNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
		{
			name:   "невалидный UUID",
			passID: "invalid-uuid",
			mockSetup: func(m *MockPassService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPassService)
			tt.mockSetup(mockService)

			log := logger.NewNoop()
			handler := NewPassHandler(mockService, log)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/passes/"+tt.passID, nil)

			// Настройка chi router context для path параметра
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.passID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.GetPassByID(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

func TestPassHandler_RevokePass(t *testing.T) {
	validID := uuid.New()
	adminID := uuid.New()

	tests := []struct {
		name           string
		passID         string
		requestBody    interface{}
		setupContext   func() context.Context
		mockSetup      func(*MockPassService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "успешный отзыв пропуска",
			passID: validID.String(),
			requestBody: map[string]string{
				"reason": "Нарушение правил",
			},
			setupContext: func() context.Context {
				return CreateAuthContext(t, adminID, "admin@test.com", domain.RoleAdmin)
			},
			mockSetup: func(m *MockPassService) {
				m.On("RevokePass", mock.Anything, validID, adminID, "Нарушение правил").Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				assert.Equal(t, "Pass revoked successfully", resp["message"])
			},
		},
		{
			name:   "пропуск не найден",
			passID: validID.String(),
			requestBody: map[string]string{
				"reason": "Test",
			},
			setupContext: func() context.Context {
				return CreateAuthContext(t, adminID, "admin@test.com", domain.RoleAdmin)
			},
			mockSetup: func(m *MockPassService) {
				m.On("RevokePass", mock.Anything, validID, adminID, "Test").
					Return(domain.ErrPassNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
		{
			name:   "невалидный UUID",
			passID: "invalid-uuid",
			requestBody: map[string]string{
				"reason": "Test",
			},
			setupContext: func() context.Context {
				return CreateAuthContext(t, adminID, "admin@test.com", domain.RoleAdmin)
			},
			mockSetup: func(m *MockPassService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
		{
			name:   "отсутствие авторизации",
			passID: validID.String(),
			requestBody: map[string]string{
				"reason": "Test",
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			mockSetup: func(m *MockPassService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPassService)
			tt.mockSetup(mockService)

			log := logger.NewNoop()
			handler := NewPassHandler(mockService, log)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/passes/"+tt.passID+"/revoke", bytes.NewReader(body))

			// Настройка chi router context для path параметра
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.passID)
			ctx := context.WithValue(tt.setupContext(), chi.RouteCtxKey, rctx)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.RevokePass(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}
