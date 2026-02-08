package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type blacklistRepository struct {
	db *pgxpool.Pool
}

func NewBlacklistRepository(db *pgxpool.Pool) repository.BlacklistRepository {
	return &blacklistRepository{db: db}
}

func (r *blacklistRepository) Create(ctx context.Context, entry *domain.BlacklistEntry) error {
	query := `
		INSERT INTO blacklist (id, license_plate, reason, added_by, added_at, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	entry.ID = uuid.New()
	entry.AddedAt = time.Now()

	// Нормализуем номер
	entry.LicensePlate = domain.NormalizeLicensePlate(entry.LicensePlate)

	_, err := r.db.Exec(ctx, query,
		entry.ID,
		entry.LicensePlate,
		entry.Reason,
		entry.AddedBy,
		entry.AddedAt,
		entry.ExpiresAt,
		entry.IsActive,
	)

	return err
}

func (r *blacklistRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BlacklistEntry, error) {
	query := `
		SELECT id, license_plate, reason, added_by, added_at, expires_at, is_active
		FROM blacklist
		WHERE id = $1
	`

	entry := &domain.BlacklistEntry{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&entry.ID,
		&entry.LicensePlate,
		&entry.Reason,
		&entry.AddedBy,
		&entry.AddedAt,
		&entry.ExpiresAt,
		&entry.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBlacklistEntryNotFound
		}
		return nil, err
	}

	return entry, nil
}

func (r *blacklistRepository) GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.BlacklistEntry, error) {
	query := `
		SELECT id, license_plate, reason, added_by, added_at, expires_at, is_active
		FROM blacklist
		WHERE license_plate = $1 AND is_active = true
	`

	normalizedPlate := domain.NormalizeLicensePlate(licensePlate)

	entry := &domain.BlacklistEntry{}
	err := r.db.QueryRow(ctx, query, normalizedPlate).Scan(
		&entry.ID,
		&entry.LicensePlate,
		&entry.Reason,
		&entry.AddedBy,
		&entry.AddedAt,
		&entry.ExpiresAt,
		&entry.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBlacklistEntryNotFound
		}
		return nil, err
	}

	return entry, nil
}

// IsBlacklisted - КРИТИЧНЫЙ МЕТОД для проверки доступа
func (r *blacklistRepository) IsBlacklisted(ctx context.Context, licensePlate string) (bool, string, error) {
	query := `
		SELECT reason
		FROM blacklist
		WHERE license_plate = $1
		  AND is_active = true
		  AND (expires_at IS NULL OR expires_at > NOW())
		LIMIT 1
	`

	normalizedPlate := domain.NormalizeLicensePlate(licensePlate)

	var reason string
	err := r.db.QueryRow(ctx, query, normalizedPlate).Scan(&reason)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Номера нет в черном списке - это нормально
			return false, "", nil
		}
		return false, "", err
	}

	// Номер найден в черном списке
	return true, reason, nil
}

func (r *blacklistRepository) Update(ctx context.Context, entry *domain.BlacklistEntry) error {
	query := `
		UPDATE blacklist
		SET license_plate = $2, reason = $3, expires_at = $4, is_active = $5
		WHERE id = $1
	`

	entry.LicensePlate = domain.NormalizeLicensePlate(entry.LicensePlate)

	result, err := r.db.Exec(ctx, query,
		entry.ID,
		entry.LicensePlate,
		entry.Reason,
		entry.ExpiresAt,
		entry.IsActive,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBlacklistEntryNotFound
	}

	return nil
}

func (r *blacklistRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM blacklist WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBlacklistEntryNotFound
	}

	return nil
}

func (r *blacklistRepository) List(ctx context.Context, limit, offset int) ([]*domain.BlacklistEntry, error) {
	query := `
		SELECT id, license_plate, reason, added_by, added_at, expires_at, is_active
		FROM blacklist
		ORDER BY added_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEntries(rows)
}

func (r *blacklistRepository) GetExpired(ctx context.Context) ([]*domain.BlacklistEntry, error) {
	query := `
		SELECT id, license_plate, reason, added_by, added_at, expires_at, is_active
		FROM blacklist
		WHERE is_active = true
		  AND expires_at IS NOT NULL
		  AND expires_at < NOW()
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEntries(rows)
}

func (r *blacklistRepository) scanEntries(rows pgx.Rows) ([]*domain.BlacklistEntry, error) {
	var entries []*domain.BlacklistEntry
	for rows.Next() {
		entry := &domain.BlacklistEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.LicensePlate,
			&entry.Reason,
			&entry.AddedBy,
			&entry.AddedAt,
			&entry.ExpiresAt,
			&entry.IsActive,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
