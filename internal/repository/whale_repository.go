package repository

import (
	"crypto-opportunities-bot/internal/models"
	"time"

	"gorm.io/gorm"
)

type WhaleRepository interface {
	// CRUD
	Create(whale *models.WhaleTransaction) error
	GetByID(id uint) (*models.WhaleTransaction, error)
	GetByTxHash(txHash string) (*models.WhaleTransaction, error)
	Update(whale *models.WhaleTransaction) error
	Delete(id uint) error

	// Queries
	GetRecent(limit int) ([]*models.WhaleTransaction, error)
	GetRecentByChain(chain string, limit int) ([]*models.WhaleTransaction, error)
	GetRecentByToken(token string, limit int) ([]*models.WhaleTransaction, error)
	GetByDirection(direction string, limit int) ([]*models.WhaleTransaction, error)
	GetPendingNotifications() ([]*models.WhaleTransaction, error)
	GetLast24h() ([]*models.WhaleTransaction, error)

	// Filters
	GetByMinAmount(minUSD float64, limit int) ([]*models.WhaleTransaction, error)
	GetByChainAndToken(chain, token string, limit int) ([]*models.WhaleTransaction, error)
	GetByAddress(address string, limit int) ([]*models.WhaleTransaction, error)

	// Statistics
	GetStats24h(chain, token string) (*models.WhaleStats, error)
	GetTopTokens24h(limit int) ([]string, error)
	CountLast24h() (int64, error)

	// Actions
	MarkAsNotified(id uint) error
	MarkAsProcessed(id uint) error
	CleanupOld(daysOld int) error
}

type whaleRepository struct {
	db *gorm.DB
}

func NewWhaleRepository(db *gorm.DB) WhaleRepository {
	return &whaleRepository{db: db}
}

// CRUD
func (r *whaleRepository) Create(whale *models.WhaleTransaction) error {
	return r.db.Create(whale).Error
}

func (r *whaleRepository) GetByID(id uint) (*models.WhaleTransaction, error) {
	var whale models.WhaleTransaction
	err := r.db.First(&whale, id).Error
	if err != nil {
		return nil, err
	}
	return &whale, nil
}

func (r *whaleRepository) GetByTxHash(txHash string) (*models.WhaleTransaction, error) {
	var whale models.WhaleTransaction
	err := r.db.Where("tx_hash = ?", txHash).First(&whale).Error
	if err != nil {
		return nil, err
	}
	return &whale, nil
}

func (r *whaleRepository) Update(whale *models.WhaleTransaction) error {
	return r.db.Save(whale).Error
}

func (r *whaleRepository) Delete(id uint) error {
	return r.db.Delete(&models.WhaleTransaction{}, id).Error
}

// Queries
func (r *whaleRepository) GetRecent(limit int) ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	err := r.db.Order("block_timestamp DESC").Limit(limit).Find(&whales).Error
	return whales, err
}

func (r *whaleRepository) GetRecentByChain(chain string, limit int) ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	err := r.db.Where("chain = ?", chain).
		Order("block_timestamp DESC").
		Limit(limit).
		Find(&whales).Error
	return whales, err
}

func (r *whaleRepository) GetRecentByToken(token string, limit int) ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	err := r.db.Where("token = ?", token).
		Order("block_timestamp DESC").
		Limit(limit).
		Find(&whales).Error
	return whales, err
}

func (r *whaleRepository) GetByDirection(direction string, limit int) ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	err := r.db.Where("direction = ?", direction).
		Order("block_timestamp DESC").
		Limit(limit).
		Find(&whales).Error
	return whales, err
}

func (r *whaleRepository) GetPendingNotifications() ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	err := r.db.Where("is_notified = ? AND status = ?", false, models.WhaleStatusNew).
		Order("block_timestamp DESC").
		Find(&whales).Error
	return whales, err
}

func (r *whaleRepository) GetLast24h() ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	err := r.db.Where("block_timestamp >= ?", cutoff).
		Order("block_timestamp DESC").
		Find(&whales).Error
	return whales, err
}

// Filters
func (r *whaleRepository) GetByMinAmount(minUSD float64, limit int) ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	err := r.db.Where("amount_usd >= ?", minUSD).
		Order("amount_usd DESC").
		Limit(limit).
		Find(&whales).Error
	return whales, err
}

