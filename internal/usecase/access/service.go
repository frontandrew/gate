package access

import (
	"context"
	"fmt"
	"time"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/infrastructure/ml"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/repository"
	"github.com/google/uuid"
)

// CheckAccessRequest - запрос на проверку доступа
type CheckAccessRequest struct {
	ImageBase64 string `json:"image_base64" validate:"required"`
	GateID      string `json:"gate_id" validate:"required"`
	Direction   string `json:"direction" validate:"required,oneof=IN OUT"`
}

// CheckAccessResponse - ответ на проверку доступа
type CheckAccessResponse struct {
	AccessGranted bool            `json:"access_granted"`
	LicensePlate  string          `json:"license_plate"`
	Confidence    float64         `json:"confidence"`
	Vehicle       *domain.Vehicle `json:"vehicle,omitempty"`
	User          *domain.User    `json:"user,omitempty"`
	Pass          *domain.Pass    `json:"pass,omitempty"`
	Reason        string          `json:"reason"`
	Timestamp     time.Time       `json:"timestamp"`
}

// Service содержит бизнес-логику проверки доступа
type Service struct {
	vehicleRepo   repository.VehicleRepository
	userRepo      repository.UserRepository
	passRepo      repository.PassRepository
	accessLogRepo repository.AccessLogRepository
	whitelistRepo repository.WhitelistRepository // ПРИОРИТЕТ 1
	blacklistRepo repository.BlacklistRepository // ПРИОРИТЕТ 2
	mlClient      ml.Client
	logger        logger.Logger
	minConfidence float64
}

// NewService создает новый экземпляр AccessService
func NewService(
	vehicleRepo repository.VehicleRepository,
	userRepo repository.UserRepository,
	passRepo repository.PassRepository,
	accessLogRepo repository.AccessLogRepository,
	whitelistRepo repository.WhitelistRepository,
	blacklistRepo repository.BlacklistRepository,
	mlClient ml.Client,
	logger logger.Logger,
	minConfidence float64,
) *Service {
	return &Service{
		vehicleRepo:   vehicleRepo,
		userRepo:      userRepo,
		passRepo:      passRepo,
		accessLogRepo: accessLogRepo,
		whitelistRepo: whitelistRepo,
		blacklistRepo: blacklistRepo,
		mlClient:      mlClient,
		logger:        logger,
		minConfidence: minConfidence,
	}
}

