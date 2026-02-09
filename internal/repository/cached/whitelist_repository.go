package cached

import (
	"context"
	"fmt"
	"time"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/redis"
	"github.com/frontandrew/gate/internal/repository"
	redisv9 "github.com/redis/go-redis/v9"
)

const (
	whitelistCachePrefix = "whitelist:"
	whitelistCacheTTL    = 1 * time.Hour
)

// WhitelistRepository добавляет кэширование к whitelist repository
type WhitelistRepository struct {
	repo  repository.WhitelistRepository
	cache *redis.Client
}

// NewWhitelistRepository создает новый кэшируемый whitelist repository
func NewWhitelistRepository(repo repository.WhitelistRepository, cache *redis.Client) *WhitelistRepository {
	return &WhitelistRepository{
		repo:  repo,
		cache: cache,
	}
}

// IsInWhitelist проверяет, находится ли номер в whitelist (с кэшированием)
func (r *WhitelistRepository) IsInWhitelist(ctx context.Context, licensePlate string) (bool, error) {
	// Формируем ключ кэша
	cacheKey := whitelistCachePrefix + licensePlate

	// 1. Проверяем кэш
	cached, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		// Cache hit
		return cached == "1", nil
	}

	// Если ошибка не redis.Nil (ключ не найден), то это реальная ошибка
	if err != redisv9.Nil {
		// Логируем ошибку кэша, но продолжаем работу с БД
		// В production здесь можно добавить метрику
	}

	// 2. Cache miss - идем в БД
	inWhitelist, err := r.repo.IsInWhitelist(ctx, licensePlate)
	if err != nil {
		return false, err
	}

	// 3. Сохраняем результат в кэш
	cacheValue := "0"
	if inWhitelist {
		cacheValue = "1"
	}

	// Игнорируем ошибку записи в кэш (не критично)
	_ = r.cache.Set(ctx, cacheKey, cacheValue, whitelistCacheTTL)

	return inWhitelist, nil
}

// Create добавляет запись в whitelist и инвалидирует кэш
func (r *WhitelistRepository) Create(ctx context.Context, whitelist *domain.Whitelist) error {
	// Создаем запись в БД
	if err := r.repo.Create(ctx, whitelist); err != nil {
		return err
	}

	// Инвалидируем кэш для этого номера
	cacheKey := whitelistCachePrefix + whitelist.LicensePlate
	_ = r.cache.Del(ctx, cacheKey)

	return nil
}

// GetByLicensePlate получает запись по номеру
func (r *WhitelistRepository) GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.Whitelist, error) {
	// Для полных данных не кэшируем - используется редко
	return r.repo.GetByLicensePlate(ctx, licensePlate)
}

// GetAll получает все записи
func (r *WhitelistRepository) GetAll(ctx context.Context, limit, offset int) ([]*domain.Whitelist, error) {
	// Списки не кэшируем - используются редко (только для админки)
	return r.repo.GetAll(ctx, limit, offset)
}

// Delete удаляет запись и инвалидирует кэш
func (r *WhitelistRepository) Delete(ctx context.Context, licensePlate string) error {
	// Удаляем из БД
	if err := r.repo.Delete(ctx, licensePlate); err != nil {
		return err
	}

	// Инвалидируем кэш
	cacheKey := whitelistCachePrefix + licensePlate
	_ = r.cache.Del(ctx, cacheKey)

	return nil
}

// DeleteExpired удаляет истекшие записи и сбрасывает весь кэш whitelist
func (r *WhitelistRepository) DeleteExpired(ctx context.Context) error {
	// Удаляем истекшие записи из БД
	if err := r.repo.DeleteExpired(ctx); err != nil {
		return err
	}

	// Сбрасываем весь кэш whitelist (это происходит редко - раз в день)
	// В production здесь можно использовать SCAN для удаления только whitelist:* ключей
	pattern := whitelistCachePrefix + "*"

	// Получаем все ключи по паттерну
	iter := r.cache.GetClient().Scan(ctx, 0, pattern, 0).Iterator()
	keys := []string{}
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan whitelist cache keys: %w", err)
	}

	// Удаляем найденные ключи
	if len(keys) > 0 {
		_ = r.cache.Del(ctx, keys...)
	}

	return nil
}
