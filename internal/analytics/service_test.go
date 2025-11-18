package analytics

import (
	"crypto-opportunities-bot/internal/models"
	"testing"
	"time"
)

// Mock repositories for testing
type mockAnalyticsRepo struct {
	userAnalytics map[uint]*models.UserAnalytics
	dailyStats    map[string]*models.DailyStats
	oppStats      map[uint]*models.OpportunityStats
	engagement    map[string]*models.UserEngagement
}

func newMockAnalyticsRepo() *mockAnalyticsRepo {
	return &mockAnalyticsRepo{
		userAnalytics: make(map[uint]*models.UserAnalytics),
		dailyStats:    make(map[string]*models.DailyStats),
		oppStats:      make(map[uint]*models.OpportunityStats),
		engagement:    make(map[string]*models.UserEngagement),
	}
}

func (m *mockAnalyticsRepo) GetUserAnalytics(userID uint) (*models.UserAnalytics, error) {
	return m.userAnalytics[userID], nil
}

func (m *mockAnalyticsRepo) CreateUserAnalytics(analytics *models.UserAnalytics) error {
	m.userAnalytics[analytics.UserID] = analytics
	return nil
}

func (m *mockAnalyticsRepo) UpdateUserAnalytics(analytics *models.UserAnalytics) error {
	m.userAnalytics[analytics.UserID] = analytics
	return nil
}

func (m *mockAnalyticsRepo) GetOrCreateUserAnalytics(userID uint) (*models.UserAnalytics, error) {
	if analytics, exists := m.userAnalytics[userID]; exists {
		return analytics, nil
	}
	analytics := &models.UserAnalytics{UserID: userID}
	m.userAnalytics[userID] = analytics
	return analytics, nil
}

func (m *mockAnalyticsRepo) ListTopUsers(limit int, orderBy string) ([]*models.UserAnalytics, error) {
	result := make([]*models.UserAnalytics, 0)
	for _, a := range m.userAnalytics {
		result = append(result, a)
	}
	return result, nil
}

func (m *mockAnalyticsRepo) GetDailyStats(date time.Time) (*models.DailyStats, error) {
	key := date.Format("2006-01-02")
	return m.dailyStats[key], nil
}

func (m *mockAnalyticsRepo) CreateDailyStats(stats *models.DailyStats) error {
	key := stats.Date.Format("2006-01-02")
	m.dailyStats[key] = stats
	return nil
}

func (m *mockAnalyticsRepo) UpdateDailyStats(stats *models.DailyStats) error {
	key := stats.Date.Format("2006-01-02")
	m.dailyStats[key] = stats
	return nil
}

func (m *mockAnalyticsRepo) GetOrCreateDailyStats(date time.Time) (*models.DailyStats, error) {
	key := date.Format("2006-01-02")
	if stats, exists := m.dailyStats[key]; exists {
		return stats, nil
	}
	stats := &models.DailyStats{Date: date}
	m.dailyStats[key] = stats
	return stats, nil
}

