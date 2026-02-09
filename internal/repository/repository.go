package repository

import (
	"context"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/google/uuid"
)

// UserRepository определяет методы для работы с пользователями
type UserRepository interface {
	// Create создает нового пользователя
	Create(ctx context.Context, user *domain.User) error

	// GetByID возвращает пользователя по ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetByEmail возвращает пользователя по email
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update обновляет данные пользователя
	Update(ctx context.Context, user *domain.User) error

	// Delete удаляет пользователя (мягкое удаление - is_active = false)
	Delete(ctx context.Context, id uuid.UUID) error

	// List возвращает список пользователей с пагинацией
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)

	// UpdateLastLogin обновляет время последнего входа
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// VehicleRepository определяет методы для работы с автомобилями
type VehicleRepository interface {
	// Create создает новый автомобиль
	Create(ctx context.Context, vehicle *domain.Vehicle) error

	// GetByID возвращает автомобиль по ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error)

	// GetByLicensePlate возвращает автомобиль по номеру
	GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.Vehicle, error)

	// GetByOwnerID возвращает все автомобили пользователя
	GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*domain.Vehicle, error)

	// Update обновляет данные автомобиля
	Update(ctx context.Context, vehicle *domain.Vehicle) error

	// Delete удаляет автомобиль (мягкое удаление - is_active = false)
	Delete(ctx context.Context, id uuid.UUID) error

	// List возвращает список автомобилей с пагинацией
	List(ctx context.Context, limit, offset int) ([]*domain.Vehicle, error)
}

// PassRepository определяет методы для работы с пропусками
type PassRepository interface {
	// Create создает новый пропуск
	Create(ctx context.Context, pass *domain.Pass) error

	// GetByID возвращает пропуск по ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Pass, error)

	// GetByUserID возвращает все пропуска пользователя
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Pass, error)

	// GetActivePassesByUser возвращает все активные пропуска пользователя
	GetActivePassesByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Pass, error)

	// GetActivePassesByUserAndVehicle возвращает активные пропуска пользователя, включающие указанный автомобиль
	// КЛЮЧЕВОЙ МЕТОД для проверки доступа
	GetActivePassesByUserAndVehicle(ctx context.Context, userID, vehicleID uuid.UUID) ([]*domain.Pass, error)

	// Update обновляет данные пропуска
	Update(ctx context.Context, pass *domain.Pass) error

	// Revoke отзывает пропуск
	Revoke(ctx context.Context, id, revokedBy uuid.UUID, reason string) error

	// List возвращает список всех пропусков с пагинацией
	List(ctx context.Context, limit, offset int) ([]*domain.Pass, error)

	// GetExpiredPasses возвращает истекшие временные пропуска
	GetExpiredPasses(ctx context.Context) ([]*domain.Pass, error)
}

// PassVehicleRepository определяет методы для работы со связями пропуск-автомобиль
type PassVehicleRepository interface {
	// Create создает связь пропуск-автомобиль
	Create(ctx context.Context, passVehicle *domain.PassVehicle) error

	// GetByID возвращает связь по ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PassVehicle, error)

	// GetByPassID возвращает все автомобили, привязанные к пропуску
	GetByPassID(ctx context.Context, passID uuid.UUID) ([]*domain.PassVehicle, error)

	// GetByVehicleID возвращает все пропуска, к которым привязан автомобиль
	GetByVehicleID(ctx context.Context, vehicleID uuid.UUID) ([]*domain.PassVehicle, error)

	// Delete удаляет связь пропуск-автомобиль
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByPassAndVehicle удаляет связь по pass_id и vehicle_id
	DeleteByPassAndVehicle(ctx context.Context, passID, vehicleID uuid.UUID) error
}

// AccessLogRepository определяет методы для работы с логами доступа
type AccessLogRepository interface {
	// Create создает новую запись в логе доступа
	Create(ctx context.Context, log *domain.AccessLog) error

	// GetByID возвращает запись лога по ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AccessLog, error)

	// GetByUserID возвращает историю проездов пользователя
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AccessLog, error)

	// GetByVehicleID возвращает историю проездов автомобиля
	GetByVehicleID(ctx context.Context, vehicleID uuid.UUID, limit, offset int) ([]*domain.AccessLog, error)

	// GetByLicensePlate возвращает историю проездов по номеру автомобиля
	GetByLicensePlate(ctx context.Context, licensePlate string, limit, offset int) ([]*domain.AccessLog, error)

	// List возвращает список всех логов с пагинацией
	List(ctx context.Context, limit, offset int) ([]*domain.AccessLog, error)

	// GetStatsByPeriod возвращает статистику проездов за период
	GetStatsByPeriod(ctx context.Context, from, to string) (map[string]interface{}, error)
}

// BlacklistRepository определяет методы для работы с черным списком
type BlacklistRepository interface {
	// Create создает новую запись в черном списке
	Create(ctx context.Context, entry *domain.BlacklistEntry) error

	// GetByID возвращает запись по ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BlacklistEntry, error)

	// GetByLicensePlate возвращает запись по номеру автомобиля
	GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.BlacklistEntry, error)

	// IsBlacklisted проверяет, находится ли номер в черном списке
	// Возвращает (isBlacklisted, reason, error)
	IsBlacklisted(ctx context.Context, licensePlate string) (bool, string, error)

	// Update обновляет запись
	Update(ctx context.Context, entry *domain.BlacklistEntry) error

	// Delete удаляет запись
	Delete(ctx context.Context, id uuid.UUID) error

	// List возвращает список с пагинацией
	List(ctx context.Context, limit, offset int) ([]*domain.BlacklistEntry, error)

	// GetExpired возвращает истекшие записи для удаления
	GetExpired(ctx context.Context) ([]*domain.BlacklistEntry, error)
}

// WhitelistRepository определяет методы для работы с белым списком
type WhitelistRepository interface {
	// Create создает новую запись в белом списке
	Create(ctx context.Context, entry *domain.WhitelistEntry) error

	// GetByID возвращает запись по ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WhitelistEntry, error)

	// GetByLicensePlate возвращает запись по номеру автомобиля
	GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.WhitelistEntry, error)

	// IsWhitelisted проверяет, находится ли номер в белом списке
	// Возвращает (isWhitelisted, reason, error)
	IsWhitelisted(ctx context.Context, licensePlate string) (bool, string, error)

	// Update обновляет запись
	Update(ctx context.Context, entry *domain.WhitelistEntry) error

	// Delete удаляет запись
	Delete(ctx context.Context, id uuid.UUID) error

	// List возвращает список с пагинацией
	List(ctx context.Context, limit, offset int) ([]*domain.WhitelistEntry, error)

	// GetExpired возвращает истекшие записи для удаления
	GetExpired(ctx context.Context) ([]*domain.WhitelistEntry, error)
}

// RefreshTokenRepository определяет методы для работы с refresh токенами
type RefreshTokenRepository interface {
	// Create сохраняет новый refresh token
	Create(ctx context.Context, token *domain.RefreshToken) error

	// GetByTokenHash возвращает refresh token по хешу
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)

	// Revoke отзывает refresh token
	Revoke(ctx context.Context, tokenHash string) error

	// RevokeAllUserTokens отзывает все токены пользователя
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error

	// DeleteExpired удаляет истекшие токены
	DeleteExpired(ctx context.Context) error
}
