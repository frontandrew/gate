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

// IsInBlacklist проверяет, находится ли номер в blacklist (с кэшированием)
func (r *BlacklistRepository) IsInBlacklist(ctx context.Context, licensePlate string) (bool, error) {
	// Формируем ключ кэша
	cacheKey := blacklistCachePrefix + licensePlate

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
	inBlacklist, err := r.repo.IsInBlacklist(ctx, licensePlate)
	if err != nil {
		return false, err
	}

	// 3. Сохраняем результат в кэш
	cacheValue := "0"
	if inBlacklist {
		cacheValue = "1"
	}

	// Игнорируем ошибку записи в кэш (не критично)
	_ = r.cache.Set(ctx, cacheKey, cacheValue, blacklistCacheTTL)

	return inBlacklist, nil
}

// Create добавляет запись в blacklist и инвалидирует кэш
func (r *BlacklistRepository) Create(ctx context.Context, blacklist *domain.Blacklist) error {
	// Создаем запись в БД
	if err := r.repo.Create(ctx, blacklist); err != nil {
		return err
	}

	// Инвалидируем кэш для этого номера
	cacheKey := blacklistCachePrefix + blacklist.LicensePlate
	_ = r.cache.Del(ctx, cacheKey)

	return nil
}

// GetByLicensePlate получает запись по номеру
func (r *BlacklistRepository) GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.Blacklist, error) {
	// Для полных данных не кэшируем - используется редко
	return r.repo.GetByLicensePlate(ctx, licensePlate)
}

// GetAll получает все записи
func (r *BlacklistRepository) GetAll(ctx context.Context, limit, offset int) ([]*domain.Blacklist, error) {
	// Списки не кэшируем - используются редко (только для админки)
	return r.repo.GetAll(ctx, limit, offset)
}

// Delete удаляет запись и инвалидирует кэш
func (r *BlacklistRepository) Delete(ctx context.Context, licensePlate string) error {
	// Удаляем из БД
	if err := r.repo.Delete(ctx, licensePlate); err != nil {
		return err
	}

	// Инвалидируем кэш
	cacheKey := blacklistCachePrefix + licensePlate
	_ = r.cache.Del(ctx, cacheKey)

	return nil
}

// DeleteExpired удаляет истекшие записи и сбрасывает весь кэш blacklist
func (r *BlacklistRepository) DeleteExpired(ctx context.Context) error {
	// Удаляем истекшие записи из БД
	if err := r.repo.DeleteExpired(ctx); err != nil {
		return err
	}

	// Сбрасываем весь кэш blacklist
	pattern := blacklistCachePrefix + "*"

	// Получаем все ключи по паттерну
	iter := r.cache.GetClient().Scan(ctx, 0, pattern, 0).Iterator()
	keys := []string{}
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan blacklist cache keys: %w", err)
	}

	// Удаляем найденные ключи
	if len(keys) > 0 {
		_ = r.cache.Del(ctx, keys...)
	}

	return nil
}
