package repository

import (
	"crypto-opportunities-bot/internal/models"
	"time"

	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	// UserAnalytics
	GetUserAnalytics(userID uint) (*models.UserAnalytics, error)
	CreateUserAnalytics(analytics *models.UserAnalytics) error
	UpdateUserAnalytics(analytics *models.UserAnalytics) error
	GetOrCreateUserAnalytics(userID uint) (*models.UserAnalytics, error)
	ListTopUsers(limit int, orderBy string) ([]*models.UserAnalytics, error)

	// DailyStats
	GetDailyStats(date time.Time) (*models.DailyStats, error)
	CreateDailyStats(stats *models.DailyStats) error
	UpdateDailyStats(stats *models.DailyStats) error
	GetOrCreateDailyStats(date time.Time) (*models.DailyStats, error)
	ListDailyStats(from, to time.Time) ([]*models.DailyStats, error)

	// OpportunityStats
	GetOpportunityStats(opportunityID uint) (*models.OpportunityStats, error)
	CreateOpportunityStats(stats *models.OpportunityStats) error
	UpdateOpportunityStats(stats *models.OpportunityStats) error
	GetOrCreateOpportunityStats(opportunityID uint) (*models.OpportunityStats, error)
	ListTopOpportunities(limit int) ([]*models.OpportunityStats, error)

	// UserEngagement
	GetUserEngagement(userID uint, date time.Time) (*models.UserEngagement, error)
	CreateUserEngagement(engagement *models.UserEngagement) error
	UpdateUserEngagement(engagement *models.UserEngagement) error
	GetOrCreateUserEngagement(userID uint, date time.Time) (*models.UserEngagement, error)
	ListUserEngagementHistory(userID uint, days int) ([]*models.UserEngagement, error)
}

type analyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

// UserAnalytics methods

func (r *analyticsRepository) GetUserAnalytics(userID uint) (*models.UserAnalytics, error) {
	var analytics models.UserAnalytics
	err := r.db.Where("user_id = ?", userID).First(&analytics).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &analytics, err
}

func (r *analyticsRepository) CreateUserAnalytics(analytics *models.UserAnalytics) error {
	return r.db.Create(analytics).Error
}

func (r *analyticsRepository) UpdateUserAnalytics(analytics *models.UserAnalytics) error {
	return r.db.Save(analytics).Error
}

func (r *analyticsRepository) GetOrCreateUserAnalytics(userID uint) (*models.UserAnalytics, error) {
	analytics, err := r.GetUserAnalytics(userID)
	if err != nil {
		return nil, err
	}

	if analytics == nil {
		analytics = &models.UserAnalytics{
			UserID: userID,
		}
		if err := r.CreateUserAnalytics(analytics); err != nil {
			return nil, err
		}
	}

	return analytics, nil
}

func (r *analyticsRepository) ListTopUsers(limit int, orderBy string) ([]*models.UserAnalytics, error) {
	var analytics []*models.UserAnalytics

	query := r.db.Limit(limit)

	switch orderBy {
	case "viewed":
		query = query.Order("viewed_opportunities DESC")
	case "participated":
		query = query.Order("participated_opportunities DESC")
	case "conversion":
		query = query.Order("overall_conversion_rate DESC")
	case "engagement":
		query = query.Order("total_sessions DESC")
	default:
		query = query.Order("participated_opportunities DESC")
	}

	err := query.Find(&analytics).Error
	return analytics, err
}

// DailyStats methods

func (r *analyticsRepository) GetDailyStats(date time.Time) (*models.DailyStats, error) {
	var stats models.DailyStats
	dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	err := r.db.Where("date = ?", dateStart).First(&stats).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &stats, err
}

func (r *analyticsRepository) CreateDailyStats(stats *models.DailyStats) error {
	return r.db.Create(stats).Error
}

func (r *analyticsRepository) UpdateDailyStats(stats *models.DailyStats) error {
	return r.db.Save(stats).Error
}

func (r *analyticsRepository) GetOrCreateDailyStats(date time.Time) (*models.DailyStats, error) {
	stats, err := r.GetDailyStats(date)
	if err != nil {
		return nil, err
	}

	if stats == nil {
		dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		stats = &models.DailyStats{
			Date: dateStart,
		}
		if err := r.CreateDailyStats(stats); err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func (r *analyticsRepository) ListDailyStats(from, to time.Time) ([]*models.DailyStats, error) {
	var stats []*models.DailyStats
	err := r.db.Where("date >= ? AND date <= ?", from, to).
		Order("date ASC").
		Find(&stats).Error
	return stats, err
}

// OpportunityStats methods

func (r *analyticsRepository) GetOpportunityStats(opportunityID uint) (*models.OpportunityStats, error) {
	var stats models.OpportunityStats
	err := r.db.Where("opportunity_id = ?", opportunityID).First(&stats).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &stats, err
}

func (r *analyticsRepository) CreateOpportunityStats(stats *models.OpportunityStats) error {
	return r.db.Create(stats).Error
}

func (r *analyticsRepository) UpdateOpportunityStats(stats *models.OpportunityStats) error {
	return r.db.Save(stats).Error
}

func (r *analyticsRepository) GetOrCreateOpportunityStats(opportunityID uint) (*models.OpportunityStats, error) {
	stats, err := r.GetOpportunityStats(opportunityID)
	if err != nil {
		return nil, err
	}

	if stats == nil {
		stats = &models.OpportunityStats{
			OpportunityID: opportunityID,
		}
		if err := r.CreateOpportunityStats(stats); err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func (r *analyticsRepository) ListTopOpportunities(limit int) ([]*models.OpportunityStats, error) {
	var stats []*models.OpportunityStats
	err := r.db.Order("performance_score DESC").
		Limit(limit).
		Preload("Opportunity").
		Find(&stats).Error
	return stats, err
}

// UserEngagement methods

func (r *analyticsRepository) GetUserEngagement(userID uint, date time.Time) (*models.UserEngagement, error) {
	var engagement models.UserEngagement
	dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	err := r.db.Where("user_id = ? AND date = ?", userID, dateStart).First(&engagement).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &engagement, err
}

func (r *analyticsRepository) CreateUserEngagement(engagement *models.UserEngagement) error {
	return r.db.Create(engagement).Error
}

func (r *analyticsRepository) UpdateUserEngagement(engagement *models.UserEngagement) error {
	return r.db.Save(engagement).Error
}

func (r *analyticsRepository) GetOrCreateUserEngagement(userID uint, date time.Time) (*models.UserEngagement, error) {
	engagement, err := r.GetUserEngagement(userID, date)
	if err != nil {
		return nil, err
	}

	if engagement == nil {
		dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		now := time.Now()
		engagement = &models.UserEngagement{
			UserID:          userID,
			Date:            dateStart,
			FirstActivityAt: &now,
			LastActivityAt:  &now,
		}
		if err := r.CreateUserEngagement(engagement); err != nil {
			return nil, err
		}
	}

	return engagement, nil
}

func (r *analyticsRepository) ListUserEngagementHistory(userID uint, days int) ([]*models.UserEngagement, error) {
	var engagements []*models.UserEngagement
	startDate := time.Now().AddDate(0, 0, -days)
	err := r.db.Where("user_id = ? AND date >= ?", userID, startDate).
		Order("date DESC").
		Find(&engagements).Error
	return engagements, err
}