func (m *mockAnalyticsRepo) ListDailyStats(from, to time.Time) ([]*models.DailyStats, error) {
	result := make([]*models.DailyStats, 0)
	for _, s := range m.dailyStats {
		if s.Date.After(from) && s.Date.Before(to) || s.Date.Equal(from) || s.Date.Equal(to) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockAnalyticsRepo) GetOpportunityStats(opportunityID uint) (*models.OpportunityStats, error) {
	return m.oppStats[opportunityID], nil
}

func (m *mockAnalyticsRepo) CreateOpportunityStats(stats *models.OpportunityStats) error {
	m.oppStats[stats.OpportunityID] = stats
	return nil
}

func (m *mockAnalyticsRepo) UpdateOpportunityStats(stats *models.OpportunityStats) error {
	m.oppStats[stats.OpportunityID] = stats
	return nil
}

func (m *mockAnalyticsRepo) GetOrCreateOpportunityStats(opportunityID uint) (*models.OpportunityStats, error) {
	if stats, exists := m.oppStats[opportunityID]; exists {
		return stats, nil
	}
	stats := &models.OpportunityStats{OpportunityID: opportunityID}
	m.oppStats[opportunityID] = stats
	return stats, nil
}

func (m *mockAnalyticsRepo) ListTopOpportunities(limit int) ([]*models.OpportunityStats, error) {
	result := make([]*models.OpportunityStats, 0)
	for _, s := range m.oppStats {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockAnalyticsRepo) GetUserEngagement(userID uint, date time.Time) (*models.UserEngagement, error) {
	key := date.Format("2006-01-02") + "-" + string(rune(userID))
	return m.engagement[key], nil
}

func (m *mockAnalyticsRepo) CreateUserEngagement(engagement *models.UserEngagement) error {
	key := engagement.Date.Format("2006-01-02") + "-" + string(rune(engagement.UserID))
	m.engagement[key] = engagement
	return nil
}

func (m *mockAnalyticsRepo) UpdateUserEngagement(engagement *models.UserEngagement) error {
	key := engagement.Date.Format("2006-01-02") + "-" + string(rune(engagement.UserID))
	m.engagement[key] = engagement
	return nil
}

func (m *mockAnalyticsRepo) GetOrCreateUserEngagement(userID uint, date time.Time) (*models.UserEngagement, error) {
	key := date.Format("2006-01-02") + "-" + string(rune(userID))
	if eng, exists := m.engagement[key]; exists {
		return eng, nil
	}
	now := time.Now()
	eng := &models.UserEngagement{
		UserID:          userID,
		Date:            date,
		FirstActivityAt: &now,
		LastActivityAt:  &now,
	}
	m.engagement[key] = eng
	return eng, nil
}

func (m *mockAnalyticsRepo) ListUserEngagementHistory(userID uint, days int) ([]*models.UserEngagement, error) {
	result := make([]*models.UserEngagement, 0)
	for _, e := range m.engagement {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

type mockActionRepo struct{}

func (m *mockActionRepo) Create(action *models.UserAction) error {
	return nil
}

func (m *mockActionRepo) CountByUserAndType(userID uint, actionType string, opportunityID uint) (int64, error) {
	return 1, nil
}

type mockUserRepo struct{}

func (m *mockUserRepo) GetByID(id uint) (*models.User, error) {
	return &models.User{
		BaseModel:          models.BaseModel{ID: id},
		SubscriptionTier:   "free",
	}, nil
}

type mockOppRepo struct{}

func (m *mockOppRepo) ListActive(limit, offset int) ([]*models.Opportunity, error) {
	return []*models.Opportunity{}, nil
}

// Tests

func TestTrackAction(t *testing.T) {
	analyticsRepo := newMockAnalyticsRepo()
	actionRepo := &mockActionRepo{}
	userRepo := &mockUserRepo{}
	oppRepo := &mockOppRepo{}

	service := NewService(analyticsRepo, actionRepo, userRepo, oppRepo)

	userID := uint(1)
	oppID := uint(100)

	// Track view
	err := service.TrackAction(userID, models.ActionTypeViewed, &oppID, nil)
	if err != nil {
		t.Fatalf("Failed to track action: %v", err)
	}

	// Check user analytics updated
	analytics, err := analyticsRepo.GetUserAnalytics(userID)
	if err != nil {
		t.Fatalf("Failed to get user analytics: %v", err)
	}

	if analytics.ViewedOpportunities != 1 {
		t.Errorf("Expected 1 viewed opportunity, got %d", analytics.ViewedOpportunities)
	}
}

func TestRecordSession(t *testing.T) {
	analyticsRepo := newMockAnalyticsRepo()
	actionRepo := &mockActionRepo{}
	userRepo := &mockUserRepo{}
	oppRepo := &mockOppRepo{}

	service := NewService(analyticsRepo, actionRepo, userRepo, oppRepo)

	userID := uint(1)
	duration := 300 // 5 minutes

	err := service.RecordSession(userID, duration)
	if err != nil {
		t.Fatalf("Failed to record session: %v", err)
	}

	analytics, err := analyticsRepo.GetUserAnalytics(userID)
	if err != nil {
		t.Fatalf("Failed to get user analytics: %v", err)
	}

	if analytics.TotalSessions != 1 {
		t.Errorf("Expected 1 session, got %d", analytics.TotalSessions)
	}

	if analytics.TotalTimeSpent != duration {
		t.Errorf("Expected %d seconds time spent, got %d", duration, analytics.TotalTimeSpent)
	}

	if analytics.AverageSessionTime != duration {
		t.Errorf("Expected %d seconds average, got %d", duration, analytics.AverageSessionTime)
	}
}

func TestConversionRateCalculation(t *testing.T) {
	analytics := &models.UserAnalytics{
		ViewedOpportunities:       100,
		ClickedOpportunities:      50,
		ParticipatedOpportunities: 20,
	}

	analytics.CalculateConversionRates()

	expectedViewToClick := 50.0
	if analytics.ViewToClickRate != expectedViewToClick {
		t.Errorf("Expected view to click rate %.2f, got %.2f", expectedViewToClick, analytics.ViewToClickRate)
	}

	expectedClickToParticipate := 40.0
	if analytics.ClickToParticipateRate != expectedClickToParticipate {
		t.Errorf("Expected click to participate rate %.2f, got %.2f", expectedClickToParticipate, analytics.ClickToParticipateRate)
	}

	expectedOverall := 20.0
	if analytics.OverallConversionRate != expectedOverall {
		t.Errorf("Expected overall conversion rate %.2f, got %.2f", expectedOverall, analytics.OverallConversionRate)
	}
}

func TestEngagementLevelCalculation(t *testing.T) {
	tests := []struct {
		name     string
		sessions int
		time     int
		actions  int
		expected string
	}{
		{"High engagement", 5, 1800, 50, "high"},
		{"Medium engagement", 2, 600, 15, "medium"},
		{"Low engagement", 1, 120, 3, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engagement := &models.UserEngagement{
				SessionsCount:          tt.sessions,
				TimeSpent:              tt.time,
				ActionsCount:           tt.actions,
				OpportunitiesParticipated: 0,
			}

			engagement.CalculateEngagementLevel()

			if engagement.EngagementLevel != tt.expected {
				t.Errorf("Expected engagement level %s, got %s", tt.expected, engagement.EngagementLevel)
			}
		})
	}
}

func TestPerformanceScoreCalculation(t *testing.T) {
	stats := &models.OpportunityStats{
		UniqueViews:         200,
		UniqueClicks:        100,
		UniqueParticipations: 50,
	}

	stats.CalculateConversionRates()
	stats.CalculatePerformanceScore()

	if stats.PerformanceScore <= 0 || stats.PerformanceScore > 100 {
		t.Errorf("Performance score should be between 0 and 100, got %.2f", stats.PerformanceScore)
	}

	// High conversion should result in high score
	if stats.PerformanceScore < 30 {
		t.Errorf("Expected high performance score for good conversion, got %.2f", stats.PerformanceScore)
	}
}
