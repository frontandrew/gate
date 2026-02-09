package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	deliveryHTTP "github.com/frontandrew/gate/internal/delivery/http"
	"github.com/frontandrew/gate/internal/infrastructure/ml"
	"github.com/frontandrew/gate/internal/pkg/config"
	"github.com/frontandrew/gate/internal/pkg/database"
	"github.com/frontandrew/gate/internal/pkg/jwt"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/frontandrew/gate/internal/repository/postgres"
	"github.com/frontandrew/gate/internal/usecase/access"
	"github.com/frontandrew/gate/internal/usecase/auth"
	"github.com/frontandrew/gate/internal/usecase/pass"
	"github.com/frontandrew/gate/internal/usecase/vehicle"
)

func main() {
	// =========================================================================
	// Загрузка конфигурации
	// =========================================================================

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// =========================================================================
	// Инициализация logger
	// =========================================================================

	log := logger.New(cfg.Logger.Level, cfg.Logger.Format, cfg.Logger.Output)
	log.Info("Starting GATE API server", map[string]interface{}{
		"version": "1.0.0",
		"env":     "development",
	})

	// =========================================================================
	// Подключение к PostgreSQL
	// =========================================================================

	ctx := context.Background()
	db, err := database.Connect(ctx, &cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer database.Close(db)

	log.Info("Connected to PostgreSQL", map[string]interface{}{
		"host":     cfg.Database.Host,
		"port":     cfg.Database.Port,
		"database": cfg.Database.Database,
	})

	// =========================================================================
	// Создание repositories
	// =========================================================================

	userRepo := postgres.NewUserRepository(db)
	vehicleRepo := postgres.NewVehicleRepository(db)
	passRepo := postgres.NewPassRepository(db)
	passVehicleRepo := postgres.NewPassVehicleRepository(db)
	accessLogRepo := postgres.NewAccessLogRepository(db)
	whitelistRepo := postgres.NewWhitelistRepository(db)
	blacklistRepo := postgres.NewBlacklistRepository(db)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(db)

	log.Info("Repositories initialized")

	// =========================================================================
	// Создание ML клиента
	// =========================================================================

	mlClient := ml.NewHTTPClient(cfg.ML.ServiceURL, cfg.ML.Timeout)

	// Проверяем доступность ML сервиса
	if err := mlClient.Health(ctx); err != nil {
		log.Warn("ML service is not available", map[string]interface{}{
			"error": err.Error(),
			"url":   cfg.ML.ServiceURL,
		})
		log.Warn("Access checks will fail until ML service is running")
	} else {
		log.Info("ML service is healthy", map[string]interface{}{
			"url": cfg.ML.ServiceURL,
		})
	}

	// =========================================================================
	// Создание JWT token service
	// =========================================================================

	tokenService := jwt.NewTokenService(
		cfg.JWT.SecretKey,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	log.Info("JWT token service initialized")

	// =========================================================================
	// Создание use case services
	// =========================================================================

	authService := auth.NewService(userRepo, refreshTokenRepo, tokenService, log)
	vehicleService := vehicle.NewService(vehicleRepo, userRepo, log)
	passService := pass.NewService(passRepo, passVehicleRepo, userRepo, vehicleRepo, log)
	accessService := access.NewService(vehicleRepo, userRepo, passRepo, accessLogRepo, whitelistRepo, blacklistRepo, mlClient, log, cfg.ML.MinConfidence)

	log.Info("Use case services initialized")

	// =========================================================================
	// Создание HTTP handlers
	// =========================================================================

	authHandler := deliveryHTTP.NewAuthHandler(authService, log)
	vehicleHandler := deliveryHTTP.NewVehicleHandler(vehicleService, log)
	passHandler := deliveryHTTP.NewPassHandler(passService, log)
	accessHandler := deliveryHTTP.NewAccessHandler(accessService, log)

	log.Info("HTTP handlers initialized")

	// =========================================================================
	// Создание и настройка HTTP router
	// =========================================================================

	router := deliveryHTTP.NewRouter(
		accessHandler,
		authHandler,
		vehicleHandler,
		passHandler,
		tokenService,
		cfg,
		log,
	)

	handler := router.Setup()

	log.Info("HTTP router configured")

	// =========================================================================
	// Создание HTTP сервера
	// =========================================================================

	srv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// =========================================================================
	// Запуск сервера в goroutine
	// =========================================================================

	serverErrors := make(chan error, 1)

	go func() {
		log.Info("API server listening", map[string]interface{}{
			"address": srv.Addr,
		})
		serverErrors <- srv.ListenAndServe()
	}()

	// =========================================================================
	// Graceful shutdown
	// =========================================================================

	// Канал для получения сигналов операционной системы
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Блокируемся до получения сигнала или ошибки сервера
	select {
	case err := <-serverErrors:
		log.Fatal("Server error", map[string]interface{}{
			"error": err.Error(),
		})

	case sig := <-shutdown:
		log.Info("Shutdown signal received", map[string]interface{}{
			"signal": sig.String(),
		})

		// Даем серверу 30 секунд на graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("Graceful shutdown failed", map[string]interface{}{
				"error": err.Error(),
			})

			// Принудительное закрытие
			if err := srv.Close(); err != nil {
				log.Fatal("Failed to close server", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}

		log.Info("Server stopped gracefully")
	}
}
