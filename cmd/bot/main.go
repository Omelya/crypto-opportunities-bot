package main

import (
	"context"
	"crypto-opportunities-bot/internal/arbitrage"
	"crypto-opportunities-bot/internal/arbitrage/websocket"
	"crypto-opportunities-bot/internal/bot"
	"crypto-opportunities-bot/internal/config"
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/notification"
	"crypto-opportunities-bot/internal/payment"
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
	actionRepo := repository.NewUserActionRepository(db)
	subsRepo := repository.NewSubscriptionRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	arbRepo := repository.NewArbitrageRepository(db)
	defiRepo := repository.NewDeFiRepository(db)

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
		arbRepo,
		defiRepo,
	)
	log.Printf("✅ Notification service initialized")

	// Payment service (Monobank)
	var paymentService *payment.Service
	var webhookHandler *payment.WebhookHandler

	if cfg.Payment.MonobankToken != "" {
		paymentCfg := &payment.Config{
			MonobankToken: cfg.Payment.MonobankToken,
			WebhookURL:    cfg.Payment.WebhookURL,
			RedirectURL:   cfg.Payment.RedirectURL,
		}

		paymentService = payment.NewService(paymentCfg, subsRepo, paymentRepo, userRepo)
		log.Printf("✅ Payment service initialized (Monobank)")

		// Webhook handler
		webhookHandler = payment.NewWebhookHandler(paymentService, cfg.Payment.MonobankPublicKey)

		// Start webhook server у окремій go routine
		if cfg.Payment.WebhookPort != "" {
			go func() {
				log.Printf("🌐 Starting webhook server on port %s...", cfg.Payment.WebhookPort)
				if err := payment.StartWebhookServer(webhookHandler, cfg.Payment.WebhookPort); err != nil {
					log.Printf("⚠️ Webhook server error: %v", err)
				}
			}()
		}

		// Subscription expiration checker
		subscriptionTicker := startSubscriptionChecker(paymentService)
		defer subscriptionTicker.Stop()
	} else {
		log.Printf("⚠️ Monobank not configured - payment features disabled")
	}

	scraperService := scraper.NewScraperService(oppRepo)
	scraperService.RegisterScraper(scraper.NewBinanceScraper())
	scraperService.RegisterScraper(scraper.NewBybitScraper())

	scraperService.OnNewOpportunity(func(opp *models.Opportunity) {
		log.Printf("📢 Creating notifications for: %s", opp.Title)
		if err := notificationService.CreateOpportunityNotifications(opp); err != nil {
			log.Printf("❌ Failed to create notifications: %v", err)
		}
	})

	// DeFi Scraper (Premium feature)
	if cfg.DeFi.Enabled {
		defiScraperConfig := scraper.DeFiScraperConfig{
			Chains:       cfg.DeFi.Chains,
			Protocols:    cfg.DeFi.Protocols,
			MinAPY:       cfg.DeFi.MinAPY,
			MinTVL:       cfg.DeFi.MinTVL,
			MaxIL:        cfg.DeFi.MaxILRisk,
			MinVolume24h: cfg.DeFi.MinVolume24h,
		}

		defiScraper := scraper.NewDeFiScraper(defiRepo, defiScraperConfig)

		// Wire DeFi callbacks to notification system
		defiScraper.OnNewDeFi(func(defi *models.DeFiOpportunity) {
			log.Printf("🌾 New DeFi opportunity: %s on %s (APY: %.2f%%)", defi.PoolName, defi.Chain, defi.APY)

			// Create notifications for premium users
			if err := notificationService.CreateDeFiNotifications(defi); err != nil {
				log.Printf("❌ Failed to create DeFi notifications: %v", err)
			}
		})

		log.Printf("✅ DeFi scraper initialized")
		log.Printf("   Chains: %v", cfg.DeFi.Chains)
		log.Printf("   Min APY: %.2f%%", cfg.DeFi.MinAPY)
		log.Printf("   Min TVL: $%.0f", cfg.DeFi.MinTVL)
		log.Printf("   Max IL Risk: %.2f%%", cfg.DeFi.MaxILRisk)

		// TODO: Add DeFi scraper scheduler (separate from regular scrapers due to longer interval)
	} else {
		log.Printf("⚠️ DeFi monitoring disabled in config")
	}

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

	// Arbitrage System (Premium feature)
	var arbitrageDetector *arbitrage.Detector
	var premiumWatcher *time.Ticker
	if cfg.Arbitrage.Enabled {
		arbitrageDetector = startArbitrageMonitoring(cfg, arbRepo, userRepo, notificationService)

		// If arbitrage didn't start (no premium users), start watcher
		if arbitrageDetector == nil {
			premiumWatcher = startPremiumWatcher(cfg, arbRepo, userRepo, notificationService, &arbitrageDetector)
		}
	} else {
		log.Printf("⚠️ Arbitrage monitoring disabled in config")
	}

	if arbitrageDetector != nil {
		defer arbitrageDetector.Stop()
	}
	if premiumWatcher != nil {
		defer premiumWatcher.Stop()
	}

	telegramBot, err := bot.NewBot(cfg, userRepo, prefsRepo, oppRepo, actionRepo, subsRepo, arbRepo, defiRepo, paymentService)
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

