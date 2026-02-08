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

type passRepository struct {
	db *pgxpool.Pool
}

func NewPassRepository(db *pgxpool.Pool) repository.PassRepository {
	return &passRepository{db: db}
}

func (r *passRepository) Create(ctx context.Context, pass *domain.Pass) error {
	query := `
		INSERT INTO passes (id, user_id, pass_type, valid_from, valid_until, is_active, created_at, created_by, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	pass.ID = uuid.New()
	pass.CreatedAt = time.Now()
	pass.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		pass.ID,
		pass.UserID,
		pass.PassType,
		pass.ValidFrom,
		pass.ValidUntil,
		pass.IsActive,
		pass.CreatedAt,
		pass.CreatedBy,
		pass.UpdatedAt,
	)

	return err
}

func (r *passRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Pass, error) {
	query := `
		SELECT id, user_id, pass_type, valid_from, valid_until, is_active,
		       revoked_at, revoked_by, revoke_reason, created_at, created_by, updated_at
		FROM passes
		WHERE id = $1
	`

	pass := &domain.Pass{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&pass.ID,
		&pass.UserID,
		&pass.PassType,
		&pass.ValidFrom,
		&pass.ValidUntil,
		&pass.IsActive,
		&pass.RevokedAt,
		&pass.RevokedBy,
		&pass.RevokeReason,
		&pass.CreatedAt,
		&pass.CreatedBy,
		&pass.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPassNotFound
		}
		return nil, err
	}

	return pass, nil
}

func (r *passRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Pass, error) {
	query := `
		SELECT id, user_id, pass_type, valid_from, valid_until, is_active,
		       revoked_at, revoked_by, revoke_reason, created_at, created_by, updated_at
		FROM passes
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPasses(rows)
}

func (r *passRepository) GetActivePassesByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Pass, error) {
	query := `
		SELECT id, user_id, pass_type, valid_from, valid_until, is_active,
		       revoked_at, revoked_by, revoke_reason, created_at, created_by, updated_at
		FROM passes
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPasses(rows)
}

// GetActivePassesByUserAndVehicle - КЛЮЧЕВОЙ МЕТОД для проверки доступа
// Возвращает все активные пропуска пользователя, которые включают указанный автомобиль
func (r *passRepository) GetActivePassesByUserAndVehicle(ctx context.Context, userID, vehicleID uuid.UUID) ([]*domain.Pass, error) {
	query := `
		SELECT DISTINCT p.id, p.user_id, p.pass_type, p.valid_from, p.valid_until, p.is_active,
		       p.revoked_at, p.revoked_by, p.revoke_reason, p.created_at, p.created_by, p.updated_at
		FROM passes p
		INNER JOIN pass_vehicles pv ON p.id = pv.pass_id
		WHERE p.user_id = $1
		  AND pv.vehicle_id = $2
		  AND p.is_active = true
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPasses(rows)
}

func (r *passRepository) Update(ctx context.Context, pass *domain.Pass) error {
	query := `
		UPDATE passes
		SET user_id = $2, pass_type = $3, valid_from = $4, valid_until = $5, is_active = $6,
		    revoked_at = $7, revoked_by = $8, revoke_reason = $9, updated_at = $10
		WHERE id = $1
	`

	pass.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, query,
		pass.ID,
		pass.UserID,
		pass.PassType,
		pass.ValidFrom,
		pass.ValidUntil,
		pass.IsActive,
		pass.RevokedAt,
		pass.RevokedBy,
		pass.RevokeReason,
		pass.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrPassNotFound
	}

	return nil
}

func (r *passRepository) Revoke(ctx context.Context, id, revokedBy uuid.UUID, reason string) error {
	query := `
		UPDATE passes
		SET is_active = false, revoked_at = $2, revoked_by = $3, revoke_reason = $4, updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(ctx, query, id, now, revokedBy, reason)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrPassNotFound
	}

	return nil
}

func (r *passRepository) List(ctx context.Context, limit, offset int) ([]*domain.Pass, error) {
	query := `
		SELECT id, user_id, pass_type, valid_from, valid_until, is_active,
		       revoked_at, revoked_by, revoke_reason, created_at, created_by, updated_at
		FROM passes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPasses(rows)
}

func (r *passRepository) GetExpiredPasses(ctx context.Context) ([]*domain.Pass, error) {
	query := `
		SELECT id, user_id, pass_type, valid_from, valid_until, is_active,
		       revoked_at, revoked_by, revoke_reason, created_at, created_by, updated_at
		FROM passes
		WHERE pass_type = 'temporary'
		  AND is_active = true
		  AND valid_until < NOW()
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPasses(rows)
}

// scanPasses - вспомогательная функция для сканирования результатов запроса
func (r *passRepository) scanPasses(rows pgx.Rows) ([]*domain.Pass, error) {
	var passes []*domain.Pass
	for rows.Next() {
		pass := &domain.Pass{}
		err := rows.Scan(
			&pass.ID,
			&pass.UserID,
			&pass.PassType,
			&pass.ValidFrom,
			&pass.ValidUntil,
			&pass.IsActive,
			&pass.RevokedAt,
			&pass.RevokedBy,
			&pass.RevokeReason,
			&pass.CreatedAt,
			&pass.CreatedBy,
			&pass.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		passes = append(passes, pass)
	}

	return passes, nil
}
