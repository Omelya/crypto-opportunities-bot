package repository

import (
	"crypto-opportunities-bot/internal/models"
	"errors"

	"gorm.io/gorm"
)

type ClientStatisticsRepository interface {
	Create(stats *models.ClientStatistics) error
	GetByID(id uint) (*models.ClientStatistics, error)
	GetByUserID(userID uint) (*models.ClientStatistics, error)
	Update(stats *models.ClientStatistics) error
	UpdateFromTrade(trade *models.ClientTrade) error
	GetOrCreate(userID uint) (*models.ClientStatistics, error)
	GetLeaderboard(limit int) ([]*models.ClientStatistics, error)
	GetLeaderboardByProfit(limit int) ([]*models.ClientStatistics, error)
	GetLeaderboardByWinRate(limit int) ([]*models.ClientStatistics, error)
	RecalculateStats(userID uint) error
	List(offset, limit int) ([]*models.ClientStatistics, error)
}

type ClientStatisticsRepositoryImpl struct {
	db *gorm.DB
}

func NewClientStatisticsRepository(db *gorm.DB) ClientStatisticsRepository {
	return &ClientStatisticsRepositoryImpl{db: db}
}

func (r *ClientStatisticsRepositoryImpl) Create(stats *models.ClientStatistics) error {
	return r.db.Create(stats).Error
}

func (r *ClientStatisticsRepositoryImpl) GetByID(id uint) (*models.ClientStatistics, error) {
	var stats models.ClientStatistics
	err := r.db.Preload("User").First(&stats, id).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *ClientStatisticsRepositoryImpl) GetByUserID(userID uint) (*models.ClientStatistics, error) {
	var stats models.ClientStatistics
	err := r.db.Preload("User").Where("user_id = ?", userID).First(&stats).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

func (r *ClientStatisticsRepositoryImpl) Update(stats *models.ClientStatistics) error {
	return r.db.Save(stats).Error
}

func (r *ClientStatisticsRepositoryImpl) UpdateFromTrade(trade *models.ClientTrade) error {
	// Get or create statistics
	stats, err := r.GetOrCreate(trade.UserID)
	if err != nil {
		return err
	}

	// Update stats based on trade
	stats.UpdateFromTrade(trade)

	// Save updated statistics
	return r.Update(stats)
}

func (r *ClientStatisticsRepositoryImpl) GetOrCreate(userID uint) (*models.ClientStatistics, error) {
	stats, err := r.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	// If not found, create new
	if stats == nil {
		stats = &models.ClientStatistics{
			UserID:           userID,
			TotalTrades:      0,
			SuccessfulTrades: 0,
			FailedTrades:     0,
			TotalProfit:      0,
			TotalLoss:        0,
			NetProfit:        0,
			BestTrade:        0,
			WorstTrade:       0,
			AvgProfit:        0,
			WinRate:          0,
			TotalVolume:      0,
		}

		if err := r.Create(stats); err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func (r *ClientStatisticsRepositoryImpl) GetLeaderboard(limit int) ([]*models.ClientStatistics, error) {
	// Leaderboard по чистому прибутку
	return r.GetLeaderboardByProfit(limit)
}

func (r *ClientStatisticsRepositoryImpl) GetLeaderboardByProfit(limit int) ([]*models.ClientStatistics, error) {
	var stats []*models.ClientStatistics
	err := r.db.Preload("User").
		Where("total_trades > ?", 0).
		Order("net_profit DESC").
		Limit(limit).
		Find(&stats).Error

	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *ClientStatisticsRepositoryImpl) GetLeaderboardByWinRate(limit int) ([]*models.ClientStatistics, error) {
	var stats []*models.ClientStatistics
	err := r.db.Preload("User").
		Where("total_trades >= ?", 10). // Мінімум 10 трейдів для fair comparison
		Order("win_rate DESC, net_profit DESC").
		Limit(limit).
		Find(&stats).Error

	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *ClientStatisticsRepositoryImpl) RecalculateStats(userID uint) error {
	// Get all completed trades
	var trades []*models.ClientTrade
	err := r.db.Where("user_id = ? AND status IN (?)", userID, []string{models.TradeStatusCompleted, models.TradeStatusFailed}).
		Find(&trades).Error
	if err != nil {
		return err
	}

	// Get or create stats
	stats, err := r.GetOrCreate(userID)
	if err != nil {
		return err
	}

	// Reset stats
	stats.TotalTrades = 0
	stats.SuccessfulTrades = 0
	stats.FailedTrades = 0
	stats.TotalProfit = 0
	stats.TotalLoss = 0
	stats.NetProfit = 0
	stats.BestTrade = 0
	stats.WorstTrade = 0
	stats.AvgProfit = 0
	stats.WinRate = 0
	stats.TotalVolume = 0

	// Recalculate from all trades
	for _, trade := range trades {
		stats.UpdateFromTrade(trade)
	}

	// Save
	return r.Update(stats)
}

func (r *ClientStatisticsRepositoryImpl) List(offset, limit int) ([]*models.ClientStatistics, error) {
	var stats []*models.ClientStatistics
	err := r.db.Preload("User").
		Offset(offset).
		Limit(limit).
		Order("net_profit DESC").
		Find(&stats).Error

	if err != nil {
		return nil, err
	}
	return stats, nil
}
