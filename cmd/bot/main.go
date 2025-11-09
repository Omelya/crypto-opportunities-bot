package main

import (
	"crypto-opportunities-bot/internal/bot"
	"crypto-opportunities-bot/internal/config"
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/notification"
	"crypto-opportunities-bot/internal/repository"
	"crypto-opportunities-bot/internal/scraper"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.App.Environment == "development" {
		log.Printf("Config loaded:\n%s", cfg.SafeString())
	}

	// Ініціалізація БД
	db, err := repository.InitDatabase(cfg.Database, cfg.App)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Printf("✅ Database initialized")

	defer func() {
		if err := repository.CloseDatabase(db); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Міграція
	migrateErr := repository.AutoMigrate(db)
	if migrateErr != nil {
		log.Fatalf("Failed to migrate database: %v", migrateErr)
	}
	log.Printf("✅ Database migrated")

	userRepo := repository.NewUserRepository(db)
	prefsRepo := repository.NewUserPreferencesRepository(db)
	oppRepo := repository.NewOpportunityRepository(db)
	notifRepo := repository.NewNotificationRepository(db)

	botAPI, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		log.Fatalf("Failed to create bot API: %v", err)
	}
	botAPI.Debug = cfg.Telegram.Debug
	log.Printf("✅ Telegram Bot API initialized: @%s", botAPI.Self.UserName)

	notificationService := notification.NewService(
		botAPI,
		notifRepo,
		userRepo,
		prefsRepo,
		oppRepo,
	)
	log.Printf("✅ Notification service initialized")

	scraperService := scraper.NewScraperService(oppRepo)
	scraperService.RegisterScraper(scraper.NewBinanceScraper())
	scraperService.RegisterScraper(scraper.NewBybitScraper())

	scraperService.OnNewOpportunity(func(opp *models.Opportunity) {
		log.Printf("📢 Creating notifications for: %s", opp.Title)
		if err := notificationService.CreateOpportunityNotifications(opp); err != nil {
			log.Printf("❌ Failed to create notifications: %v", err)
		}
	})

	scraperScheduler := scraper.NewScheduler(scraperService)
	if err := scraperScheduler.Start(); err != nil {
		log.Fatalf("Failed to start scraper scheduler: %v", err)
	}
	log.Printf("✅ Scraper scheduler started")

	if cfg.App.Environment == "development" {
		log.Println("Running initial scraping...")
		if err := scraperScheduler.RunNow(); err != nil {
			log.Printf("Warning: Initial scraping failed: %v", err)
		}
	}

	notificationTicker := startNotificationDispatcher(notificationService)
	defer notificationTicker.Stop()

	// Daily Digest Scheduler
	digestScheduler := notification.NewDigestScheduler(notificationService)
	if err := digestScheduler.Start(); err != nil {
		log.Fatalf("Failed to start digest scheduler: %v", err)
	}
	log.Printf("✅ Daily digest scheduler started")
	defer digestScheduler.Stop()

	telegramBot, err := bot.NewBot(cfg, userRepo, prefsRepo, oppRepo)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	log.Printf("✅ Telegram bot initialized")

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
	log.Println("👋 Goodbye!")
}

func startNotificationDispatcher(service *notification.Service) *time.Ticker {
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for range ticker.C {
			if err := service.SendPendingNotifications(50); err != nil {
				log.Printf("Notification dispatcher error: %v", err)
			}

			if err := service.RetryFailedNotifications(20); err != nil {
				log.Printf("Retry failed notifications error: %v", err)
			}
		}
	}()

	log.Println("✅ Notification dispatcher started (every 10s)")
	return ticker
}
