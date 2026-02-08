package domain

import (
	"time"

	"github.com/google/uuid"
)

// Direction представляет направление проезда
type Direction string

const (
	DirectionIn  Direction = "IN"  // Въезд на территорию
	DirectionOut Direction = "OUT" // Выезд с территории
)

// AccessLog - запись о проезде
// ВАЖНО: Главная информация - КТО (User) получил доступ, ЧЕРЕЗ ЧТО (Vehicle) - вспомогательная
type AccessLog struct {
	ID                     uuid.UUID  `json:"id"`
	UserID                 *uuid.UUID `json:"user_id,omitempty"`     // КТО - главная информация
	VehicleID              *uuid.UUID `json:"vehicle_id,omitempty"`  // ЧЕРЕЗ ЧТО - вспомогательная информация
	LicensePlate           string     `json:"license_plate"`         // Распознанный номер (может не совпадать с БД)
	ImageURL               string     `json:"image_url,omitempty"`   // URL изображения с камеры
	RecognitionConfidence  float64    `json:"recognition_confidence"` // Уверенность распознавания (0-100)
	AccessGranted          bool       `json:"access_granted"`        // Разрешен ли доступ
	AccessReason           string     `json:"access_reason"`         // Причина решения
	GateID                 string     `json:"gate_id,omitempty"`     // ID ворот
	Direction              Direction  `json:"direction"`
	Timestamp              time.Time  `json:"timestamp"`

	// Связанные данные (не хранятся в БД, заполняются при необходимости)
	User    *User    `json:"user,omitempty"`
	Vehicle *Vehicle `json:"vehicle,omitempty"`
}

// Validate проверяет корректность данных лога
func (al *AccessLog) Validate() error {
	if al.LicensePlate == "" {
		return ErrInvalidAccessLogData
	}

	if al.Direction != DirectionIn && al.Direction != DirectionOut {
		return ErrInvalidDirection
	}

	if al.RecognitionConfidence < 0 || al.RecognitionConfidence > 100 {
		return ErrInvalidConfidence
	}

	return nil
}

// IsHighConfidence проверяет, достаточна ли уверенность распознавания
func (al *AccessLog) IsHighConfidence(minConfidence float64) bool {
	return al.RecognitionConfidence >= minConfidence
}
