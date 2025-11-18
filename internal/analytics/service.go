package analytics

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"fmt"
	"log"
	"time"
)

type Service struct {
	analyticsRepo repository.AnalyticsRepository
	actionRepo    repository.UserActionRepository
	userRepo      repository.UserRepository
	oppRepo       repository.OpportunityRepository
}

func NewService(
	analyticsRepo repository.AnalyticsRepository,
	actionRepo repository.UserActionRepository,
	userRepo repository.UserRepository,
	oppRepo repository.OpportunityRepository,
) *Service {
	return &Service{
		analyticsRepo: analyticsRepo,
		actionRepo:    actionRepo,
		userRepo:      userRepo,
		oppRepo:       oppRepo,
	}
}

// TrackAction tracks user action and updates analytics
func (s *Service) TrackAction(userID uint, actionType string, opportunityID *uint, metadata map[string]interface{}) error {
	// Create user action record
	action := &models.UserAction{
		UserID:        userID,
		OpportunityID: opportunityID,
		ActionType:    actionType,
		Metadata:      metadata,
	}

	if err := s.actionRepo.Create(action); err != nil {
		log.Printf("Error creating user action: %v", err)
		return err
	}

	// Update user analytics
	if err := s.updateUserAnalytics(userID, actionType, opportunityID); err != nil {
		log.Printf("Error updating user analytics: %v", err)
		return err
	}

	// Update daily stats
	if err := s.updateDailyStats(actionType); err != nil {
		log.Printf("Error updating daily stats: %v", err)
		return err
	}

	// Update opportunity stats if applicable
	if opportunityID != nil {
		if err := s.updateOpportunityStats(*opportunityID, userID, actionType); err != nil {
			log.Printf("Error updating opportunity stats: %v", err)
			return err
		}
	}

	// Update user engagement
	if err := s.updateUserEngagement(userID, actionType); err != nil {
		log.Printf("Error updating user engagement: %v", err)
		return err
	}

	return nil
}

// updateUserAnalytics updates aggregated user analytics
func (s *Service) updateUserAnalytics(userID uint, actionType string, opportunityID *uint) error {
	analytics, err := s.analyticsRepo.GetOrCreateUserAnalytics(userID)
	if err != nil {
		return err
	}

	now := time.Now()
	analytics.LastActivityAt = &now

	// Update user registration days
	user, err := s.userRepo.GetByID(userID)
	if err == nil && user != nil {
		analytics.DaysSinceRegistration = int(time.Since(user.CreatedAt).Hours() / 24)
	}

	switch actionType {
	case models.ActionTypeViewed:
		analytics.ViewedOpportunities++
	case models.ActionTypeClicked:
		analytics.ClickedOpportunities++
	case models.ActionTypeParticipated:
		analytics.ParticipatedOpportunities++
	case models.ActionTypeIgnored:
		analytics.IgnoredOpportunities++
	}

	// Recalculate conversion rates
	analytics.CalculateConversionRates()

	return s.analyticsRepo.UpdateUserAnalytics(analytics)
}

// updateDailyStats updates platform-wide daily statistics
func (s *Service) updateDailyStats(actionType string) error {
	today := time.Now()
	stats, err := s.analyticsRepo.GetOrCreateDailyStats(today)
	if err != nil {
		return err
	}

	switch actionType {
	case models.ActionTypeViewed:
		stats.ViewedOpportunities++
	case models.ActionTypeClicked:
		stats.ClickedOpportunities++
	case models.ActionTypeParticipated:
		stats.ParticipatedOpportunities++
	}

	// Recalculate conversion rate
	stats.CalculateRates()

	return s.analyticsRepo.UpdateDailyStats(stats)
}