func startSubscriptionChecker(service *payment.Service) *time.Ticker {
	ticker := time.NewTicker(1 * time.Hour)

	// Перевірка при запуску
	go func() {
		if err := service.CheckExpiredSubscriptions(); err != nil {
			log.Printf("Subscription checker error: %v", err)
		}
	}()

	go func() {
		for range ticker.C {
			if err := service.CheckExpiredSubscriptions(); err != nil {
				log.Printf("Subscription checker error: %v", err)
			}
		}
	}()

	log.Println("✅ Subscription checker started (every 1h)")
	return ticker
}

func startArbitrageMonitoring(
	cfg *config.Config,
	arbRepo repository.ArbitrageRepository,
	userRepo repository.UserRepository,
	notificationService *notification.Service,
) *arbitrage.Detector {
	// Перевірити чи є Premium користувачі
	premiumCount, err := userRepo.CountPremium()
	if err != nil {
		log.Printf("⚠️ Error checking premium users: %v", err)
		return nil
	}

	if premiumCount == 0 {
		log.Printf("⚠️ No premium users - arbitrage monitoring paused")
		// TODO: Періодично перевіряти чи з'явились Premium користувачі
		return nil
	}

	log.Printf("📊 Found %d premium users - starting arbitrage monitoring", premiumCount)

	// Create OrderBook Manager
	obManager := arbitrage.NewOrderBookManager()

	// Initialize WebSocket managers for each exchange
	ctx := context.Background()

	for _, exchange := range cfg.Arbitrage.Exchanges {
		var wsManager websocket.Manager

		switch exchange {
		case "binance":
			wsManager = websocket.NewBinanceManager()
		case "bybit":
			wsManager = websocket.NewBybitManager()
		case "okx":
			wsManager = websocket.NewOKXManager()
		default:
			log.Printf("⚠️ Unknown exchange: %s", exchange)
			continue
		}

		// Connect
		if err := wsManager.Connect(ctx); err != nil {
			log.Printf("❌ Failed to connect to %s: %v", exchange, err)
			continue
		}

		// Subscribe to all pairs at once
		if err := wsManager.Subscribe(cfg.Arbitrage.Pairs); err != nil {
			log.Printf("⚠️ Failed to subscribe to pairs on %s: %v", exchange, err)
			continue
		}

		// Register with OrderBook Manager
		obManager.RegisterExchange(exchange, wsManager)
		log.Printf("✅ Connected to %s WebSocket (%d pairs)", exchange, len(cfg.Arbitrage.Pairs))
	}

	// Create Calculator
	calculator := arbitrage.NewCalculator()

	// Create Deduplicator
	deduplicator := arbitrage.NewDeduplicator(time.Duration(cfg.Arbitrage.DeduplicateTTL) * time.Minute)

	// Create Detector
	detector := arbitrage.NewDetector(
		obManager,
		calculator,
		arbRepo,
		userRepo,
		deduplicator,
		&cfg.Arbitrage,
	)

	// Wire detector callbacks to notification system
	detector.OnArbitrageDetected(func(arb *models.ArbitrageOpportunity) {
		log.Printf("🔥 Arbitrage detected: %s (%.2f%% profit)", arb.Pair, arb.NetProfitPercent)

		// Create notifications for premium users
		if err := notificationService.CreateArbitrageNotifications(arb); err != nil {
			log.Printf("❌ Failed to create arbitrage notifications: %v", err)
		}
	})

	// Start detector
	detector.Start()

	log.Printf("✅ Arbitrage monitoring started")
	log.Printf("   Pairs: %v", cfg.Arbitrage.Pairs)
	log.Printf("   Exchanges: %v", cfg.Arbitrage.Exchanges)
	log.Printf("   Min Profit: %.2f%%", cfg.Arbitrage.MinProfitPercent)

	return detector
}

func startPremiumWatcher(
	cfg *config.Config,
	arbRepo repository.ArbitrageRepository,
	userRepo repository.UserRepository,
	notificationService *notification.Service,
	detectorPtr **arbitrage.Detector,
) *time.Ticker {
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for range ticker.C {
			// Check if detector already started
			if *detectorPtr != nil {
				log.Printf("✅ Arbitrage monitoring already running, stopping watcher")
				ticker.Stop()
				return
			}

			// Check for premium users
			premiumCount, err := userRepo.CountPremium()
			if err != nil {
				log.Printf("⚠️ Premium watcher error: %v", err)
				continue
			}

			if premiumCount > 0 {
				log.Printf("🎉 Premium user detected! Starting arbitrage monitoring...")

				// Start arbitrage monitoring
				detector := startArbitrageMonitoring(cfg, arbRepo, userRepo, notificationService)
				if detector != nil {
					*detectorPtr = detector
					log.Printf("✅ Arbitrage monitoring started successfully")
					ticker.Stop()
					return
				}
			}
		}
	}()

	log.Println("👀 Premium watcher started (every 5 min)")
	return ticker
}
