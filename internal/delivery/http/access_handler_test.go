package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frontandrew/gate/internal/delivery/http/middleware"
	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/usecase/access"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAccessHandler_CheckAccess(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAccessService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешная проверка доступа - разрешен",
			requestBody: access.CheckAccessRequest{
				ImageBase64: "base64encodedimage",
				GateID:      "gate_001",
				Direction:   "IN",
			},
			mockSetup: func(m *MockAccessService) {
				response := &access.CheckAccessResponse{
					AccessGranted:    true,
					LicensePlate:     "А123ВС777",
					Confidence:       95.5,
					Reason:           "Valid pass found",
					RecognitionTime:  150,
					ValidationTime:   50,
				}
				m.On("CheckAccess", mock.Anything, mock.AnythingOfType("*access.CheckAccessRequest")).
					Return(response, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].(map[string]interface{})
				assert.True(t, data["access_granted"].(bool))
				assert.Equal(t, "А123ВС777", data["license_plate"])
			},
		},
		{
			name: "успешная проверка доступа - запрещен",
			requestBody: access.CheckAccessRequest{
				ImageBase64: "base64encodedimage",
				GateID:      "gate_001",
				Direction:   "IN",
			},
			mockSetup: func(m *MockAccessService) {
				response := &access.CheckAccessResponse{
					AccessGranted:   false,
					LicensePlate:    "У777КР199",
					Confidence:      92.0,
					Reason:          "Vehicle in blacklist",
					RecognitionTime: 140,
					ValidationTime:  30,
				}
				m.On("CheckAccess", mock.Anything, mock.AnythingOfType("*access.CheckAccessRequest")).
					Return(response, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].(map[string]interface{})
				assert.False(t, data["access_granted"].(bool))
				assert.Equal(t, "Vehicle in blacklist", data["reason"])
			},
		},
		{
			name:        "невалидный JSON",
			requestBody: "invalid json",
			mockSetup: func(m *MockAccessService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessService)
			tt.mockSetup(mockService)

			log := logger.NewNoop()
			handler := NewAccessHandler(mockService, log)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/access/check", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.CheckAccess(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

func TestAccessHandler_GetAccessLogs(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*MockAccessService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:        "успешное получение логов без фильтра",
			queryParams: "",
			mockSetup: func(m *MockAccessService) {
				logs := []*domain.AccessLog{
					CreateTestAccessLog(uuid.New(), uuid.New(), "А123ВС777", true, "Valid pass"),
					CreateTestAccessLog(uuid.New(), uuid.New(), "В456ЕК777", true, "Valid pass"),
				}
				m.On("GetAccessLogs", mock.Anything, (*uuid.UUID)(nil), 50, 0).
					Return(logs, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].([]interface{})
				assert.Len(t, data, 2)
				pagination := resp["pagination"].(map[string]interface{})
				assert.Equal(t, float64(50), pagination["limit"])
			},
		},
		{
			name:        "получение логов с пагинацией",
			queryParams: "?limit=10&offset=20",
			mockSetup: func(m *MockAccessService) {
				logs := []*domain.AccessLog{
					CreateTestAccessLog(uuid.New(), uuid.New(), "А123ВС777", true, "Valid pass"),
				}
				m.On("GetAccessLogs", mock.Anything, (*uuid.UUID)(nil), 10, 20).
					Return(logs, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				pagination := resp["pagination"].(map[string]interface{})
				assert.Equal(t, float64(10), pagination["limit"])
				assert.Equal(t, float64(20), pagination["offset"])
			},
		},
		{
			name:        "получение логов с фильтром по user_id",
			queryParams: "?user_id=" + uuid.New().String(),
			mockSetup: func(m *MockAccessService) {
				logs := []*domain.AccessLog{}
				m.On("GetAccessLogs", mock.Anything, mock.AnythingOfType("*uuid.UUID"), 50, 0).
					Return(logs, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].([]interface{})
				assert.Empty(t, data)
			},
		},
		{
			name:        "невалидный user_id",
			queryParams: "?user_id=invalid-uuid",
			mockSetup: func(m *MockAccessService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessService)
			tt.mockSetup(mockService)

			log := logger.NewNoop()
			handler := NewAccessHandler(mockService, log)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/access/logs"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.GetAccessLogs(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

func TestAccessHandler_GetVehicleAccessLogs(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name           string
		vehicleID      string
		queryParams    string
		mockSetup      func(*MockAccessService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:        "успешное получение логов автомобиля",
			vehicleID:   validID.String(),
			queryParams: "",
			mockSetup: func(m *MockAccessService) {
				logs := []*domain.AccessLog{
					CreateTestAccessLog(uuid.New(), validID, "А123ВС777", true, "Valid pass"),
					CreateTestAccessLog(uuid.New(), validID, "А123ВС777", true, "Valid pass"),
				}
				m.On("GetAccessLogsByVehicle", mock.Anything, validID, 50, 0).
					Return(logs, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].([]interface{})
				assert.Len(t, data, 2)
			},
		},
		{
			name:        "невалидный vehicle ID",
			vehicleID:   "invalid-uuid",
			queryParams: "",
			mockSetup: func(m *MockAccessService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
		{
			name:        "пустая история проездов",
			vehicleID:   validID.String(),
			queryParams: "",
			mockSetup: func(m *MockAccessService) {
				m.On("GetAccessLogsByVehicle", mock.Anything, validID, 50, 0).
					Return([]*domain.AccessLog{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].([]interface{})
				assert.Empty(t, data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessService)
			tt.mockSetup(mockService)

			log := logger.NewNoop()
			handler := NewAccessHandler(mockService, log)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/access/logs/vehicle/"+tt.vehicleID+tt.queryParams, nil)

			// Настройка chi router context для path параметра
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.vehicleID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.GetVehicleAccessLogs(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

func TestAccessHandler_GetMyAccessLogs(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		setupContext   func() context.Context
		queryParams    string
		mockSetup      func(*MockAccessService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешное получение моих логов",
			setupContext: func() context.Context {
				return CreateAuthContext(t, userID, "user@test.com", domain.RoleUser)
			},
			queryParams: "",
			mockSetup: func(m *MockAccessService) {
				logs := []*domain.AccessLog{
					CreateTestAccessLog(uuid.New(), uuid.New(), "А123ВС777", true, "Valid pass"),
				}
				m.On("GetAccessLogs", mock.Anything, &userID, 50, 0).
					Return(logs, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].([]interface{})
				assert.Len(t, data, 1)
			},
		},
		{
			name: "отсутствие авторизации",
			setupContext: func() context.Context {
				return context.Background()
			},
			queryParams: "",
			mockSetup: func(m *MockAccessService) {
				// Mock не будет вызван
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
		{
			name: "пустая история проездов",
			setupContext: func() context.Context {
				return CreateAuthContext(t, userID, "user@test.com", domain.RoleUser)
			},
			queryParams: "",
			mockSetup: func(m *MockAccessService) {
				m.On("GetAccessLogs", mock.Anything, &userID, 50, 0).
					Return([]*domain.AccessLog{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].([]interface{})
				assert.Empty(t, data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessService)
			tt.mockSetup(mockService)

			log := logger.NewNoop()
			handler := NewAccessHandler(mockService, log)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/access/me/logs"+tt.queryParams, nil)
			req = req.WithContext(tt.setupContext())
			w := httptest.NewRecorder()

			handler.GetMyAccessLogs(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}
