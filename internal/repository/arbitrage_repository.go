package repository

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ArbitrageRepository interface {
	Create(arb *models.ArbitrageOpportunity) error
	GetByID(id uint) (*models.ArbitrageOpportunity, error)
	GetByExternalID(externalID string) (*models.ArbitrageOpportunity, error)
	GetActive(limit int) ([]*models.ArbitrageOpportunity, error)
	GetActiveByPair(pair string, limit int) ([]*models.ArbitrageOpportunity, error)
	ListAll(limit, offset int) ([]*models.ArbitrageOpportunity, error)
	Update(arb *models.ArbitrageOpportunity) error
	MarkAsNotified(id uint) error
	DeleteExpired() error
	DeleteOlderThan(duration time.Duration) error
	CountActive() (int64, error)
	GetTopByProfit(limit int) ([]*models.ArbitrageOpportunity, error)
}

type arbitrageRepository struct {
	db *gorm.DB
}

func NewArbitrageRepository(db *gorm.DB) ArbitrageRepository {
	return &arbitrageRepository{db: db}
}

// Create створює нову арбітражну можливість
func (r *arbitrageRepository) Create(arb *models.ArbitrageOpportunity) error {
	if arb == nil {
		return fmt.Errorf("arbitrage opportunity is nil")
	}

	return r.db.Create(arb).Error
}

// GetByID отримує арбітраж по ID
func (r *arbitrageRepository) GetByID(id uint) (*models.ArbitrageOpportunity, error) {
	var arb models.ArbitrageOpportunity
	err := r.db.First(&arb, id).Error
	if err != nil {
		return nil, err
	}
	return &arb, nil
}

// GetByExternalID отримує арбітраж по зовнішньому ID
func (r *arbitrageRepository) GetByExternalID(externalID string) (*models.ArbitrageOpportunity, error) {
	var arb models.ArbitrageOpportunity
	err := r.db.Where("external_id = ?", externalID).First(&arb).Error
	if err != nil {
		return nil, err
	}
	return &arb, nil
}

// GetActive отримує активні арбітражні можливості
func (r *arbitrageRepository) GetActive(limit int) ([]*models.ArbitrageOpportunity, error) {
	var arbitrages []*models.ArbitrageOpportunity

	query := r.db.Where("expires_at > ?", time.Now()).
		Order("net_profit_percent DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&arbitrages).Error
	return arbitrages, err
}

// GetActiveByPair отримує активні арбітражі для конкретної пари
func (r *arbitrageRepository) GetActiveByPair(pair string, limit int) ([]*models.ArbitrageOpportunity, error) {
	var arbitrages []*models.ArbitrageOpportunity

	query := r.db.Where("pair = ? AND expires_at > ?", pair, time.Now()).
		Order("net_profit_percent DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&arbitrages).Error
	return arbitrages, err
}

// ListAll отримує всі арбітражі з пагінацією
func (r *arbitrageRepository) ListAll(limit, offset int) ([]*models.ArbitrageOpportunity, error) {
	var arbitrages []*models.ArbitrageOpportunity

	err := r.db.Order("detected_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&arbitrages).Error

	return arbitrages, err
}

// Update оновлює арбітражну можливість
func (r *arbitrageRepository) Update(arb *models.ArbitrageOpportunity) error {
	if arb == nil {
		return fmt.Errorf("arbitrage opportunity is nil")
	}

	return r.db.Save(arb).Error
}

// MarkAsNotified позначає арбітраж як відправлений користувачам
func (r *arbitrageRepository) MarkAsNotified(id uint) error {
	return r.db.Model(&models.ArbitrageOpportunity{}).
		Where("id = ?", id).
		Update("is_notified", true).Error
}

// DeleteExpired видаляє застарілі арбітражі
func (r *arbitrageRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).
		Delete(&models.ArbitrageOpportunity{}).Error
}

// DeleteOlderThan видаляє арбітражі старіші за вказаний період
func (r *arbitrageRepository) DeleteOlderThan(duration time.Duration) error {
	cutoff := time.Now().Add(-duration)
	return r.db.Where("detected_at < ?", cutoff).
		Delete(&models.ArbitrageOpportunity{}).Error
}

// CountActive підраховує кількість активних арбітражів
func (r *arbitrageRepository) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&models.ArbitrageOpportunity{}).
		Where("expires_at > ?", time.Now()).
		Count(&count).Error
	return count, err
}

// GetTopByProfit отримує топ арбітражі по прибутковості
func (r *arbitrageRepository) GetTopByProfit(limit int) ([]*models.ArbitrageOpportunity, error) {
	var arbitrages []*models.ArbitrageOpportunity

	err := r.db.Where("expires_at > ?", time.Now()).
		Order("net_profit_percent DESC").
		Limit(limit).
		Find(&arbitrages).Error

	return arbitrages, err
}
