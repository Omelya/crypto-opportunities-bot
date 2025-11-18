package repository

import (
	"crypto-opportunities-bot/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type OpportunityRepository interface {
	Create(opp *models.Opportunity) error
	GetByID(id uint) (*models.Opportunity, error)
	GetByExternalID(externalID string) (*models.Opportunity, error)
	Update(opp *models.Opportunity) error
	Delete(id uint) error
	ListActive(limit, offset int) ([]*models.Opportunity, error)
	ListCreatedToday(limit, offset int) ([]*models.Opportunity, error)
	ListByType(oppType string, limit, offset int) ([]*models.Opportunity, error)
	ListByExchange(exchange string, limit, offset int) ([]*models.Opportunity, error)
	ListByFilters(filters OpportunityFilters) ([]*models.Opportunity, error)
	CountActive() (int64, error)
	CountByType(oppType string) (int64, error)
	DeactivateExpired() error
	DeleteOld(days int) error
}

type OpportunityFilters struct {
	Exchange      string
	Type          string
	MinROI        float64
	MaxInvestment float64
	IsActive      bool
	Limit         int
	Offset        int
}

type opportunityRepository struct {
	db *gorm.DB
}

func NewOpportunityRepository(db *gorm.DB) OpportunityRepository {
	return &opportunityRepository{db: db}
}

func (r *opportunityRepository) Create(opp *models.Opportunity) error {
	return r.db.Create(opp).Error
}

func (r *opportunityRepository) GetByID(id uint) (*models.Opportunity, error) {
	var opp models.Opportunity
	err := r.db.First(&opp, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &opp, nil
}

func (r *opportunityRepository) GetByExternalID(externalID string) (*models.Opportunity, error) {
	var opp models.Opportunity
	err := r.db.Where("external_id = ?", externalID).First(&opp).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &opp, nil
}

func (r *opportunityRepository) Update(opp *models.Opportunity) error {
	return r.db.Save(opp).Error
}

func (r *opportunityRepository) Delete(id uint) error {
	return r.db.Delete(&models.Opportunity{}, id).Error
}

func (r *opportunityRepository) ListActive(limit, offset int) ([]*models.Opportunity, error) {
	var opps []*models.Opportunity
	err := r.db.
		Where("is_active = ?", true).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&opps).Error

	return opps, err
}

func (r *opportunityRepository) ListCreatedToday(limit, offset int) ([]*models.Opportunity, error) {
	var opps []*models.Opportunity

	// Get start of today (00:00:00)
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	err := r.db.
		Where("is_active = ? AND created_at >= ?", true, startOfDay).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&opps).Error

	return opps, err
}

func (r *opportunityRepository) ListByType(oppType string, limit, offset int) ([]*models.Opportunity, error) {
	var opps []*models.Opportunity
	err := r.db.
		Where("type = ? AND is_active = ?", oppType, true).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&opps).Error

	return opps, err
}

func (r *opportunityRepository) ListByExchange(exchange string, limit, offset int) ([]*models.Opportunity, error) {
	var opps []*models.Opportunity
	err := r.db.
		Where("exchange = ? AND is_active = ?", exchange, true).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&opps).Error

	return opps, err
}

func (r *opportunityRepository) ListByFilters(filters OpportunityFilters) ([]*models.Opportunity, error) {
	query := r.db.Model(&models.Opportunity{})

	if filters.Exchange != "" {
		query = query.Where("exchange = ?", filters.Exchange)
	}

	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}

	if filters.MinROI > 0 {
		query = query.Where("estimated_roi >= ?", filters.MinROI)
	}

	if filters.MaxInvestment > 0 {
		query = query.Where("min_investment <= ?", filters.MaxInvestment)
	}

	query = query.Where("is_active = ?", filters.IsActive)

	var opps []*models.Opportunity
	err := query.
		Order("created_at DESC").
		Limit(filters.Limit).
		Offset(filters.Offset).
		Find(&opps).Error

	return opps, err
}

func (r *opportunityRepository) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&models.Opportunity{}).
		Where("is_active = ?", true).
		Count(&count).Error

	return count, err
}

func (r *opportunityRepository) CountByType(oppType string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Opportunity{}).
		Where("type = ? AND is_active = ?", oppType, true).
		Count(&count).Error

	return count, err
}

func (r *opportunityRepository) DeactivateExpired() error {
	return r.db.Model(&models.Opportunity{}).
		Where("end_date < ? AND is_active = ?", time.Now(), true).
		Update("is_active", false).Error
}

func (r *opportunityRepository) DeleteOld(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)

	return r.db.
		Where("created_at < ? AND is_active = ?", cutoff, false).
		Delete(&models.Opportunity{}).Error
}
