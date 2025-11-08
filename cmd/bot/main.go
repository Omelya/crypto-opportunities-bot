package main

import (
	"crypto-opportunities-bot/internal/bot"
	"crypto-opportunities-bot/internal/repository"
	"crypto-opportunities-bot/internal/scraper"
	"log"
	"os"
	"os/signal"
	"syscall"

	"crypto-opportunities-bot/internal/config"
)

func main() {
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.App.Environment == "development" {
		log.Printf("Config loaded:\n%s", cfg.SafeString())
	}

	db, err := repository.InitDatabase(cfg.Database, cfg.App)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Printf("Database initialized")

	defer func() {
		if err := repository.CloseDatabase(db); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	migrateErr := repository.AutoMigrate(db)
	if migrateErr != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Printf("Database migrated")

	userRepo := repository.NewUserRepository(db)
	prefsRepo := repository.NewUserPreferencesRepository(db)
	oppRepo := repository.NewOpportunityRepository(db)

	scraperService := scraper.NewScraperService(oppRepo)
	scraperService.RegisterScraper(scraper.NewBinanceScraper())

	scraperScheduler := scraper.NewScheduler(scraperService)
	if err := scraperScheduler.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	if cfg.App.Environment == "development" {
		log.Println("Running initial scraping...")
		err := scraperScheduler.RunNow()
		if err != nil {
			log.Fatalf("Failed to run scraper: %v", err)
		}
	}

	telegramBot, err := bot.NewBot(cfg, userRepo, prefsRepo)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	log.Println("✅ Telegram bot initialized")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := telegramBot.Start(); err != nil {
			log.Fatalf("Bot error: %v", err)
		}
	}()

	log.Println("✅ Application started successfully!")
	log.Println("Press Ctrl+C to stop...")

	<-quit
	log.Println("\n🛑 Shutting down gracefully...")

	scraperScheduler.Stop()
}
