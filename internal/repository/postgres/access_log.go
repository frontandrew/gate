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

type accessLogRepository struct {
	db *pgxpool.Pool
}

func NewAccessLogRepository(db *pgxpool.Pool) repository.AccessLogRepository {
	return &accessLogRepository{db: db}
}

func (r *accessLogRepository) Create(ctx context.Context, log *domain.AccessLog) error {
	query := `
		INSERT INTO access_logs (id, user_id, vehicle_id, license_plate, image_url, recognition_confidence,
		                        access_granted, access_reason, gate_id, direction, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	log.ID = uuid.New()
	log.Timestamp = time.Now()

	_, err := r.db.Exec(ctx, query,
		log.ID,
		log.UserID,
		log.VehicleID,
		log.LicensePlate,
		log.ImageURL,
		log.RecognitionConfidence,
		log.AccessGranted,
		log.AccessReason,
		log.GateID,
		log.Direction,
		log.Timestamp,
	)

	return err
}

func (r *accessLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AccessLog, error) {
	query := `
		SELECT id, user_id, vehicle_id, license_plate, image_url, recognition_confidence,
		       access_granted, access_reason, gate_id, direction, timestamp
		FROM access_logs
		WHERE id = $1
	`

	log := &domain.AccessLog{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&log.ID,
		&log.UserID,
		&log.VehicleID,
		&log.LicensePlate,
		&log.ImageURL,
		&log.RecognitionConfidence,
		&log.AccessGranted,
		&log.AccessReason,
		&log.GateID,
		&log.Direction,
		&log.Timestamp,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccessLogNotFound
		}
		return nil, err
	}

	return log, nil
}

func (r *accessLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AccessLog, error) {
	query := `
		SELECT id, user_id, vehicle_id, license_plate, image_url, recognition_confidence,
		       access_granted, access_reason, gate_id, direction, timestamp
		FROM access_logs
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAccessLogs(rows)
}

func (r *accessLogRepository) GetByVehicleID(ctx context.Context, vehicleID uuid.UUID, limit, offset int) ([]*domain.AccessLog, error) {
	query := `
		SELECT id, user_id, vehicle_id, license_plate, image_url, recognition_confidence,
		       access_granted, access_reason, gate_id, direction, timestamp
		FROM access_logs
		WHERE vehicle_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, vehicleID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAccessLogs(rows)
}

func (r *accessLogRepository) GetByLicensePlate(ctx context.Context, licensePlate string, limit, offset int) ([]*domain.AccessLog, error) {
	query := `
		SELECT id, user_id, vehicle_id, license_plate, image_url, recognition_confidence,
		       access_granted, access_reason, gate_id, direction, timestamp
		FROM access_logs
		WHERE license_plate = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, licensePlate, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAccessLogs(rows)
}

func (r *accessLogRepository) List(ctx context.Context, limit, offset int) ([]*domain.AccessLog, error) {
	query := `
		SELECT id, user_id, vehicle_id, license_plate, image_url, recognition_confidence,
		       access_granted, access_reason, gate_id, direction, timestamp
		FROM access_logs
		ORDER BY timestamp DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAccessLogs(rows)
}

func (r *accessLogRepository) GetStatsByPeriod(ctx context.Context, from, to string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			SUM(CASE WHEN access_granted = true THEN 1 ELSE 0 END) as granted_count,
			SUM(CASE WHEN access_granted = false THEN 1 ELSE 0 END) as denied_count,
			AVG(recognition_confidence) as avg_confidence
		FROM access_logs
		WHERE timestamp BETWEEN $1 AND $2
	`

	var totalCount, grantedCount, deniedCount int
	var avgConfidence float64

	err := r.db.QueryRow(ctx, query, from, to).Scan(
		&totalCount,
		&grantedCount,
		&deniedCount,
		&avgConfidence,
	)

	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_count":    totalCount,
		"granted_count":  grantedCount,
		"denied_count":   deniedCount,
		"avg_confidence": avgConfidence,
	}

	return stats, nil
}

func (r *accessLogRepository) scanAccessLogs(rows pgx.Rows) ([]*domain.AccessLog, error) {
	var logs []*domain.AccessLog
	for rows.Next() {
		log := &domain.AccessLog{}
		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.VehicleID,
			&log.LicensePlate,
			&log.ImageURL,
			&log.RecognitionConfidence,
			&log.AccessGranted,
			&log.AccessReason,
			&log.GateID,
			&log.Direction,
			&log.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}
