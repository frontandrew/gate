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

type passVehicleRepository struct {
	db *pgxpool.Pool
}

func NewPassVehicleRepository(db *pgxpool.Pool) repository.PassVehicleRepository {
	return &passVehicleRepository{db: db}
}

func (r *passVehicleRepository) Create(ctx context.Context, passVehicle *domain.PassVehicle) error {
	query := `
		INSERT INTO pass_vehicles (id, pass_id, vehicle_id, added_at, added_by)
		VALUES ($1, $2, $3, $4, $5)
	`

	passVehicle.ID = uuid.New()
	passVehicle.AddedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		passVehicle.ID,
		passVehicle.PassID,
		passVehicle.VehicleID,
		passVehicle.AddedAt,
		passVehicle.AddedBy,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *passVehicleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PassVehicle, error) {
	query := `
		SELECT id, pass_id, vehicle_id, added_at, added_by
		FROM pass_vehicles
		WHERE id = $1
	`

	passVehicle := &domain.PassVehicle{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&passVehicle.ID,
		&passVehicle.PassID,
		&passVehicle.VehicleID,
		&passVehicle.AddedAt,
		&passVehicle.AddedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPassVehicleNotFound
		}
		return nil, err
	}

	return passVehicle, nil
}

func (r *passVehicleRepository) GetByPassID(ctx context.Context, passID uuid.UUID) ([]*domain.PassVehicle, error) {
	query := `
		SELECT id, pass_id, vehicle_id, added_at, added_by
		FROM pass_vehicles
		WHERE pass_id = $1
		ORDER BY added_at DESC
	`

	rows, err := r.db.Query(ctx, query, passID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPassVehicles(rows)
}

func (r *passVehicleRepository) GetByVehicleID(ctx context.Context, vehicleID uuid.UUID) ([]*domain.PassVehicle, error) {
	query := `
		SELECT id, pass_id, vehicle_id, added_at, added_by
		FROM pass_vehicles
		WHERE vehicle_id = $1
		ORDER BY added_at DESC
	`

	rows, err := r.db.Query(ctx, query, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPassVehicles(rows)
}

func (r *passVehicleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM pass_vehicles WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrPassVehicleNotFound
	}

	return nil
}

func (r *passVehicleRepository) DeleteByPassAndVehicle(ctx context.Context, passID, vehicleID uuid.UUID) error {
	query := `DELETE FROM pass_vehicles WHERE pass_id = $1 AND vehicle_id = $2`

	result, err := r.db.Exec(ctx, query, passID, vehicleID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrPassVehicleNotFound
	}

	return nil
}

func (r *passVehicleRepository) scanPassVehicles(rows pgx.Rows) ([]*domain.PassVehicle, error) {
	var passVehicles []*domain.PassVehicle
	for rows.Next() {
		pv := &domain.PassVehicle{}
		err := rows.Scan(
			&pv.ID,
			&pv.PassID,
			&pv.VehicleID,
			&pv.AddedAt,
			&pv.AddedBy,
		)
		if err != nil {
			return nil, err
		}
		passVehicles = append(passVehicles, pv)
	}

	return passVehicles, nil
}
