package domain

import (
	"time"

	"github.com/google/uuid"
)

// WhitelistEntry - запись в белом списке
// Автомобили в белом списке ВСЕГДА получают доступ без проверки пропусков
type WhitelistEntry struct {
	ID           uuid.UUID  `json:"id"`
	LicensePlate string     `json:"license_plate"` // Номер автомобиля (нормализованный)
	Reason       string     `json:"reason"`        // Причина безусловного доступа
	AddedBy      uuid.UUID  `json:"added_by"`      // Кто добавил в список
	AddedAt      time.Time  `json:"added_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"` // NULL = бессрочно
	IsActive     bool       `json:"is_active"`
}

// IsExpired проверяет, истекла ли запись в белом списке
func (w *WhitelistEntry) IsExpired() bool {
	if w.ExpiresAt == nil {
		return false // Бессрочная привилегия никогда не истекает
	}
	return time.Now().After(*w.ExpiresAt)
}

// IsValid проверяет, действительна ли запись
func (w *WhitelistEntry) IsValid() bool {
	if !w.IsActive {
		return false
	}
	return !w.IsExpired()
}

// Validate проверяет корректность данных
func (w *WhitelistEntry) Validate() error {
	if w.LicensePlate == "" {
		return ErrInvalidLicensePlate
	}
	if w.Reason == "" {
		return ErrInvalidWhitelistData
	}
	if w.AddedBy == uuid.Nil {
		return ErrInvalidWhitelistData
	}

	// Нормализуем номер
	w.LicensePlate = NormalizeLicensePlate(w.LicensePlate)

	return nil
}
