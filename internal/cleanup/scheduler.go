package cleanup

import (
	"crypto-opportunities-bot/internal/repository"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler відповідає за автоматичне очищення старих даних
type Scheduler struct {
	cron         *cron.Cron
	oppRepo      repository.OpportunityRepository
	arbRepo      repository.ArbitrageRepository
	defiRepo     repository.DeFiRepository
	notifRepo    repository.NotificationRepository
	config       *Config
}

// Config налаштування для cleanup операцій
type Config struct {
	// OpportunitiesRetentionDays - скільки днів зберігати звичайні opportunities
	OpportunitiesRetentionDays int

	// ArbitrageRetentionDays - скільки днів зберігати arbitrage opportunities
	ArbitrageRetentionDays int

	// DeFiRetentionDays - скільки днів зберігати DeFi opportunities
	DeFiRetentionDays int

	// SentNotificationsRetentionDays - скільки днів зберігати відправлені notifications
	SentNotificationsRetentionDays int

	// FailedNotificationsRetentionDays - скільки днів зберігати failed notifications
	FailedNotificationsRetentionDays int

	// Schedule - cron schedule для cleanup (default: "0 2 * * *" - щодня о 2:00)
	Schedule string
}

// DefaultConfig повертає конфігурацію за замовчуванням
func DefaultConfig() *Config {
	return &Config{
		OpportunitiesRetentionDays:       30,  // 30 днів для звичайних opportunities
		ArbitrageRetentionDays:           7,   // 7 днів для arbitrage
		DeFiRetentionDays:                7,   // 7 днів для DeFi
		SentNotificationsRetentionDays:   90,  // 90 днів для відправлених
		FailedNotificationsRetentionDays: 30,  // 30 днів для failed
		Schedule:                         "0 2 * * *", // Щодня о 2:00 AM
	}
}

// NewScheduler створює новий Cleanup Scheduler
func NewScheduler(
	oppRepo repository.OpportunityRepository,
	arbRepo repository.ArbitrageRepository,
	defiRepo repository.DeFiRepository,
	notifRepo repository.NotificationRepository,
	config *Config,
) *Scheduler {
	if config == nil {
		config = DefaultConfig()
	}

	return &Scheduler{
		cron:      cron.New(),
		oppRepo:   oppRepo,
		arbRepo:   arbRepo,
		defiRepo:  defiRepo,
		notifRepo: notifRepo,
		config:    config,
	}
}

// Start запускає cleanup scheduler
func (s *Scheduler) Start() error {
	_, err := s.cron.AddFunc(s.config.Schedule, func() {
		log.Println("🧹 Starting scheduled cleanup...")
		s.RunCleanup()
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	log.Printf("✅ Cleanup scheduler started (schedule: %s)", s.config.Schedule)

	return nil
}

// Stop зупиняє cleanup scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("Cleanup scheduler stopped")
}

// RunCleanup виконує cleanup операції
func (s *Scheduler) RunCleanup() {
	startTime := time.Now()
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("🧹 Cleanup Job Started")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. Cleanup старих opportunities
	s.cleanupOpportunities()

	// 2. Cleanup старих arbitrage opportunities
	s.cleanupArbitrage()

	// 3. Cleanup старих DeFi opportunities
	s.cleanupDeFi()

	// 4. Cleanup старих notifications
	s.cleanupNotifications()

	elapsed := time.Since(startTime)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("✅ Cleanup completed in %v", elapsed)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// cleanupOpportunities видаляє старі opportunities
func (s *Scheduler) cleanupOpportunities() {
	log.Printf("🗑️  Cleaning up opportunities older than %d days...", s.config.OpportunitiesRetentionDays)

	if err := s.oppRepo.DeleteOld(s.config.OpportunitiesRetentionDays); err != nil {
		log.Printf("❌ Failed to cleanup opportunities: %v", err)
		return
	}

	log.Printf("✅ Opportunities cleanup completed")
}

// cleanupArbitrage видаляє старі arbitrage opportunities
func (s *Scheduler) cleanupArbitrage() {
	log.Printf("🗑️  Cleaning up arbitrage opportunities older than %d days...", s.config.ArbitrageRetentionDays)

	duration := time.Duration(s.config.ArbitrageRetentionDays) * 24 * time.Hour
	if err := s.arbRepo.DeleteOlderThan(duration); err != nil {
		log.Printf("❌ Failed to cleanup arbitrage: %v", err)
		return
	}

	log.Printf("✅ Arbitrage cleanup completed")
}

// cleanupDeFi видаляє старі DeFi opportunities
func (s *Scheduler) cleanupDeFi() {
	log.Printf("🗑️  Cleaning up DeFi opportunities older than %d days...", s.config.DeFiRetentionDays)

	cutoff := time.Now().AddDate(0, 0, -s.config.DeFiRetentionDays)
	if err := s.defiRepo.DeleteOld(cutoff); err != nil {
		log.Printf("❌ Failed to cleanup DeFi: %v", err)
		return
	}

	log.Printf("✅ DeFi cleanup completed")
}

// cleanupNotifications видаляє старі notifications
func (s *Scheduler) cleanupNotifications() {
	log.Println("🗑️  Cleaning up old notifications...")

	// Для sent notifications - зберігаємо довше (90 днів)
	if err := s.notifRepo.DeleteOld(s.config.SentNotificationsRetentionDays); err != nil {
		log.Printf("❌ Failed to cleanup notifications: %v", err)
		return
	}

	log.Printf("✅ Notifications cleanup completed")
}

// RunNow запускає cleanup негайно (для тестування)
func (s *Scheduler) RunNow() {
	s.RunCleanup()
}
