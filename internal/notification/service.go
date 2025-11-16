package notification

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Service struct {
	bot       *tgbotapi.BotAPI
	notifRepo repository.NotificationRepository
	userRepo  repository.UserRepository
	prefsRepo repository.UserPreferencesRepository
	oppRepo   repository.OpportunityRepository
	arbRepo   repository.ArbitrageRepository
	formatter *Formatter
	filter    *Filter
}

func NewService(
	bot *tgbotapi.BotAPI,
	notifRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	prefsRepo repository.UserPreferencesRepository,
	oppRepo repository.OpportunityRepository,
	arbRepo repository.ArbitrageRepository,
) *Service {
	return &Service{
		bot:       bot,
		notifRepo: notifRepo,
		userRepo:  userRepo,
		prefsRepo: prefsRepo,
		oppRepo:   oppRepo,
		arbRepo:   arbRepo,
		formatter: NewFormatter(),
		filter:    NewFilter(),
	}
}

func (s *Service) CreateOpportunityNotifications(opp *models.Opportunity) error {
	log.Printf("Creating notifications for opportunity: %s", opp.Title)

	users, err := s.userRepo.List(0, 10000)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	created := 0

	for _, user := range users {
		prefs, err := s.prefsRepo.GetByUserID(user.ID)
		if err != nil {
			log.Printf("Failed to get preferences for user %d: %v", user.ID, err)
			continue
		}

		if prefs == nil {
			log.Printf("No preferences for user %d, skipping", user.ID)
			continue
		}

		if !s.filter.ShouldNotify(user, prefs, opp) {
			continue
		}

		if !user.IsPremium() {
			limit := s.filter.GetDailyAlertLimit(user)
			todayCount, err := s.notifRepo.CountTodayByUser(user.ID)
			if err != nil {
				log.Printf("Failed to count today notifications for user %d: %v", user.ID, err)
				continue
			}

			if todayCount >= int64(limit) {
				log.Printf("Daily limit reached for user %d (%d/%d)", user.ID, todayCount, limit)
				continue
			}
		}

		message := s.formatter.FormatOpportunity(opp)

		priority := s.filter.GetNotificationPriority(user, opp)

		var scheduledFor *time.Time
		if delay := s.filter.CalculateDelay(user); delay > 0 {
			scheduled := time.Now().Add(delay)
			scheduledFor = &scheduled
		}

		notification := &models.Notification{
			UserID:        user.ID,
			OpportunityID: &opp.ID,
			Type:          opp.Type,
			Priority:      priority,
			Status:        models.NotificationStatusPending,
			Message:       message,
			ScheduledFor:  scheduledFor,
			MessageData: models.JSONMap{
				"opportunity_id": opp.ID,
				"exchange":       opp.Exchange,
				"url":            opp.URL,
			},
		}

		if err := s.notifRepo.Create(notification); err != nil {
			log.Printf("Failed to create notification for user %d: %v", user.ID, err)
			continue
		}

		created++
	}

	log.Printf("Created %d notifications for opportunity: %s", created, opp.Title)
	return nil
}

// CreateArbitrageNotifications створює notification для арбітражної можливості (Premium only)
func (s *Service) CreateArbitrageNotifications(arb *models.ArbitrageOpportunity) error {
	log.Printf("Creating arbitrage notifications for: %s (%.2f%% profit)", arb.Pair, arb.NetProfitPercent)

	// Get all premium users
	users, err := s.userRepo.List(0, 10000)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	created := 0

	for _, user := range users {
		// Only Premium users get arbitrage notifications
		if !user.IsPremium() {
			continue
		}

		prefs, err := s.prefsRepo.GetByUserID(user.ID)
		if err != nil {
			log.Printf("Failed to get preferences for user %d: %v", user.ID, err)
			continue
		}

		if prefs == nil {
			log.Printf("No preferences for user %d, skipping", user.ID)
			continue
		}

		// Check if user wants instant notifications
		if !prefs.NotifyInstant {
			continue
		}

		// Format arbitrage message
		message := s.formatter.FormatArbitrage(arb)

		// Premium users get instant notifications (no delay)
		notification := &models.Notification{
			UserID:       user.ID,
			Type:         "arbitrage",
			Priority:     "high",
			Status:       models.NotificationStatusPending,
			Message:      message,
			ScheduledFor: nil, // Instant
			MessageData: models.JSONMap{
				"arbitrage_id":   arb.ID,
				"pair":           arb.Pair,
				"exchange_buy":   arb.ExchangeBuy,
				"exchange_sell":  arb.ExchangeSell,
				"net_profit":     arb.NetProfitPercent,
				"profit_usd":     arb.NetProfitUSD,
			},
		}

		if err := s.notifRepo.Create(notification); err != nil {
			log.Printf("Failed to create arbitrage notification for user %d: %v", user.ID, err)
			continue
		}

		created++
	}

	log.Printf("Created %d arbitrage notifications for: %s", created, arb.Pair)
	return nil
}

