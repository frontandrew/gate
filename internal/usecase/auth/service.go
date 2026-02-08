package auth

import (
	"context"
	"fmt"

	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/hash"
	"github.com/frontandrew/gate/internal/pkg/jwt"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/repository"
	"github.com/google/uuid"
)

// RegisterRequest - запрос на регистрацию
type RegisterRequest struct {
	Email    string          `json:"email" validate:"required,email"`
	Password string          `json:"password" validate:"required,min=8"`
	FullName string          `json:"full_name" validate:"required"`
	Phone    string          `json:"phone,omitempty"`
	Role     domain.UserRole `json:"role,omitempty"`
}

// LoginRequest - запрос на вход
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse - ответ на вход
type LoginResponse struct {
	User         *domain.User      `json:"user"`
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	ExpiresAt    string            `json:"expires_at"`
}

// Service содержит бизнес-логику аутентификации
type Service struct {
	userRepo     repository.UserRepository
	tokenService *jwt.TokenService
	logger       logger.Logger
}

// NewService создает новый экземпляр AuthService
func NewService(
	userRepo repository.UserRepository,
	tokenService *jwt.TokenService,
	logger logger.Logger,
) *Service {
	return &Service{
		userRepo:     userRepo,
		tokenService: tokenService,
		logger:       logger,
	}
}

// Register регистрирует нового пользователя
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*domain.User, error) {
	s.logger.Info("Registering new user", map[string]interface{}{
		"email": req.Email,
	})

	// Проверяем, что пользователь с таким email еще не существует
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil {
		s.logger.Warn("User already exists", map[string]interface{}{
			"email": req.Email,
		})
		return nil, domain.ErrUserAlreadyExists
	}

	// Хешируем пароль
	passwordHash, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создаем пользователя
	user := &domain.User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		FullName:     req.FullName,
		Phone:        req.Phone,
		Role:         req.Role,
		IsActive:     true,
	}

	// Если роль не указана, устанавливаем по умолчанию "user"
	if user.Role == "" {
		user.Role = domain.RoleUser
	}

	// Валидируем данные пользователя
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Сохраняем в БД
	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("Failed to create user", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User registered successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	// Не возвращаем password_hash
	user.PasswordHash = ""

	return user, nil
}

// Login аутентифицирует пользователя и возвращает JWT токены
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	s.logger.Info("User login attempt", map[string]interface{}{
		"email": req.Email,
	})

	// Находим пользователя по email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			s.logger.Warn("Login failed: user not found", map[string]interface{}{
				"email": req.Email,
			})
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Проверяем, активен ли пользователь
	if !user.IsActive {
		s.logger.Warn("Login failed: user inactive", map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, domain.ErrUserInactive
	}

	// Проверяем пароль
	if !hash.CheckPassword(user.PasswordHash, req.Password) {
		s.logger.Warn("Login failed: invalid password", map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, domain.ErrInvalidCredentials
	}

	// Генерируем JWT токены
	tokenPair, err := s.tokenService.GenerateTokenPair(user)
	if err != nil {
		s.logger.Error("Failed to generate tokens", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Обновляем last_login_at
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Error("Failed to update last login", map[string]interface{}{
			"error": err.Error(),
		})
	}

	s.logger.Info("User logged in successfully", map[string]interface{}{
		"user_id": user.ID,
	})

	// Не возвращаем password_hash
	user.PasswordHash = ""

	return &LoginResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// GetUserByID возвращает пользователя по ID
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Не возвращаем password_hash
	user.PasswordHash = ""

	return user, nil
}

// ValidateToken валидирует JWT токен и возвращает claims
func (s *Service) ValidateToken(tokenString string) (*jwt.Claims, error) {
	return s.tokenService.ValidateToken(tokenString)
}
