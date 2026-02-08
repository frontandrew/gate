package domain

import (
	"time"

	"github.com/google/uuid"
)

// PassVehicle - связь между пропуском и автомобилем (many-to-many)
// Один пропуск может включать несколько автомобилей
// Один автомобиль может быть привязан к нескольким пропускам
type PassVehicle struct {
	ID        uuid.UUID  `json:"id"`
	PassID    uuid.UUID  `json:"pass_id"`
	VehicleID uuid.UUID  `json:"vehicle_id"`
	AddedAt   time.Time  `json:"added_at"`
	AddedBy   *uuid.UUID `json:"added_by,omitempty"` // Кто добавил автомобиль к пропуску

	// Связанные данные (не хранятся в БД, заполняются при необходимости)
	Pass    *Pass    `json:"pass,omitempty"`
	Vehicle *Vehicle `json:"vehicle,omitempty"`
}

// Validate проверяет корректность данных связи
func (pv *PassVehicle) Validate() error {
	if pv.PassID == uuid.Nil {
		return ErrInvalidPassVehicleData
	}
	if pv.VehicleID == uuid.Nil {
		return ErrInvalidPassVehicleData
	}
	return nil
}
