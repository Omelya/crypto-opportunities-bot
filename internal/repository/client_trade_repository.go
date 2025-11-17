package repository

import (
	"crypto-opportunities-bot/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type ClientTradeRepository interface {
	Create(trade *models.ClientTrade) error
	GetByID(id uint) (*models.ClientTrade, error)
	Update(trade *models.ClientTrade) error
	Delete(id uint) error
	GetByUserID(userID uint, limit int) ([]*models.ClientTrade, error)
	GetByOpportunityID(oppID uint) ([]*models.ClientTrade, error)
	GetRecentByUserID(userID uint, since time.Time) ([]*models.ClientTrade, error)
	GetSuccessfulByUserID(userID uint, limit int) ([]*models.ClientTrade, error)
	GetFailedByUserID(userID uint, limit int) ([]*models.ClientTrade, error)
	CountByUserID(userID uint) (int64, error)
	CountByStatus(status string) (int64, error)
	GetStats(userID uint, since time.Time) (*TradeStats, error)
	List(offset, limit int) ([]*models.ClientTrade, error)
}

type ClientTradeRepositoryImpl struct {
	db *gorm.DB
}

type TradeStats struct {
	TotalTrades      int64
	SuccessfulTrades int64
	FailedTrades     int64
	TotalProfit      float64
	TotalLoss        float64
	AvgProfit        float64
	BestTrade        float64
	WorstTrade       float64
}

func NewClientTradeRepository(db *gorm.DB) ClientTradeRepository {
	return &ClientTradeRepositoryImpl{db: db}
}

func (r *ClientTradeRepositoryImpl) Create(trade *models.ClientTrade) error {
	return r.db.Create(trade).Error
}

func (r *ClientTradeRepositoryImpl) GetByID(id uint) (*models.ClientTrade, error) {
	var trade models.ClientTrade
	err := r.db.Preload("User").Preload("Opportunity").First(&trade, id).Error
	if err != nil {
		return nil, err
	}
	return &trade, nil
}

func (r *ClientTradeRepositoryImpl) Update(trade *models.ClientTrade) error {
	return r.db.Save(trade).Error
}

func (r *ClientTradeRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.ClientTrade{}, id).Error
}

func (r *ClientTradeRepositoryImpl) GetByUserID(userID uint, limit int) ([]*models.ClientTrade, error) {
	var trades []*models.ClientTrade
	query := r.db.Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&trades).Error
	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *ClientTradeRepositoryImpl) GetByOpportunityID(oppID uint) ([]*models.ClientTrade, error) {
	var trades []*models.ClientTrade
	err := r.db.Preload("User").
		Where("opportunity_id = ?", oppID).
		Order("created_at DESC").
		Find(&trades).Error

	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *ClientTradeRepositoryImpl) GetRecentByUserID(userID uint, since time.Time) ([]*models.ClientTrade, error) {
	var trades []*models.ClientTrade
	err := r.db.Where("user_id = ? AND created_at >= ?", userID, since).
		Order("created_at DESC").
		Find(&trades).Error

	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *ClientTradeRepositoryImpl) GetSuccessfulByUserID(userID uint, limit int) ([]*models.ClientTrade, error) {
	var trades []*models.ClientTrade
	err := r.db.Where("user_id = ? AND status = ?", userID, models.TradeStatusCompleted).
		Order("actual_profit DESC").
		Limit(limit).
		Find(&trades).Error

	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *ClientTradeRepositoryImpl) GetFailedByUserID(userID uint, limit int) ([]*models.ClientTrade, error) {
	var trades []*models.ClientTrade
	err := r.db.Where("user_id = ? AND status = ?", userID, models.TradeStatusFailed).
		Order("created_at DESC").
		Limit(limit).
		Find(&trades).Error

	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *ClientTradeRepositoryImpl) CountByUserID(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.ClientTrade{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ClientTradeRepositoryImpl) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&models.ClientTrade{}).
		Where("status = ?", status).
		Count(&count).Error

	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ClientTradeRepositoryImpl) GetStats(userID uint, since time.Time) (*TradeStats, error) {
	var stats TradeStats

	// Count trades by status
	err := r.db.Model(&models.ClientTrade{}).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Count(&stats.TotalTrades).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.ClientTrade{}).
		Where("user_id = ? AND status = ? AND created_at >= ?", userID, models.TradeStatusCompleted, since).
		Count(&stats.SuccessfulTrades).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.ClientTrade{}).
		Where("user_id = ? AND status = ? AND created_at >= ?", userID, models.TradeStatusFailed, since).
		Count(&stats.FailedTrades).Error
	if err != nil {
		return nil, err
	}

	// Calculate profit/loss
	var profitSum struct {
		TotalProfit float64
		AvgProfit   float64
		MaxProfit   float64
		MinProfit   float64
	}

	err = r.db.Model(&models.ClientTrade{}).
		Select("SUM(actual_profit) as total_profit, AVG(actual_profit) as avg_profit, MAX(actual_profit) as max_profit, MIN(actual_profit) as min_profit").
		Where("user_id = ? AND created_at >= ?", userID, since).
		Scan(&profitSum).Error
	if err != nil {
		return nil, err
	}

	stats.TotalProfit = profitSum.TotalProfit
	stats.AvgProfit = profitSum.AvgProfit
	stats.BestTrade = profitSum.MaxProfit
	stats.WorstTrade = profitSum.MinProfit

	// Calculate total loss (sum of negative profits)
	var lossSum float64
	err = r.db.Model(&models.ClientTrade{}).
		Select("SUM(actual_profit)").
		Where("user_id = ? AND actual_profit < 0 AND created_at >= ?", userID, since).
		Scan(&lossSum).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	stats.TotalLoss = -lossSum // Make positive

	return &stats, nil
}

func (r *ClientTradeRepositoryImpl) List(offset, limit int) ([]*models.ClientTrade, error) {
	var trades []*models.ClientTrade
	err := r.db.Preload("User").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&trades).Error

	if err != nil {
		return nil, err
	}
	return trades, nil
}