func (s *Service) SendPendingNotifications(batchSize int) error {
	notifications, err := s.notifRepo.GetPending(batchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending notifications: %w", err)
	}

	if len(notifications) == 0 {
		return nil
	}

	log.Printf("Sending %d pending notifications", len(notifications))

	sent := 0
	failed := 0

	for _, notification := range notifications {
		if err := s.sendNotification(notification); err != nil {
			log.Printf("Failed to send notification %d: %v", notification.ID, err)
			notification.MarkAsFailed(err.Error())
			failed++
		} else {
			notification.MarkAsSent()
			sent++
		}

		if err := s.notifRepo.Update(notification); err != nil {
			log.Printf("Failed to update notification %d: %v", notification.ID, err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("Sent %d notifications, failed %d", sent, failed)
	return nil
}

func (s *Service) SendDailyDigest(userID uint) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found: %d", userID)
	}

	prefs, err := s.prefsRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get preferences: %w", err)
	}

	if prefs == nil {
		return fmt.Errorf("preferences not found for user: %d", userID)
	}

	if !prefs.DailyDigestEnabled {
		return nil
	}

	opportunities, err := s.getRecentOpportunities(user, prefs, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to get recent opportunities: %w", err)
	}

	message := s.formatter.FormatDailyDigest(opportunities, user)

	if !user.IsPremium() && len(opportunities) > 0 {
		message += s.formatter.FormatPremiumTeaser(10)
	}

	notification := &models.Notification{
		UserID:   userID,
		Type:     "daily_digest",
		Priority: models.NotificationPriorityNormal,
		Status:   models.NotificationStatusPending,
		Message:  message,
	}

	if err := s.notifRepo.Create(notification); err != nil {
		return fmt.Errorf("failed to create digest notification: %w", err)
	}

	if err := s.sendNotification(notification); err != nil {
		notification.MarkAsFailed(err.Error())
		s.notifRepo.Update(notification)
		return fmt.Errorf("failed to send digest: %w", err)
	}

	notification.MarkAsSent()
	s.notifRepo.Update(notification)

	log.Printf("Daily digest sent to user %d", userID)
	return nil
}

func (s *Service) SendDailyDigestToAll() error {
	users, err := s.userRepo.List(0, 10000)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	sent := 0
	failed := 0

	for _, user := range users {
		prefs, err := s.prefsRepo.GetByUserID(user.ID)
		if err != nil || prefs == nil {
			continue
		}

		if !s.filter.ShouldSendDailyDigest(user, prefs) {
			continue
		}

		if err := s.SendDailyDigest(user.ID); err != nil {
			log.Printf("Failed to send digest to user %d: %v", user.ID, err)
			failed++
		} else {
			sent++
		}

		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Daily digest: sent %d, failed %d", sent, failed)
	return nil
}

func (s *Service) RetryFailedNotifications(batchSize int) error {
	notifications, err := s.notifRepo.GetFailed(batchSize)
	if err != nil {
		return fmt.Errorf("failed to get failed notifications: %w", err)
	}

	if len(notifications) == 0 {
		return nil
	}

	log.Printf("Retrying %d failed notifications", len(notifications))

	for _, notification := range notifications {
		waitTime := time.Duration(notification.RetryCount*notification.RetryCount) * time.Minute
		if time.Since(notification.UpdatedAt) < waitTime {
			continue
		}

		if err := s.sendNotification(notification); err != nil {
			log.Printf("Retry failed for notification %d: %v", notification.ID, err)
			notification.MarkAsFailed(err.Error())
		} else {
			notification.MarkAsSent()
		}

		err := s.notifRepo.Update(notification)
		if err != nil {
			log.Printf("Failed to update notification %d: %v", notification.ID, err)
		}
	}

	return nil
}

func (s *Service) sendNotification(notification *models.Notification) error {
	if notification.User.TelegramID == 0 {
		return fmt.Errorf("invalid telegram_id for user %d", notification.UserID)
	}

	msg := tgbotapi.NewMessage(notification.User.TelegramID, notification.Message)
	msg.ParseMode = "HTML"

	if notification.Opportunity != nil && notification.Opportunity.URL != "" {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("🔗 Перейти на біржу", notification.Opportunity.URL),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💰 Всі можливості", "menu_all"),
			),
		)
		msg.ReplyMarkup = keyboard
	}

	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("telegram send error: %w", err)
	}

	return nil
}

func (s *Service) getRecentOpportunities(user *models.User, prefs *models.UserPreferences, duration time.Duration) ([]*models.Opportunity, error) {
	allOpps, err := s.oppRepo.ListActive(100, 0)
	if err != nil {
		return nil, err
	}

	var filtered []*models.Opportunity
	cutoff := time.Now().Add(-duration)

	for _, opp := range allOpps {
		if opp.CreatedAt.Before(cutoff) {
			continue
		}

		if s.filter.ShouldNotify(user, prefs, opp) {
			filtered = append(filtered, opp)
		}
	}

	return filtered, nil
}
