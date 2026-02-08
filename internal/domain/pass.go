package domain

import (
	"time"

	"github.com/google/uuid"
)

// PassType представляет тип пропуска
type PassType string

const (
	PassTypePermanent PassType = "permanent" // Постоянный пропуск
	PassTypeTemporary PassType = "temporary" // Временный пропуск
)

// Pass - пропуск на территорию
// Пропуск выдается ПОЛЬЗОВАТЕЛЮ, а не автомобилю
// Один пользователь может иметь несколько активных пропусков
// Каждый пропуск может включать несколько автомобилей (через pass_vehicles)
type Pass struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`                 // Пользователь, которому выдан пропуск
	PassType     PassType   `json:"pass_type"`
	ValidFrom    time.Time  `json:"valid_from"`
	ValidUntil   *time.Time `json:"valid_until,omitempty"`   // NULL для постоянных пропусков
	IsActive     bool       `json:"is_active"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokedBy    *uuid.UUID `json:"revoked_by,omitempty"`
	RevokeReason string     `json:"revoke_reason,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Связанные данные (не хранятся в БД, заполняются при необходимости)
	User     *User      `json:"user,omitempty"`
	Vehicles []*Vehicle `json:"vehicles,omitempty"` // Автомобили, связанные с пропуском
}

// IsValid проверяет, действителен ли пропуск в данный момент времени
func (p *Pass) IsValid() bool {
	if !p.IsActive {
		return false
	}

	now := time.Now()

	// Проверяем, что пропуск уже вступил в силу
	if now.Before(p.ValidFrom) {
		return false
	}

	// Для временных пропусков проверяем дату истечения
	if p.PassType == PassTypeTemporary && p.ValidUntil != nil {
		if now.After(*p.ValidUntil) {
			return false
		}
	}

	return true
}

// IsExpired проверяет, истек ли временный пропуск
func (p *Pass) IsExpired() bool {
	if p.PassType != PassTypeTemporary || p.ValidUntil == nil {
		return false
	}
	return time.Now().After(*p.ValidUntil)
}

// Revoke отзывает пропуск
func (p *Pass) Revoke(revokedBy uuid.UUID, reason string) {
	now := time.Now()
	p.IsActive = false
	p.RevokedAt = &now
	p.RevokedBy = &revokedBy
	p.RevokeReason = reason
	p.UpdatedAt = now
}

// Validate проверяет корректность данных пропуска
func (p *Pass) Validate() error {
	if p.UserID == uuid.Nil {
		return ErrInvalidPassData
	}

	if p.PassType != PassTypePermanent && p.PassType != PassTypeTemporary {
		return ErrInvalidPassType
	}

	// Для временных пропусков ValidUntil обязателен
	if p.PassType == PassTypeTemporary {
		if p.ValidUntil == nil {
			return ErrInvalidPassData
		}
		if p.ValidUntil.Before(p.ValidFrom) {
			return ErrInvalidDateRange
		}
	}

	return nil
}
