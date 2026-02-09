package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	ML       MLConfig
	CORS     CORSConfig
	Logger   LoggerConfig
}

// ServerConfig содержит настройки HTTP сервера
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig содержит настройки подключения к PostgreSQL
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig содержит настройки подключения к Redis
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig содержит настройки JWT аутентификации
type JWTConfig struct {
	SecretKey     string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

// MLConfig содержит настройки ML сервиса
type MLConfig struct {
	ServiceURL    string
	MinConfidence float64
	Timeout       time.Duration
}

// CORSConfig содержит настройки CORS
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// LoggerConfig содержит настройки логирования
type LoggerConfig struct {
	Level  string
	Format string // json или console
	Output string // stdout или путь к файлу
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	// Загружаем .env файл (игнорируем ошибку, если файла нет)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "gate_user"),
			Password:        getEnv("DB_PASSWORD", "gate_password"),
			Database:        getEnv("DB_NAME", "gate_db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			SecretKey:     getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
			AccessExpiry:  getDurationEnv("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshExpiry: getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		},
		ML: MLConfig{
			ServiceURL:    getEnv("ML_SERVICE_URL", "http://localhost:8001"),
			MinConfidence: getFloatEnv("ML_MIN_CONFIDENCE", 0.7),
			Timeout:       getDurationEnv("ML_TIMEOUT", 30*time.Second),
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{
				getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000"),
			},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
	}

	return cfg, nil
}

// DSN возвращает строку подключения к PostgreSQL
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// Address возвращает адрес сервера
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// Address возвращает адрес Redis
func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// Вспомогательные функции для чтения переменных окружения

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
