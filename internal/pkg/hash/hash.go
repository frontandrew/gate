package hash

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost - стоимость хеширования по умолчанию (12)
	DefaultCost = 12
)

// HashPassword хеширует пароль с использованием bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword сравнивает хешированный пароль с plain-text паролем
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
