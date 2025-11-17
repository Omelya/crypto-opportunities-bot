package main

import (
	"context"
	"crypto-opportunities-bot/internal/api"
	"crypto-opportunities-bot/internal/config"
	"crypto-opportunities-bot/internal/repository"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("🚀 Starting Crypto Opportunities Admin API...")
	log.Printf("Environment: %s", cfg.App.Environment)

	// Database connection
	dsn := repository.BuildDSN(&cfg.Database)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("✅ Database connected")

	// Auto-migrate admin tables
	if err := repository.MigrateAdminTables(db); err != nil {
		log.Fatalf("Failed to migrate admin tables: %v", err)
	}
	log.Println("✅ Admin tables migrated")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	oppRepo := repository.NewOpportunityRepository(db)
	arbRepo := repository.NewArbitrageRepository(db)
	defiRepo := repository.NewDeFiRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	actionRepo := repository.NewUserActionRepository(db)

	// Create default admin if environment variables are set
	if username := os.Getenv("ADMIN_DEFAULT_USERNAME"); username != "" {
		password := os.Getenv("ADMIN_DEFAULT_PASSWORD")
		email := os.Getenv("ADMIN_DEFAULT_EMAIL")

		if password != "" && email != "" {
			if err := repository.CreateDefaultAdmin(db, username, password, email); err != nil {
				log.Printf("⚠️ Could not create default admin: %v", err)
			} else {
				log.Printf("✅ Default admin created: %s", username)
			}
		}
	}

	// Create API server
	server := api.NewServer(
		cfg,
		userRepo,
		oppRepo,
		arbRepo,
		defiRepo,
		notifRepo,
		adminRepo,
		actionRepo,
	)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("✅ Admin API server started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down Admin API...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Admin API stopped gracefully")
}
