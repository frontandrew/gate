package pass

import (
	"context"
	"fmt"
	"time"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/repository"
	"github.com/google/uuid"
)

// CreatePassRequest - запрос на создание пропуска
type CreatePassRequest struct {
	UserID     uuid.UUID       `json:"user_id" validate:"required"`
	PassType   domain.PassType `json:"pass_type" validate:"required"`
	ValidFrom  time.Time       `json:"valid_from" validate:"required"`
	ValidUntil *time.Time      `json:"valid_until,omitempty"`
	VehicleIDs []uuid.UUID     `json:"vehicle_ids" validate:"required,min=1"`
	CreatedBy  uuid.UUID       `json:"created_by" validate:"required"`
}

// Service содержит бизнес-логику работы с пропусками
type Service struct {
	passRepo        repository.PassRepository
	passVehicleRepo repository.PassVehicleRepository
	userRepo        repository.UserRepository
	vehicleRepo     repository.VehicleRepository
	logger          logger.Logger
}

// NewService создает новый экземпляр PassService
func NewService(
	passRepo repository.PassRepository,
	passVehicleRepo repository.PassVehicleRepository,
	userRepo repository.UserRepository,
	vehicleRepo repository.VehicleRepository,
	logger logger.Logger,
) *Service {
	return &Service{
		passRepo:        passRepo,
		passVehicleRepo: passVehicleRepo,
		userRepo:        userRepo,
		vehicleRepo:     vehicleRepo,
		logger:          logger,
	}
}

// CreatePass создает новый пропуск
func (s *Service) CreatePass(ctx context.Context, req *CreatePassRequest) (*domain.Pass, error) {
	s.logger.Info("Creating new pass", map[string]interface{}{
		"user_id":   req.UserID,
		"pass_type": req.PassType,
	})

	// Проверяем, что пользователь существует
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	// Проверяем, что все указанные автомобили существуют и принадлежат пользователю
	for _, vehicleID := range req.VehicleIDs {
		vehicle, err := s.vehicleRepo.GetByID(ctx, vehicleID)
		if err != nil {
			if err == domain.ErrVehicleNotFound {
				return nil, fmt.Errorf("vehicle %s not found", vehicleID)
			}
			return nil, fmt.Errorf("failed to get vehicle: %w", err)
		}

		if vehicle.OwnerID != req.UserID {
			return nil, fmt.Errorf("vehicle %s does not belong to user %s", vehicleID, req.UserID)
		}

		if !vehicle.IsActive {
			return nil, fmt.Errorf("vehicle %s is inactive", vehicleID)
		}
	}

	// Создаем пропуск
	pass := &domain.Pass{
		UserID:     req.UserID,
		PassType:   req.PassType,
		ValidFrom:  req.ValidFrom,
		ValidUntil: req.ValidUntil,
		IsActive:   true,
		CreatedBy:  &req.CreatedBy,
	}

	// Валидируем данные
	if err := pass.Validate(); err != nil {
		return nil, err
	}

	// Сохраняем пропуск в БД
	if err := s.passRepo.Create(ctx, pass); err != nil {
		s.logger.Error("Failed to create pass", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to create pass: %w", err)
	}

	// Привязываем автомобили к пропуску
	for _, vehicleID := range req.VehicleIDs {
		passVehicle := &domain.PassVehicle{
			PassID:    pass.ID,
			VehicleID: vehicleID,
			AddedBy:   &req.CreatedBy,
		}

		if err := s.passVehicleRepo.Create(ctx, passVehicle); err != nil {
			s.logger.Error("Failed to add vehicle to pass", map[string]interface{}{
				"pass_id":    pass.ID,
				"vehicle_id": vehicleID,
				"error":      err.Error(),
			})
			// Продолжаем, даже если не удалось добавить один автомобиль
		}
	}

	s.logger.Info("Pass created successfully", map[string]interface{}{
		"pass_id":        pass.ID,
		"vehicles_count": len(req.VehicleIDs),
	})

	return pass, nil
}

// GetPassByID возвращает пропуск по ID
func (s *Service) GetPassByID(ctx context.Context, id uuid.UUID) (*domain.Pass, error) {
	return s.passRepo.GetByID(ctx, id)
}

// GetPassesByUser возвращает все пропуска пользователя
func (s *Service) GetPassesByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Pass, error) {
	return s.passRepo.GetByUserID(ctx, userID)
}

// GetActivePassesByUser возвращает активные пропуска пользователя
func (s *Service) GetActivePassesByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Pass, error) {
	return s.passRepo.GetActivePassesByUser(ctx, userID)
}

// RevokePass отзывает пропуск
func (s *Service) RevokePass(ctx context.Context, passID, revokedBy uuid.UUID, reason string) error {
	s.logger.Info("Revoking pass", map[string]interface{}{
		"pass_id":    passID,
		"revoked_by": revokedBy,
		"reason":     reason,
	})

	// Проверяем, что пропуск существует
	pass, err := s.passRepo.GetByID(ctx, passID)
	if err != nil {
		return err
	}

	// Проверяем, что пропуск еще не отозван
	if !pass.IsActive {
		return domain.ErrPassAlreadyRevoked
	}

	// Отзываем пропуск
	if err := s.passRepo.Revoke(ctx, passID, revokedBy, reason); err != nil {
		s.logger.Error("Failed to revoke pass", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to revoke pass: %w", err)
	}

	s.logger.Info("Pass revoked successfully", map[string]interface{}{
		"pass_id": passID,
	})

	return nil
}

// AddVehicleToPass добавляет автомобиль к пропуску
func (s *Service) AddVehicleToPass(ctx context.Context, passID, vehicleID, addedBy uuid.UUID) error {
	// Проверяем, что пропуск существует
	pass, err := s.passRepo.GetByID(ctx, passID)
	if err != nil {
		return err
	}

	// Проверяем, что автомобиль существует и принадлежит владельцу пропуска
	vehicle, err := s.vehicleRepo.GetByID(ctx, vehicleID)
	if err != nil {
		return err
	}

	if vehicle.OwnerID != pass.UserID {
		return fmt.Errorf("vehicle does not belong to pass owner")
	}

	// Создаем связь
	passVehicle := &domain.PassVehicle{
		PassID:    passID,
		VehicleID: vehicleID,
		AddedBy:   &addedBy,
	}

	return s.passVehicleRepo.Create(ctx, passVehicle)
}

// RemoveVehicleFromPass удаляет автомобиль из пропуска
func (s *Service) RemoveVehicleFromPass(ctx context.Context, passID, vehicleID uuid.UUID) error {
	return s.passVehicleRepo.DeleteByPassAndVehicle(ctx, passID, vehicleID)
}
