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

	// Auto-migrate (if needed)
	// Note: Migrations are handled by the bot, but we can add admin-specific tables here
	// if err := repository.AutoMigrate(db); err != nil {
	// 	log.Fatalf("Failed to migrate database: %v", err)
	// }

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	oppRepo := repository.NewOpportunityRepository(db)
	arbRepo := repository.NewArbitrageRepository(db)
	defiRepo := repository.NewDeFiRepository(db)
	notifRepo := repository.NewNotificationRepository(db)

	// TODO: Initialize adminRepo when AdminRepository is created
	// adminRepo := repository.NewAdminRepository(db)

	// Create API server
	server := api.NewServer(
		cfg,
		userRepo,
		oppRepo,
		arbRepo,
		defiRepo,
		notifRepo,
		nil, // adminRepo - будемо додати пізніше
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
