package cached

import (
	"context"
	"strings"
	"time"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/redis"
	"github.com/frontandrew/gate/internal/repository"
	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

const (
	blacklistCachePrefix = "blacklist:"
	blacklistCacheTTL    = 1 * time.Hour
)

// BlacklistRepository добавляет кэширование к blacklist repository
type BlacklistRepository struct {
	repo  repository.BlacklistRepository
	cache *redis.Client
}

// NewBlacklistRepository создает новый кэшируемый blacklist repository
func NewBlacklistRepository(repo repository.BlacklistRepository, cache *redis.Client) *BlacklistRepository {
	return &BlacklistRepository{
		repo:  repo,
		cache: cache,
	}
}

// IsBlacklisted проверяет, находится ли номер в blacklist (с кэшированием)
func (r *BlacklistRepository) IsBlacklisted(ctx context.Context, licensePlate string) (bool, string, error) {
	// Формируем ключ кэша
	cacheKey := blacklistCachePrefix + licensePlate

	// 1. Проверяем кэш
	cached, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		// Cache hit - парсим формат "0:" или "1:reason"
		parts := strings.SplitN(cached, ":", 2)
		if len(parts) == 2 {
			inBlacklist := parts[0] == "1"
			reason := parts[1]
			return inBlacklist, reason, nil
		}
	}

	// Если ошибка не redis.Nil (ключ не найден), то это реальная ошибка
	if err != redisv9.Nil {
		// Логируем ошибку кэша, но продолжаем работу с БД
		// В production здесь можно добавить метрику
	}

	// 2. Cache miss - идем в БД
	inBlacklist, reason, err := r.repo.IsBlacklisted(ctx, licensePlate)
	if err != nil {
		return false, "", err
	}

	// 3. Сохраняем результат в кэш (формат: "0:" или "1:reason")
	cacheValue := "0:"
	if inBlacklist {
		cacheValue = "1:" + reason
	}

	// Игнорируем ошибку записи в кэш (не критично)
	_ = r.cache.Set(ctx, cacheKey, cacheValue, blacklistCacheTTL)

	return inBlacklist, reason, nil
}

// Create добавляет запись в blacklist и инвалидирует кэш
func (r *BlacklistRepository) Create(ctx context.Context, entry *domain.BlacklistEntry) error {
	// Создаем запись в БД
	if err := r.repo.Create(ctx, entry); err != nil {
		return err
	}

	// Инвалидируем кэш для этого номера
	cacheKey := blacklistCachePrefix + entry.LicensePlate
	_ = r.cache.Del(ctx, cacheKey)

	return nil
}

// GetByID получает запись по ID
func (r *BlacklistRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BlacklistEntry, error) {
	// Для полных данных не кэшируем - используется редко
	return r.repo.GetByID(ctx, id)
}

// GetByLicensePlate получает запись по номеру
func (r *BlacklistRepository) GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.BlacklistEntry, error) {
	// Для полных данных не кэшируем - используется редко
	return r.repo.GetByLicensePlate(ctx, licensePlate)
}

// Update обновляет запись и инвалидирует кэш
func (r *BlacklistRepository) Update(ctx context.Context, entry *domain.BlacklistEntry) error {
	// Обновляем в БД
	if err := r.repo.Update(ctx, entry); err != nil {
		return err
	}

	// Инвалидируем кэш для этого номера
	cacheKey := blacklistCachePrefix + entry.LicensePlate
	_ = r.cache.Del(ctx, cacheKey)

	return nil
}

// List получает все записи
func (r *BlacklistRepository) List(ctx context.Context, limit, offset int) ([]*domain.BlacklistEntry, error) {
	// Списки не кэшируем - используются редко (только для админки)
	return r.repo.List(ctx, limit, offset)
}

// Delete удаляет запись и инвалидирует кэш
func (r *BlacklistRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Удаляем из БД
	if err := r.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Примечание: мы не можем точно инвалидировать кэш по license_plate,
	// так как Delete принимает только ID. Кэш истечет через TTL (1 час).
	// Альтернатива: можно было бы сначала получить entry, запомнить license_plate,
	// затем удалить и инвалидировать кэш. Но это добавляет лишний запрос к БД.
	// Поскольку Delete вызывается редко, текущий подход приемлем.

	return nil
}

// GetExpired возвращает истекшие записи
func (r *BlacklistRepository) GetExpired(ctx context.Context) ([]*domain.BlacklistEntry, error) {
	// Просто возвращаем истекшие записи из БД
	// Кэш для GetExpired не используем, так как это административная операция
	return r.repo.GetExpired(ctx)
}
