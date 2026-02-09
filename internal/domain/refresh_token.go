package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken представляет refresh токен в системе
type RefreshToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"` // Не отдаем клиенту
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// IsValid проверяет, действителен ли refresh token
func (rt *RefreshToken) IsValid() bool {
	now := time.Now()

	// Токен не должен быть отозван
	if rt.RevokedAt != nil {
		return false
	}

	// Токен не должен быть истекшим
	if now.After(rt.ExpiresAt) {
		return false
	}

	return true
}

// Revoke отзывает refresh token
func (rt *RefreshToken) Revoke() {
	now := time.Now()
	rt.RevokedAt = &now
}