func (r *whaleRepository) GetByChainAndToken(chain, token string, limit int) ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	err := r.db.Where("chain = ? AND token = ?", chain, token).
		Order("block_timestamp DESC").
		Limit(limit).
		Find(&whales).Error
	return whales, err
}

func (r *whaleRepository) GetByAddress(address string, limit int) ([]*models.WhaleTransaction, error) {
	var whales []*models.WhaleTransaction
	err := r.db.Where("from_address = ? OR to_address = ?", address, address).
		Order("block_timestamp DESC").
		Limit(limit).
		Find(&whales).Error
	return whales, err
}

// Statistics
func (r *whaleRepository) GetStats24h(chain, token string) (*models.WhaleStats, error) {
	cutoff := time.Now().Add(-24 * time.Hour).Unix()

	stats := &models.WhaleStats{
		Chain: chain,
		Token: token,
	}

	query := r.db.Model(&models.WhaleTransaction{}).
		Where("block_timestamp >= ?", cutoff)

	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if token != "" {
		query = query.Where("token = ?", token)
	}

	// Count
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}
	stats.Last24hCount = int(count)

	// Total volume
	var volumeResult struct {
		TotalVolume float64
		AvgSize     float64
		MaxSize     float64
	}
	if err := query.Select(
		"COALESCE(SUM(amount_usd), 0) as total_volume, "+
			"COALESCE(AVG(amount_usd), 0) as avg_size, "+
			"COALESCE(MAX(amount_usd), 0) as max_size",
	).Scan(&volumeResult).Error; err != nil {
		return nil, err
	}
	stats.Last24hVolume = volumeResult.TotalVolume
	stats.AverageTxSize = volumeResult.AvgSize
	stats.LargestTx = volumeResult.MaxSize

	// Accumulation vs Distribution
	var accumCount int64
	query.Where("direction = ?", models.WhaleDirectionExchangeToWallet).Count(&accumCount)
	stats.AccumulationCount = int(accumCount)

	var distCount int64
	query.Where("direction = ?", models.WhaleDirectionWalletToExchange).Count(&distCount)
	stats.DistributionCount = int(distCount)

	// Net flow
	var netFlow struct {
		Accumulation  float64
		Distribution  float64
	}
	query.Where("direction = ?", models.WhaleDirectionExchangeToWallet).
		Select("COALESCE(SUM(amount_usd), 0) as accumulation").Scan(&netFlow)
	query.Where("direction = ?", models.WhaleDirectionWalletToExchange).
		Select("COALESCE(SUM(amount_usd), 0) as distribution").Scan(&netFlow)
	stats.NetFlow = netFlow.Accumulation - netFlow.Distribution

	return stats, nil
}

func (r *whaleRepository) GetTopTokens24h(limit int) ([]string, error) {
	cutoff := time.Now().Add(-24 * time.Hour).Unix()

	var result []struct {
		Token string
		Count int64
	}

	err := r.db.Model(&models.WhaleTransaction{}).
		Select("token, COUNT(*) as count").
		Where("block_timestamp >= ?", cutoff).
		Group("token").
		Order("count DESC").
		Limit(limit).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	tokens := make([]string, len(result))
	for i, r := range result {
		tokens[i] = r.Token
	}

	return tokens, nil
}

func (r *whaleRepository) CountLast24h() (int64, error) {
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	var count int64
	err := r.db.Model(&models.WhaleTransaction{}).
		Where("block_timestamp >= ?", cutoff).
		Count(&count).Error
	return count, err
}

// Actions
func (r *whaleRepository) MarkAsNotified(id uint) error {
	return r.db.Model(&models.WhaleTransaction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_notified": true,
			"status":      models.WhaleStatusNotified,
		}).Error
}

func (r *whaleRepository) MarkAsProcessed(id uint) error {
	return r.db.Model(&models.WhaleTransaction{}).
		Where("id = ?", id).
		Update("status", models.WhaleStatusProcessed).Error
}

func (r *whaleRepository) CleanupOld(daysOld int) error {
	cutoff := time.Now().AddDate(0, 0, -daysOld).Unix()
	return r.db.Where("block_timestamp < ?", cutoff).
		Delete(&models.WhaleTransaction{}).Error
}
