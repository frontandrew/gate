package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole представляет роль пользователя в системе
type UserRole string

const (
	RoleAdmin UserRole = "admin" // Администратор системы
	RoleUser  UserRole = "user"  // Обычный пользователь
	RoleGuard UserRole = "guard" // Охранник
)

// User - центральная сущность системы
// Пользователь владеет автомобилями и получает пропуска
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Никогда не возвращаем в JSON
	FullName     string     `json:"full_name"`
	Phone        string     `json:"phone,omitempty"`
	Role         UserRole   `json:"role"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// IsAdmin проверяет, является ли пользователь администратором
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsGuard проверяет, является ли пользователь охранником
func (u *User) IsGuard() bool {
	return u.Role == RoleGuard
}

// CanManagePasses проверяет, может ли пользователь управлять пропусками
func (u *User) CanManagePasses() bool {
	return u.Role == RoleAdmin || u.Role == RoleGuard
}

// Validate проверяет корректность данных пользователя
func (u *User) Validate() error {
	if u.Email == "" {
		return ErrInvalidEmail
	}
	if u.FullName == "" {
		return ErrInvalidUserData
	}
	if u.Role != RoleAdmin && u.Role != RoleUser && u.Role != RoleGuard {
		return ErrInvalidRole
	}
	return nil
}
