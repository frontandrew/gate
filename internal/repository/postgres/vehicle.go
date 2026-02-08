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

type vehicleRepository struct {
	db *pgxpool.Pool
}

func NewVehicleRepository(db *pgxpool.Pool) repository.VehicleRepository {
	return &vehicleRepository{db: db}
}

func (r *vehicleRepository) Create(ctx context.Context, vehicle *domain.Vehicle) error {
	query := `
		INSERT INTO vehicles (id, owner_id, license_plate, vehicle_type, model, color, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	vehicle.ID = uuid.New()
	vehicle.CreatedAt = time.Now()
	vehicle.UpdatedAt = time.Now()

	// Нормализуем номер перед сохранением
	vehicle.LicensePlate = domain.NormalizeLicensePlate(vehicle.LicensePlate)

	_, err := r.db.Exec(ctx, query,
		vehicle.ID,
		vehicle.OwnerID,
		vehicle.LicensePlate,
		vehicle.VehicleType,
		vehicle.Model,
		vehicle.Color,
		vehicle.IsActive,
		vehicle.CreatedAt,
		vehicle.UpdatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *vehicleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error) {
	query := `
		SELECT id, owner_id, license_plate, vehicle_type, model, color, is_active, created_at, updated_at
		FROM vehicles
		WHERE id = $1
	`

	vehicle := &domain.Vehicle{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&vehicle.ID,
		&vehicle.OwnerID,
		&vehicle.LicensePlate,
		&vehicle.VehicleType,
		&vehicle.Model,
		&vehicle.Color,
		&vehicle.IsActive,
		&vehicle.CreatedAt,
		&vehicle.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVehicleNotFound
		}
		return nil, err
	}

	return vehicle, nil
}

func (r *vehicleRepository) GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.Vehicle, error) {
	query := `
		SELECT id, owner_id, license_plate, vehicle_type, model, color, is_active, created_at, updated_at
		FROM vehicles
		WHERE license_plate = $1
	`

	// Нормализуем номер перед поиском
	normalizedPlate := domain.NormalizeLicensePlate(licensePlate)

	vehicle := &domain.Vehicle{}
	err := r.db.QueryRow(ctx, query, normalizedPlate).Scan(
		&vehicle.ID,
		&vehicle.OwnerID,
		&vehicle.LicensePlate,
		&vehicle.VehicleType,
		&vehicle.Model,
		&vehicle.Color,
		&vehicle.IsActive,
		&vehicle.CreatedAt,
		&vehicle.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVehicleNotFound
		}
		return nil, err
	}

	return vehicle, nil
}

func (r *vehicleRepository) GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*domain.Vehicle, error) {
	query := `
		SELECT id, owner_id, license_plate, vehicle_type, model, color, is_active, created_at, updated_at
		FROM vehicles
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []*domain.Vehicle
	for rows.Next() {
		vehicle := &domain.Vehicle{}
		err := rows.Scan(
			&vehicle.ID,
			&vehicle.OwnerID,
			&vehicle.LicensePlate,
			&vehicle.VehicleType,
			&vehicle.Model,
			&vehicle.Color,
			&vehicle.IsActive,
			&vehicle.CreatedAt,
			&vehicle.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		vehicles = append(vehicles, vehicle)
	}

	return vehicles, nil
}

func (r *vehicleRepository) Update(ctx context.Context, vehicle *domain.Vehicle) error {
	query := `
		UPDATE vehicles
		SET owner_id = $2, license_plate = $3, vehicle_type = $4, model = $5, color = $6, is_active = $7, updated_at = $8
		WHERE id = $1
	`

	vehicle.UpdatedAt = time.Now()
	vehicle.LicensePlate = domain.NormalizeLicensePlate(vehicle.LicensePlate)

	result, err := r.db.Exec(ctx, query,
		vehicle.ID,
		vehicle.OwnerID,
		vehicle.LicensePlate,
		vehicle.VehicleType,
		vehicle.Model,
		vehicle.Color,
		vehicle.IsActive,
		vehicle.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVehicleNotFound
	}

	return nil
}

func (r *vehicleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Мягкое удаление - устанавливаем is_active = false
	query := `
		UPDATE vehicles
		SET is_active = false, updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVehicleNotFound
	}

	return nil
}

func (r *vehicleRepository) List(ctx context.Context, limit, offset int) ([]*domain.Vehicle, error) {
	query := `
		SELECT id, owner_id, license_plate, vehicle_type, model, color, is_active, created_at, updated_at
		FROM vehicles
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []*domain.Vehicle
	for rows.Next() {
		vehicle := &domain.Vehicle{}
		err := rows.Scan(
			&vehicle.ID,
			&vehicle.OwnerID,
			&vehicle.LicensePlate,
			&vehicle.VehicleType,
			&vehicle.Model,
			&vehicle.Color,
			&vehicle.IsActive,
			&vehicle.CreatedAt,
			&vehicle.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		vehicles = append(vehicles, vehicle)
	}

	return vehicles, nil
}
