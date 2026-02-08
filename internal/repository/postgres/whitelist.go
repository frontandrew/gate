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

type whitelistRepository struct {
	db *pgxpool.Pool
}

func NewWhitelistRepository(db *pgxpool.Pool) repository.WhitelistRepository {
	return &whitelistRepository{db: db}
}

func (r *whitelistRepository) Create(ctx context.Context, entry *domain.WhitelistEntry) error {
	query := `
		INSERT INTO whitelist (id, license_plate, reason, added_by, added_at, expires_at, is_active)
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

func (r *whitelistRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WhitelistEntry, error) {
	query := `
		SELECT id, license_plate, reason, added_by, added_at, expires_at, is_active
		FROM whitelist
		WHERE id = $1
	`

	entry := &domain.WhitelistEntry{}
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
			return nil, domain.ErrWhitelistEntryNotFound
		}
		return nil, err
	}

	return entry, nil
}

func (r *whitelistRepository) GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.WhitelistEntry, error) {
	query := `
		SELECT id, license_plate, reason, added_by, added_at, expires_at, is_active
		FROM whitelist
		WHERE license_plate = $1 AND is_active = true
	`

	normalizedPlate := domain.NormalizeLicensePlate(licensePlate)

	entry := &domain.WhitelistEntry{}
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
			return nil, domain.ErrWhitelistEntryNotFound
		}
		return nil, err
	}

	return entry, nil
}

// IsWhitelisted - КРИТИЧНЫЙ МЕТОД для проверки доступа (ВЫСШИЙ ПРИОРИТЕТ)
func (r *whitelistRepository) IsWhitelisted(ctx context.Context, licensePlate string) (bool, string, error) {
	query := `
		SELECT reason
		FROM whitelist
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
			// Номера нет в белом списке - это нормально
			return false, "", nil
		}
		return false, "", err
	}

	// Номер найден в белом списке - БЕЗУСЛОВНЫЙ ДОСТУП!
	return true, reason, nil
}

func (r *whitelistRepository) Update(ctx context.Context, entry *domain.WhitelistEntry) error {
	query := `
		UPDATE whitelist
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
		return domain.ErrWhitelistEntryNotFound
	}

	return nil
}

func (r *whitelistRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM whitelist WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrWhitelistEntryNotFound
	}

	return nil
}

func (r *whitelistRepository) List(ctx context.Context, limit, offset int) ([]*domain.WhitelistEntry, error) {
	query := `
		SELECT id, license_plate, reason, added_by, added_at, expires_at, is_active
		FROM whitelist
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

func (r *whitelistRepository) GetExpired(ctx context.Context) ([]*domain.WhitelistEntry, error) {
	query := `
		SELECT id, license_plate, reason, added_by, added_at, expires_at, is_active
		FROM whitelist
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

func (r *whitelistRepository) scanEntries(rows pgx.Rows) ([]*domain.WhitelistEntry, error) {
	var entries []*domain.WhitelistEntry
	for rows.Next() {
		entry := &domain.WhitelistEntry{}
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
