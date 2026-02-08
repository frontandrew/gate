package domain

import (
	"time"

	"github.com/google/uuid"
)

// BlacklistEntry - запись в черном списке
// Автомобили в черном списке БЛОКИРУЮТСЯ независимо от наличия пропусков
type BlacklistEntry struct {
	ID           uuid.UUID  `json:"id"`
	LicensePlate string     `json:"license_plate"`          // Номер автомобиля (нормализованный)
	Reason       string     `json:"reason"`                 // Причина блокировки
	AddedBy      uuid.UUID  `json:"added_by"`               // Кто добавил в список
	AddedAt      time.Time  `json:"added_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`   // NULL = бессрочно
	IsActive     bool       `json:"is_active"`
}

// IsExpired проверяет, истекла ли запись в черном списке
func (b *BlacklistEntry) IsExpired() bool {
	if b.ExpiresAt == nil {
		return false // Бессрочная блокировка никогда не истекает
	}
	return time.Now().After(*b.ExpiresAt)
}

// IsValid проверяет, действительна ли запись
func (b *BlacklistEntry) IsValid() bool {
	if !b.IsActive {
		return false
	}
	return !b.IsExpired()
}

// Validate проверяет корректность данных
func (b *BlacklistEntry) Validate() error {
	if b.LicensePlate == "" {
		return ErrInvalidLicensePlate
	}
	if b.Reason == "" {
		return ErrInvalidBlacklistData
	}
	if b.AddedBy == uuid.Nil {
		return ErrInvalidBlacklistData
	}

	// Нормализуем номер
	b.LicensePlate = NormalizeLicensePlate(b.LicensePlate)

	return nil
}