// CheckAccess - КЛЮЧЕВОЙ МЕТОД системы
// Реализует user-centric логику проверки доступа с приоритетными списками:
// 1. Номер авто → [БЕЛЫЙ СПИСОК?] → РАЗРЕШИТЬ (безусловно, высший приоритет)
// 2. Номер авто → [ЧЕРНЫЙ СПИСОК?] → ОТКАЗАТЬ (безусловно)
// 3. Номер авто → Автомобиль → Владелец (Пользователь) → Активные пропуска → Решение о доступе
func (s *Service) CheckAccess(ctx context.Context, req *CheckAccessRequest) (*CheckAccessResponse, error) {
	s.logger.Info("Starting access check", map[string]interface{}{
		"gate_id":   req.GateID,
		"direction": req.Direction,
	})

	response := &CheckAccessResponse{
		Timestamp: time.Now(),
	}

	// ШАГ 1: Распознаем номер автомобиля через ML сервис
	recognitionResult, err := s.mlClient.RecognizePlate(ctx, req.ImageBase64, s.minConfidence)
	if err != nil {
		s.logger.Error("ML recognition failed", map[string]interface{}{
			"error": err.Error(),
		})
		response.AccessGranted = false
		response.Reason = "Recognition service unavailable"
		s.logAccess(ctx, response, req, nil, nil, nil)
		return response, nil
	}

	if !recognitionResult.Success {
		s.logger.Info("License plate not recognized", map[string]interface{}{
			"error": recognitionResult.Error,
		})
		response.AccessGranted = false
		response.Reason = fmt.Sprintf("License plate not recognized: %s", recognitionResult.Error)
		s.logAccess(ctx, response, req, nil, nil, nil)
		return response, nil
	}

	response.LicensePlate = recognitionResult.LicensePlate
	response.Confidence = recognitionResult.Confidence

	s.logger.Info("License plate recognized", map[string]interface{}{
		"plate":      recognitionResult.LicensePlate,
		"confidence": recognitionResult.Confidence,
	})

	// ШАГ 2 (ПРИОРИТЕТ 1): Проверяем БЕЛЫЙ СПИСОК
	// Если номер в белом списке - РАЗРЕШАЕМ доступ БЕЗ ДАЛЬНЕЙШИХ ПРОВЕРОК
	isWhitelisted, whitelistReason, err := s.whitelistRepo.IsWhitelisted(ctx, recognitionResult.LicensePlate)
	if err != nil {
		s.logger.Error("Failed to check whitelist", map[string]interface{}{
			"error": err.Error(),
		})
		// Продолжаем работу даже при ошибке whitelist (fail-open для критичных служб)
	}
	if isWhitelisted {
		s.logger.Info("License plate is whitelisted", map[string]interface{}{
			"plate":  recognitionResult.LicensePlate,
			"reason": whitelistReason,
		})
		response.AccessGranted = true
		response.Reason = fmt.Sprintf("Whitelisted: %s", whitelistReason)
		s.logAccess(ctx, response, req, nil, nil, nil)
		return response, nil
	}

	// ШАГ 3 (ПРИОРИТЕТ 2): Проверяем ЧЕРНЫЙ СПИСОК
	// Если номер в черном списке - ОТКАЗЫВАЕМ в доступе
	isBlacklisted, blacklistReason, err := s.blacklistRepo.IsBlacklisted(ctx, recognitionResult.LicensePlate)
	if err != nil {
		s.logger.Error("Failed to check blacklist", map[string]interface{}{
			"error": err.Error(),
		})
		// Продолжаем работу даже при ошибке blacklist
	}
	if isBlacklisted {
		s.logger.Info("License plate is blacklisted", map[string]interface{}{
			"plate":  recognitionResult.LicensePlate,
			"reason": blacklistReason,
		})
		response.AccessGranted = false
		response.Reason = fmt.Sprintf("Blacklisted: %s", blacklistReason)
		s.logAccess(ctx, response, req, nil, nil, nil)
		return response, nil
	}

	// ШАГ 4 (ПРИОРИТЕТ 3): Стандартная проверка через пропуски
	// Находим автомобиль в БД по номеру
	vehicle, err := s.vehicleRepo.GetByLicensePlate(ctx, recognitionResult.LicensePlate)
	if err != nil {
		if err == domain.ErrVehicleNotFound {
			s.logger.Info("Vehicle not found in database", map[string]interface{}{
				"plate": recognitionResult.LicensePlate,
			})
			response.AccessGranted = false
			response.Reason = "Vehicle not registered"
			s.logAccess(ctx, response, req, nil, nil, nil)
			return response, nil
		}
		s.logger.Error("Failed to get vehicle", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Проверяем, что автомобиль активен
	if !vehicle.IsActive {
		s.logger.Info("Vehicle is inactive", map[string]interface{}{
			"vehicle_id": vehicle.ID,
		})
		response.AccessGranted = false
		response.Reason = "Vehicle is inactive"
		s.logAccess(ctx, response, req, vehicle, nil, nil)
		return response, nil
	}

	response.Vehicle = vehicle

	// ШАГ 5: Получаем владельца автомобиля (ПОЛЬЗОВАТЕЛЬ - центральная сущность!)
	user, err := s.userRepo.GetByID(ctx, vehicle.OwnerID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			s.logger.Warn("Vehicle owner not found", map[string]interface{}{
				"vehicle_id": vehicle.ID,
				"owner_id":   vehicle.OwnerID,
			})
			response.AccessGranted = false
			response.Reason = "Vehicle owner not found"
			s.logAccess(ctx, response, req, vehicle, nil, nil)
			return response, nil
		}
		s.logger.Error("Failed to get user", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Проверяем, что пользователь активен
	if !user.IsActive {
		s.logger.Info("User is inactive", map[string]interface{}{
			"user_id": user.ID,
		})
		response.AccessGranted = false
		response.Reason = "User account is inactive"
		s.logAccess(ctx, response, req, vehicle, user, nil)
		return response, nil
	}

	response.User = user

	// ШАГ 6: Получаем ВСЕ активные пропуска пользователя, которые включают этот автомобиль
	// ВАЖНО: один пользователь может иметь несколько активных пропусков!
	passes, err := s.passRepo.GetActivePassesByUserAndVehicle(ctx, user.ID, vehicle.ID)
	if err != nil {
		s.logger.Error("Failed to get user passes", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to get user passes: %w", err)
	}

	if len(passes) == 0 {
		s.logger.Info("No active passes found for user and vehicle", map[string]interface{}{
			"user_id":    user.ID,
			"vehicle_id": vehicle.ID,
		})
		response.AccessGranted = false
		response.Reason = "No valid pass found for this vehicle"
		s.logAccess(ctx, response, req, vehicle, user, nil)
		return response, nil
	}

	// ШАГ 7: Проверяем временные ограничения для КАЖДОГО пропуска
	// Доступ разрешается, если ХОТЯ БЫ ОДИН пропуск действителен
	var validPass *domain.Pass
	for _, pass := range passes {
		if pass.IsValid() {
			validPass = pass
			break
		}
	}

	if validPass == nil {
		s.logger.Info("All passes are expired or invalid", map[string]interface{}{
			"user_id":      user.ID,
			"passes_count": len(passes),
		})
		response.AccessGranted = false
		response.Reason = "All passes expired or invalid"
		s.logAccess(ctx, response, req, vehicle, user, passes[0])
		return response, nil
	}

	// ШАГ 8: ДОСТУП РАЗРЕШЕН!
	s.logger.Info("Access granted", map[string]interface{}{
		"user_id":    user.ID,
		"vehicle_id": vehicle.ID,
		"pass_id":    validPass.ID,
		"pass_type":  validPass.PassType,
	})

	response.AccessGranted = true
	response.Pass = validPass
	response.Reason = "Valid pass found"

	// Записываем лог доступа
	s.logAccess(ctx, response, req, vehicle, user, validPass)

	return response, nil
}

// logAccess записывает информацию о попытке доступа в БД
func (s *Service) logAccess(
	ctx context.Context,
	response *CheckAccessResponse,
	request *CheckAccessRequest,
	vehicle *domain.Vehicle,
	user *domain.User,
	pass *domain.Pass,
) {
	accessLog := &domain.AccessLog{
		LicensePlate:          response.LicensePlate,
		RecognitionConfidence: response.Confidence,
		AccessGranted:         response.AccessGranted,
		AccessReason:          response.Reason,
		GateID:                request.GateID,
		Direction:             domain.Direction(request.Direction),
		Timestamp:             response.Timestamp,
	}

	if vehicle != nil {
		accessLog.VehicleID = &vehicle.ID
	}

	if user != nil {
		accessLog.UserID = &user.ID
	}

	if err := accessLog.Validate(); err != nil {
		s.logger.Error("Invalid access log data", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if err := s.accessLogRepo.Create(ctx, accessLog); err != nil {
		s.logger.Error("Failed to create access log", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// GetAccessLogs возвращает историю проездов с фильтрацией и пагинацией
func (s *Service) GetAccessLogs(ctx context.Context, userID *uuid.UUID, limit, offset int) ([]*domain.AccessLog, error) {
	if userID != nil {
		return s.accessLogRepo.GetByUserID(ctx, *userID, limit, offset)
	}
	return s.accessLogRepo.List(ctx, limit, offset)
}

// GetAccessLogsByVehicle возвращает историю проездов по автомобилю
func (s *Service) GetAccessLogsByVehicle(ctx context.Context, vehicleID uuid.UUID, limit, offset int) ([]*domain.AccessLog, error) {
	return s.accessLogRepo.GetByVehicleID(ctx, vehicleID, limit, offset)
}
