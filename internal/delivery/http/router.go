package http

import (
	"net/http"

	"github.com/frontandrew/gate/internal/delivery/http/middleware"
	"github.com/frontandrew/gate/internal/domain"
	"github.com/frontandrew/gate/internal/pkg/config"
	"github.com/frontandrew/gate/internal/pkg/jwt"
	"github.com/frontandrew/gate/internal/pkg/logger"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// Router содержит все зависимости для HTTP роутера
type Router struct {
	accessHandler  *AccessHandler
	authHandler    *AuthHandler
	vehicleHandler *VehicleHandler
	passHandler    *PassHandler
	tokenService   *jwt.TokenService
	config         *config.Config
	logger         logger.Logger
}

// NewRouter создает новый HTTP router
func NewRouter(
	accessHandler *AccessHandler,
	authHandler *AuthHandler,
	vehicleHandler *VehicleHandler,
	passHandler *PassHandler,
	tokenService *jwt.TokenService,
	config *config.Config,
	logger logger.Logger,
) *Router {
	return &Router{
		accessHandler:  accessHandler,
		authHandler:    authHandler,
		vehicleHandler: vehicleHandler,
		passHandler:    passHandler,
		tokenService:   tokenService,
		config:         config,
		logger:         logger,
	}
}

// Setup настраивает все маршруты
func (rt *Router) Setup() http.Handler {
	r := chi.NewRouter()

	// Глобальные middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.RecoveryMiddleware(rt.logger))
	r.Use(middleware.LoggingMiddleware(rt.logger))
	r.Use(middleware.CORSMiddleware(middleware.CORSConfig{
		AllowedOrigins: rt.config.CORS.AllowedOrigins,
		AllowedMethods: rt.config.CORS.AllowedMethods,
		AllowedHeaders: rt.config.CORS.AllowedHeaders,
	}))

	// Health check endpoint (публичный)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{
			"status": "healthy",
		})
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes (без аутентификации)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", rt.authHandler.Register)
			r.Post("/login", rt.authHandler.Login)
		})

		// Access check endpoint (публичный - используется камерами/шлагбаумами)
		r.Post("/access/check", rt.accessHandler.CheckAccess)

		// Protected routes (требуют аутентификации)
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(rt.tokenService))

			// Current user endpoints
			r.Route("/auth/me", func(r chi.Router) {
				r.Get("/", rt.authHandler.GetMe)
			})

			// Vehicle endpoints
			r.Route("/vehicles", func(r chi.Router) {
				r.Get("/me", rt.vehicleHandler.GetMyVehicles)
				r.Post("/", rt.vehicleHandler.CreateVehicle)
				r.Get("/{id}", rt.vehicleHandler.GetVehicleByID)
			})

			// Pass endpoints
			r.Route("/passes", func(r chi.Router) {
				r.Get("/me", rt.passHandler.GetMyPasses)
				r.Get("/{id}", rt.passHandler.GetPassByID)

				// Admin/Guard only endpoints
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole(domain.RoleAdmin, domain.RoleGuard))
					r.Post("/", rt.passHandler.CreatePass)
					r.Delete("/{id}/revoke", rt.passHandler.RevokePass)
				})
			})

			// Access log endpoints
			r.Route("/access", func(r chi.Router) {
				r.Get("/me/logs", rt.accessHandler.GetMyAccessLogs)
				r.Get("/logs/vehicle/{id}", rt.accessHandler.GetVehicleAccessLogs)

				// Admin/Guard only endpoints
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole(domain.RoleAdmin, domain.RoleGuard))
					r.Get("/logs", rt.accessHandler.GetAccessLogs)
				})
			})
		})
	})

	return r
}