// updateOpportunityStats updates statistics for a specific opportunity
func (s *Service) updateOpportunityStats(opportunityID, userID uint, actionType string) error {
	stats, err := s.analyticsRepo.GetOrCreateOpportunityStats(opportunityID)
	if err != nil {
		return err
	}

	// Check if this user has already performed this action
	count, err := s.actionRepo.CountByUserAndType(userID, actionType, opportunityID)
	if err != nil {
		return err
	}

	isFirstTime := count == 1 // We just created the action, so count should be 1 for first time

	switch actionType {
	case models.ActionTypeViewed:
		stats.TotalViews++
		if isFirstTime {
			stats.UniqueViews++
		}
	case models.ActionTypeClicked:
		stats.TotalClicks++
		if isFirstTime {
			stats.UniqueClicks++
		}
	case models.ActionTypeParticipated:
		stats.TotalParticipations++
		if isFirstTime {
			stats.UniqueParticipations++
		}
	case models.ActionTypeIgnored:
		stats.TotalIgnores++
		if isFirstTime {
			stats.UniqueIgnores++
		}
	}

	// Determine if user is premium
	user, err := s.userRepo.GetByID(userID)
	if err == nil && user != nil {
		if user.IsPremium() {
			stats.PremiumUserViews++
		} else {
			stats.FreeUserViews++
		}
	}

	// Recalculate metrics
	stats.CalculateConversionRates()
	stats.CalculatePerformanceScore()

	return s.analyticsRepo.UpdateOpportunityStats(stats)
}

// updateUserEngagement updates daily user engagement
func (s *Service) updateUserEngagement(userID uint, actionType string) error {
	today := time.Now()
	engagement, err := s.analyticsRepo.GetOrCreateUserEngagement(userID, today)
	if err != nil {
		return err
	}

	now := time.Now()
	engagement.LastActivityAt = &now
	engagement.ActionsCount++

	switch actionType {
	case models.ActionTypeViewed:
		engagement.OpportunitiesViewed++
	case models.ActionTypeClicked:
		engagement.OpportunitiesClicked++
	case models.ActionTypeParticipated:
		engagement.OpportunitiesParticipated++
	}

	// Recalculate engagement level
	engagement.CalculateEngagementLevel()

	return s.analyticsRepo.UpdateUserEngagement(engagement)
}

// RecordSession records a user session
func (s *Service) RecordSession(userID uint, duration int) error {
	// Update user analytics
	analytics, err := s.analyticsRepo.GetOrCreateUserAnalytics(userID)
	if err != nil {
		return err
	}

	analytics.TotalSessions++
	analytics.TotalTimeSpent += duration
	analytics.CalculateConversionRates() // This also calculates average session time

	now := time.Now()
	analytics.LastActivityAt = &now

	if err := s.analyticsRepo.UpdateUserAnalytics(analytics); err != nil {
		return err
	}

	// Update daily stats
	stats, err := s.analyticsRepo.GetOrCreateDailyStats(time.Now())
	if err != nil {
		return err
	}

	stats.TotalSessions++
	if stats.TotalSessions > 0 && stats.AverageSessionTime > 0 {
		// Recalculate average
		totalTime := stats.AverageSessionTime * (stats.TotalSessions - 1)
		stats.AverageSessionTime = (totalTime + duration) / stats.TotalSessions
	} else {
		stats.AverageSessionTime = duration
	}

	if err := s.analyticsRepo.UpdateDailyStats(stats); err != nil {
		return err
	}

	// Update user engagement
	engagement, err := s.analyticsRepo.GetOrCreateUserEngagement(userID, time.Now())
	if err != nil {
		return err
	}

	engagement.SessionsCount++
	engagement.TimeSpent += duration
	engagement.CalculateEngagementLevel()

	return s.analyticsRepo.UpdateUserEngagement(engagement)
}

// GetUserAnalytics returns analytics for a specific user
func (s *Service) GetUserAnalytics(userID uint) (*models.UserAnalytics, error) {
	return s.analyticsRepo.GetUserAnalytics(userID)
}

// GetUserEngagementHistory returns engagement history for a user
func (s *Service) GetUserEngagementHistory(userID uint, days int) ([]*models.UserEngagement, error) {
	return s.analyticsRepo.ListUserEngagementHistory(userID, days)
}

// GetDailyStatsRange returns daily stats for a date range
func (s *Service) GetDailyStatsRange(from, to time.Time) ([]*models.DailyStats, error) {
	return s.analyticsRepo.ListDailyStats(from, to)
}

// GetTopOpportunities returns top performing opportunities
func (s *Service) GetTopOpportunities(limit int) ([]*models.OpportunityStats, error) {
	return s.analyticsRepo.ListTopOpportunities(limit)
}

// GetTopUsers returns top users by different metrics
func (s *Service) GetTopUsers(limit int, orderBy string) ([]*models.UserAnalytics, error) {
	return s.analyticsRepo.ListTopUsers(limit, orderBy)
}

