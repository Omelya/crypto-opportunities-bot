package api

import (
	"context"
	"crypto-opportunities-bot/internal/api/auth"
	"crypto-opportunities-bot/internal/api/handlers"
	"crypto-opportunities-bot/internal/api/middleware"
	"crypto-opportunities-bot/internal/config"
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Server представляє HTTP сервер для Admin API
type Server struct {
	config     *config.Config
	httpServer *http.Server
	router     *mux.Router

	// Repositories
	userRepo   repository.UserRepository
	oppRepo    repository.OpportunityRepository
	arbRepo    repository.ArbitrageRepository
	defiRepo   repository.DeFiRepository
	notifRepo  repository.NotificationRepository
	adminRepo  repository.AdminRepository

	// Auth & Middleware
	jwtManager  *auth.JWTManager
	rateLimiter *middleware.RateLimiter

	// Handlers
	healthHandler *handlers.HealthHandler
	authHandler   *handlers.AuthHandler
	userHandler   *handlers.UserHandler
	statsHandler  *handlers.StatsHandler
}

// NewServer створює новий Admin API server
func NewServer(
	cfg *config.Config,
	userRepo repository.UserRepository,
	oppRepo repository.OpportunityRepository,
	arbRepo repository.ArbitrageRepository,
	defiRepo repository.DeFiRepository,
	notifRepo repository.NotificationRepository,
	adminRepo repository.AdminRepository,
) *Server {
	s := &Server{
		config:    cfg,
		userRepo:  userRepo,
		oppRepo:   oppRepo,
		arbRepo:   arbRepo,
		defiRepo:  defiRepo,
		notifRepo: notifRepo,
		adminRepo: adminRepo,
	}

	// Initialize JWT Manager
	s.jwtManager = auth.NewJWTManager(cfg.Admin.JWTSecret, 24*time.Hour)

	// Initialize Rate Limiter
	s.rateLimiter = middleware.NewRateLimiter(cfg.Admin.RateLimit)

	// Initialize handlers
	s.healthHandler = handlers.NewHealthHandler()
	s.authHandler = handlers.NewAuthHandler(adminRepo, s.jwtManager)
	s.userHandler = handlers.NewUserHandler(userRepo)
	s.statsHandler = handlers.NewStatsHandler(userRepo, oppRepo, arbRepo, defiRepo, notifRepo)

	// Setup router
	s.setupRouter()

	return s
}

// setupRouter налаштовує всі роути та middleware
func (s *Server) setupRouter() {
	r := mux.NewRouter()

	// Global middleware
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.CORSMiddleware(s.config.Admin.AllowedOrigins))
	r.Use(s.rateLimiter.RateLimitMiddleware) // Rate limiting

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// ========== Public routes (no auth required) ==========

	// Health check
	api.HandleFunc("/health", s.healthHandler.Health).Methods("GET")
	api.HandleFunc("/ping", s.healthHandler.Ping).Methods("GET")

	// Authentication
	api.HandleFunc("/auth/login", s.authHandler.Login).Methods("POST")
	api.HandleFunc("/auth/refresh", s.authHandler.RefreshToken).Methods("POST")

	// ========== Protected routes (require JWT) ==========

	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.JWTAuthMiddleware(s.jwtManager))

	// Auth endpoints (require auth)
	protected.HandleFunc("/auth/logout", s.authHandler.Logout).Methods("POST")
	protected.HandleFunc("/auth/me", s.authHandler.Me).Methods("GET")

	// User management (viewer+)
	protected.HandleFunc("/users", s.userHandler.ListUsers).Methods("GET")
	protected.HandleFunc("/users/{id}", s.userHandler.GetUser).Methods("GET")
	protected.HandleFunc("/users/{id}/stats", s.userHandler.GetUserStats).Methods("GET")

	// User modifications (admin+)
	adminRoutes := protected.PathPrefix("").Subrouter()
	adminRoutes.Use(middleware.RequireRole(models.AdminRoleAdmin))
	adminRoutes.HandleFunc("/users/{id}", s.userHandler.UpdateUser).Methods("PUT")

	// Statistics (viewer+)
	protected.HandleFunc("/stats/dashboard", s.statsHandler.Dashboard).Methods("GET")
	protected.HandleFunc("/stats/users", s.statsHandler.UserStats).Methods("GET")

	s.router = r
}

// Start запускає HTTP сервер
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Admin.Host, s.config.Admin.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🚀 Admin API server starting on %s", addr)
	log.Printf("📝 Swagger UI: http://%s/swagger", addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop зупиняє HTTP сервер gracefully
func (s *Server) Stop(ctx context.Context) error {
	log.Println("🛑 Shutting down Admin API server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("✅ Admin API server stopped")
	return nil
}

// Router повертає router для тестування
func (s *Server) Router() *mux.Router {
	return s.router
}
