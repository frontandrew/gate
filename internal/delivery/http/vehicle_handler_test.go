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
	"github.com/frontandrew/gate/internal/usecase/vehicle"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestVehicleHandler_CreateVehicle тестирует создание автомобиля
func TestVehicleHandler_CreateVehicle(t *testing.T) {
	userID := uuid.New()
	vehicleID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockVehicleService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "успешное создание",
			requestBody: vehicle.CreateVehicleRequest{
				OwnerID:      userID,
				LicensePlate: "А123ВС777",
				VehicleType:  "car",
				Model:        "Toyota Camry",
				Color:        "Черный",
			},
			mockSetup: func(m *MockVehicleService) {
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.CreateVehicleRequest")).
					Return(&domain.Vehicle{
						ID:           vehicleID,
						OwnerID:      userID,
						LicensePlate: "А123ВС777",
						VehicleType:  "car",
						Model:        "Toyota Camry",
						Color:        "Черный",
						IsActive:     true,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				if data, ok := resp["data"].(map[string]interface{}); ok {
					assert.Equal(t, "А123ВС777", data["license_plate"])
				}
			},
		},
		{
			name: "дублирующийся номер",
			requestBody: vehicle.CreateVehicleRequest{
				OwnerID:      userID,
				LicensePlate: "А111АА111",
				VehicleType:  "car",
			},
			mockSetup: func(m *MockVehicleService) {
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.CreateVehicleRequest")).
					Return(nil, domain.ErrVehicleAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
		{
			name:           "невалидный JSON",
			requestBody:    "invalid",
			mockSetup:      func(m *MockVehicleService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockVehicleService)
			tt.mockSetup(mockService)

			log := logger.NewDevelopment()
			handler := NewVehicleHandler(mockService, log)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", bytes.NewReader(body))
			req = req.WithContext(CreateAuthContext(t, userID, "test@example.com", domain.RoleUser))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateVehicle(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestVehicleHandler_GetMyVehicles тестирует получение автомобилей пользователя
func TestVehicleHandler_GetMyVehicles(t *testing.T) {
	userID := uuid.New()
	vehicles := []*domain.Vehicle{
		{
			ID:           uuid.New(),
			OwnerID:      userID,
			LicensePlate: "А123ВС777",
			VehicleType:  "car",
			Model:        "Toyota Camry",
			IsActive:     true,
		},
		{
			ID:           uuid.New(),
			OwnerID:      userID,
			LicensePlate: "В456ДЕ777",
			VehicleType:  "car",
			Model:        "Honda Accord",
			IsActive:     true,
		},
	}

	tests := []struct {
		name           string
		userID         uuid.UUID
		mockSetup      func(*MockVehicleService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "успешное получение",
			userID: userID,
			mockSetup: func(m *MockVehicleService) {
				m.On("GetVehiclesByOwner", mock.Anything, userID).Return(vehicles, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				if data, ok := resp["data"].([]interface{}); ok {
					assert.Len(t, data, 2)
				}
			},
		},
		{
			name:   "нет автомобилей",
			userID: userID,
			mockSetup: func(m *MockVehicleService) {
				m.On("GetVehiclesByOwner", mock.Anything, userID).Return([]*domain.Vehicle{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				if data, ok := resp["data"].([]interface{}); ok {
					assert.Len(t, data, 0)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockVehicleService)
			tt.mockSetup(mockService)

			log := logger.NewDevelopment()
			handler := NewVehicleHandler(mockService, log)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/me", nil)
			req = req.WithContext(CreateAuthContext(t, tt.userID, "test@example.com", domain.RoleUser))

			w := httptest.NewRecorder()
			handler.GetMyVehicles(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestVehicleHandler_GetVehicleByID тестирует получение автомобиля по ID
func TestVehicleHandler_GetVehicleByID(t *testing.T) {
	vehicleID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		vehicleID      string
		mockSetup      func(*MockVehicleService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:      "успешное получение",
			vehicleID: vehicleID.String(),
			mockSetup: func(m *MockVehicleService) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).Return(&domain.Vehicle{
					ID:           vehicleID,
					OwnerID:      userID,
					LicensePlate: "А123ВС777",
					VehicleType:  "car",
					Model:        "Toyota Camry",
					IsActive:     true,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.True(t, success) }
				if data, ok := resp["data"].(map[string]interface{}); ok {
					assert.Equal(t, "А123ВС777", data["license_plate"])
				}
			},
		},
		{
			name:      "автомобиль не найден",
			vehicleID: vehicleID.String(),
			mockSetup: func(m *MockVehicleService) {
				m.On("GetVehicleByID", mock.Anything, vehicleID).
					Return(nil, domain.ErrVehicleNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
		{
			name:           "невалидный UUID",
			vehicleID:      "invalid-uuid",
			mockSetup:      func(m *MockVehicleService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); ok { assert.False(t, success) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockVehicleService)
			tt.mockSetup(mockService)

			log := logger.NewDevelopment()
			handler := NewVehicleHandler(mockService, log)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+tt.vehicleID, nil)

			// Настраиваем chi router для передачи параметра id
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.vehicleID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.GetVehicleByID(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}
