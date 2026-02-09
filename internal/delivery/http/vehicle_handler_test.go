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

// MockVehicleService - мок для vehicle service
type MockVehicleService struct {
	mock.Mock
}

func (m *MockVehicleService) Create(ctx context.Context, req *vehicle.CreateVehicleRequest) (*domain.Vehicle, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *MockVehicleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *MockVehicleService) GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Vehicle, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Vehicle), args.Error(1)
}

func (m *MockVehicleService) Update(ctx context.Context, req *vehicle.UpdateVehicleRequest) (*domain.Vehicle, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *MockVehicleService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

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
				m.On("Create", mock.Anything, mock.AnythingOfType("*vehicle.CreateVehicleRequest")).
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
				assert.True(t, resp["success"].(bool))
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "А123ВС777", data["license_plate"])
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
				m.On("Create", mock.Anything, mock.AnythingOfType("*vehicle.CreateVehicleRequest")).
					Return(nil, domain.ErrVehicleAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
		{
			name:           "невалидный JSON",
			requestBody:    "invalid",
			mockSetup:      func(m *MockVehicleService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
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
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateVehicle(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
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
				m.On("GetByOwner", mock.Anything, userID).Return(vehicles, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].([]interface{})
				assert.Len(t, data, 2)
			},
		},
		{
			name:   "нет автомобилей",
			userID: userID,
			mockSetup: func(m *MockVehicleService) {
				m.On("GetByOwner", mock.Anything, userID).Return([]*domain.Vehicle{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				data := resp["data"].([]interface{})
				assert.Len(t, data, 0)
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
			// Добавляем user_id в контекст (как это делает AuthMiddleware)
			ctx := context.WithValue(req.Context(), userIDContextKey, tt.userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.GetMyVehicles(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
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
				m.On("GetByID", mock.Anything, vehicleID).Return(&domain.Vehicle{
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
				assert.True(t, resp["success"].(bool))
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "А123ВС777", data["license_plate"])
			},
		},
		{
			name:      "автомобиль не найден",
			vehicleID: vehicleID.String(),
			mockSetup: func(m *MockVehicleService) {
				m.On("GetByID", mock.Anything, vehicleID).
					Return(nil, domain.ErrVehicleNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
		{
			name:           "невалидный UUID",
			vehicleID:      "invalid-uuid",
			mockSetup:      func(m *MockVehicleService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
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
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}
