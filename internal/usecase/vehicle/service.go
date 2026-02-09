package vehicle

import (
	"context"
	"fmt"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/repository"
	"github.com/google/uuid"
)

// CreateVehicleRequest - запрос на создание автомобиля
type CreateVehicleRequest struct {
	OwnerID      uuid.UUID          `json:"owner_id" validate:"required"`
	LicensePlate string             `json:"license_plate" validate:"required"`
	VehicleType  domain.VehicleType `json:"vehicle_type" validate:"required"`
	Model        string             `json:"model,omitempty"`
	Color        string             `json:"color,omitempty"`
}

// Service содержит бизнес-логику работы с автомобилями
type Service struct {
	vehicleRepo repository.VehicleRepository
	userRepo    repository.UserRepository
	logger      logger.Logger
}

// NewService создает новый экземпляр VehicleService
func NewService(
	vehicleRepo repository.VehicleRepository,
	userRepo repository.UserRepository,
	logger logger.Logger,
) *Service {
	return &Service{
		vehicleRepo: vehicleRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// CreateVehicle создает новый автомобиль
func (s *Service) CreateVehicle(ctx context.Context, req *CreateVehicleRequest) (*domain.Vehicle, error) {
	s.logger.Info("Creating new vehicle", map[string]interface{}{
		"owner_id":      req.OwnerID,
		"license_plate": req.LicensePlate,
	})

	// Проверяем, что владелец существует
	owner, err := s.userRepo.GetByID(ctx, req.OwnerID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get owner: %w", err)
	}

	if !owner.IsActive {
		return nil, domain.ErrUserInactive
	}

	// Проверяем, что автомобиль с таким номером еще не зарегистрирован
	existingVehicle, err := s.vehicleRepo.GetByLicensePlate(ctx, req.LicensePlate)
	if err != nil && err != domain.ErrVehicleNotFound {
		return nil, fmt.Errorf("failed to check existing vehicle: %w", err)
	}

	if existingVehicle != nil {
		s.logger.Warn("Vehicle already exists", map[string]interface{}{
			"license_plate": req.LicensePlate,
		})
		return nil, domain.ErrVehicleAlreadyExists
	}

	// Создаем автомобиль
	vehicle := &domain.Vehicle{
		OwnerID:      req.OwnerID,
		LicensePlate: req.LicensePlate,
		VehicleType:  req.VehicleType,
		Model:        req.Model,
		Color:        req.Color,
		IsActive:     true,
	}

	// Валидируем данные
	if err := vehicle.Validate(); err != nil {
		return nil, err
	}

	// Сохраняем в БД
	if err := s.vehicleRepo.Create(ctx, vehicle); err != nil {
		s.logger.Error("Failed to create vehicle", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to create vehicle: %w", err)
	}

	s.logger.Info("Vehicle created successfully", map[string]interface{}{
		"vehicle_id": vehicle.ID,
	})

	return vehicle, nil
}

// GetVehicleByID возвращает автомобиль по ID
func (s *Service) GetVehicleByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error) {
	return s.vehicleRepo.GetByID(ctx, id)
}

// GetVehiclesByOwner возвращает все автомобили пользователя
func (s *Service) GetVehiclesByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Vehicle, error) {
	return s.vehicleRepo.GetByOwnerID(ctx, ownerID)
}

// GetVehicleByLicensePlate возвращает автомобиль по номеру
func (s *Service) GetVehicleByLicensePlate(ctx context.Context, licensePlate string) (*domain.Vehicle, error) {
	return s.vehicleRepo.GetByLicensePlate(ctx, licensePlate)
}

// UpdateVehicle обновляет данные автомобиля
func (s *Service) UpdateVehicle(ctx context.Context, vehicle *domain.Vehicle) error {
	// Валидируем данные
	if err := vehicle.Validate(); err != nil {
		return err
	}

	return s.vehicleRepo.Update(ctx, vehicle)
}

// DeleteVehicle удаляет автомобиль (мягкое удаление)
func (s *Service) DeleteVehicle(ctx context.Context, id uuid.UUID) error {
	return s.vehicleRepo.Delete(ctx, id)
}
