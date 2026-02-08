package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// VehicleType представляет тип транспортного средства
type VehicleType string

const (
	VehicleTypeCar        VehicleType = "car"
	VehicleTypeTruck      VehicleType = "truck"
	VehicleTypeMotorcycle VehicleType = "motorcycle"
	VehicleTypeBus        VehicleType = "bus"
	VehicleTypeOther      VehicleType = "other"
)

// Vehicle - автомобиль пользователя (способ аутентификации)
// ВАЖНО: Автомобиль ОБЯЗАТЕЛЬНО привязан к владельцу (OwnerID NOT NULL)
type Vehicle struct {
	ID           uuid.UUID   `json:"id"`
	OwnerID      uuid.UUID   `json:"owner_id"`           // ОБЯЗАТЕЛЬНАЯ связь с User
	LicensePlate string      `json:"license_plate"`      // Номер автомобиля (уникальный)
	VehicleType  VehicleType `json:"vehicle_type"`
	Model        string      `json:"model,omitempty"`
	Color        string      `json:"color,omitempty"`
	IsActive     bool        `json:"is_active"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`

	// Связанные данные (не хранятся в БД, заполняются при необходимости)
	Owner *User `json:"owner,omitempty"`
}

// NormalizeLicensePlate нормализует номер автомобиля (убирает пробелы, приводит к верхнему регистру)
func NormalizeLicensePlate(plate string) string {
	// Убираем пробелы и приводим к верхнему регистру
	normalized := strings.ToUpper(strings.ReplaceAll(plate, " ", ""))
	return normalized
}

// Validate проверяет корректность данных автомобиля
func (v *Vehicle) Validate() error {
	if v.OwnerID == uuid.Nil {
		return ErrInvalidVehicleData
	}
	if v.LicensePlate == "" {
		return ErrInvalidLicensePlate
	}
	// Нормализуем номер
	v.LicensePlate = NormalizeLicensePlate(v.LicensePlate)

	if len(v.LicensePlate) < 5 || len(v.LicensePlate) > 20 {
		return ErrInvalidLicensePlate
	}
	return nil
}