// GetPlatformSummary returns overall platform statistics
func (s *Service) GetPlatformSummary() (*PlatformSummary, error) {
	today := time.Now()
	stats, err := s.analyticsRepo.GetDailyStats(today)
	if err != nil {
		return nil, err
	}

	// Get stats for last 7 days
	weekAgo := today.AddDate(0, 0, -7)
	weekStats, err := s.analyticsRepo.ListDailyStats(weekAgo, today)
	if err != nil {
		return nil, err
	}

	summary := &PlatformSummary{
		Today: stats,
	}

	// Calculate weekly aggregates
	for _, s := range weekStats {
		if s != nil {
			summary.WeeklyActiveUsers += s.ActiveUsers
			summary.WeeklyNewUsers += s.NewUsers
			summary.WeeklyRevenue += s.DailyRevenue
			summary.WeeklyNotifications += s.NotificationsSent
		}
	}

	// Average daily active users
	if len(weekStats) > 0 {
		summary.AvgDailyActiveUsers = summary.WeeklyActiveUsers / len(weekStats)
	}

	return summary, nil
}

// CalculateDailyMetrics calculates and updates all daily metrics (run as cron job)
func (s *Service) CalculateDailyMetrics() error {
	today := time.Now()
	stats, err := s.analyticsRepo.GetOrCreateDailyStats(today)
	if err != nil {
		return err
	}

	// Count active opportunities
	opportunities, err := s.oppRepo.ListActive(10000, 0)
	if err == nil {
		stats.TotalOpportunities = len(opportunities)
	}

	// Count users by type
	// This would require additional repository methods
	// For now, we'll leave these as they are updated in real-time

	stats.CalculateRates()

	return s.analyticsRepo.UpdateDailyStats(stats)
}

// RecordNotification records notification metrics
func (s *Service) RecordNotification(userID uint, success bool, opened bool) error {
	// Update user analytics
	analytics, err := s.analyticsRepo.GetOrCreateUserAnalytics(userID)
	if err != nil {
		return err
	}

	analytics.NotificationsReceived++
	if opened {
		analytics.NotificationsOpened++
	}

	if err := s.analyticsRepo.UpdateUserAnalytics(analytics); err != nil {
		return err
	}

	// Update daily stats
	stats, err := s.analyticsRepo.GetOrCreateDailyStats(time.Now())
	if err != nil {
		return err
	}

	if success {
		stats.NotificationsSent++
	} else {
		stats.NotificationsFailed++
	}

	if opened {
		stats.NotificationsOpened++
	}

	return s.analyticsRepo.UpdateDailyStats(stats)
}

// RecordPayment records payment and revenue metrics
func (s *Service) RecordPayment(userID uint, amount float64) error {
	analytics, err := s.analyticsRepo.GetOrCreateUserAnalytics(userID)
	if err != nil {
		return err
	}

	analytics.TotalRevenue += amount
	now := time.Now()
	analytics.LastPaymentAt = &now

	if err := s.analyticsRepo.UpdateUserAnalytics(analytics); err != nil {
		return err
	}

	// Update daily stats
	stats, err := s.analyticsRepo.GetOrCreateDailyStats(time.Now())
	if err != nil {
		return err
	}

	stats.DailyRevenue += amount
	stats.NewSubscriptions++

	return s.analyticsRepo.UpdateDailyStats(stats)
}

// PlatformSummary contains aggregated platform metrics
type PlatformSummary struct {
	Today                *models.DailyStats
	WeeklyActiveUsers    int
	WeeklyNewUsers       int
	WeeklyRevenue        float64
	WeeklyNotifications  int
	AvgDailyActiveUsers  int
}

// FormatSummary formats platform summary for display
func (ps *PlatformSummary) FormatSummary() string {
	if ps.Today == nil {
		return "No data available"
	}

	return fmt.Sprintf(`📊 Platform Summary

Today:
• Active Users: %d
• New Users: %d
• Opportunities Viewed: %d
• Conversion Rate: %.2f%%
• Revenue: $%.2f

This Week:
• Active Users: %d
• New Users: %d
• Notifications Sent: %d
• Revenue: $%.2f
• Avg Daily Active: %d users
`,
		ps.Today.ActiveUsers,
		ps.Today.NewUsers,
		ps.Today.ViewedOpportunities,
		ps.Today.ConversionRate,
		ps.Today.DailyRevenue,
		ps.WeeklyActiveUsers,
		ps.WeeklyNewUsers,
		ps.WeeklyNotifications,
		ps.WeeklyRevenue,
		ps.AvgDailyActiveUsers,
	)
}
