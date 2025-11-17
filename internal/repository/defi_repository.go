package repository

import (
	"crypto-opportunities-bot/internal/models"
	"time"

	"gorm.io/gorm"
)

// DeFiRepository інтерфейс для роботи з DeFi opportunities
type DeFiRepository interface {
	Create(opp *models.DeFiOpportunity) error
	Update(opp *models.DeFiOpportunity) error
	GetByID(id uint) (*models.DeFiOpportunity, error)
	GetByExternalID(externalID string) (*models.DeFiOpportunity, error)
	Delete(id uint) error

	// Listing
	GetActive(limit int) ([]*models.DeFiOpportunity, error)
	GetAll(limit, offset int) ([]*models.DeFiOpportunity, error)

	// Filtering
	GetByChain(chain string, limit int) ([]*models.DeFiOpportunity, error)
	GetByProtocol(protocol string, limit int) ([]*models.DeFiOpportunity, error)
	GetByRiskLevel(riskLevel string, limit int) ([]*models.DeFiOpportunity, error)
	GetHighAPY(minAPY float64, limit int) ([]*models.DeFiOpportunity, error)
	GetHighTVL(minTVL float64, limit int) ([]*models.DeFiOpportunity, error)
	GetStablePairs(maxIL float64, limit int) ([]*models.DeFiOpportunity, error)
	GetAudited(limit int) ([]*models.DeFiOpportunity, error)

	// Advanced queries
	GetByFilters(filters DeFiFilters, limit int) ([]*models.DeFiOpportunity, error)
	GetTopByAPY(limit int) ([]*models.DeFiOpportunity, error)
	GetTopByTVL(limit int) ([]*models.DeFiOpportunity, error)

	// Maintenance
	DeleteOld(before time.Time) error
	MarkInactive(externalIDs []string) error
	CountActive() (int64, error)
}

// DeFiFilters фільтри для пошуку
type DeFiFilters struct {
	Chains      []string
	Protocols   []string
	RiskLevels  []string
	MinAPY      float64
	MaxAPY      float64
	MinTVL      float64
	MaxTVL      float64
	MaxIL       float64
	MinDeposit  float64
	MaxDeposit  float64
	OnlyAudited bool
	NoLockup    bool
}

// DeFiRepositoryImpl implementation
type DeFiRepositoryImpl struct {
	db *gorm.DB
}

// NewDeFiRepository створює новий DeFi repository
func NewDeFiRepository(db *gorm.DB) DeFiRepository {
	return &DeFiRepositoryImpl{db: db}
}

// Create створює нову DeFi opportunity
func (r *DeFiRepositoryImpl) Create(opp *models.DeFiOpportunity) error {
	return r.db.Create(opp).Error
}

// Update оновлює DeFi opportunity
func (r *DeFiRepositoryImpl) Update(opp *models.DeFiOpportunity) error {
	return r.db.Save(opp).Error
}

// GetByID отримує DeFi opportunity за ID
func (r *DeFiRepositoryImpl) GetByID(id uint) (*models.DeFiOpportunity, error) {
	var opp models.DeFiOpportunity
	err := r.db.First(&opp, id).Error
	if err != nil {
		return nil, err
	}
	return &opp, nil
}

// GetByExternalID отримує DeFi opportunity за зовнішнім ID
func (r *DeFiRepositoryImpl) GetByExternalID(externalID string) (*models.DeFiOpportunity, error) {
	var opp models.DeFiOpportunity
	err := r.db.Where("external_id = ?", externalID).First(&opp).Error
	if err != nil {
		return nil, err
	}
	return &opp, nil
}

// Delete видаляє DeFi opportunity
func (r *DeFiRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.DeFiOpportunity{}, id).Error
}

// GetActive отримує активні DeFi opportunities
func (r *DeFiRepositoryImpl) GetActive(limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("is_active = ?", true).
		Order("apy DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetAll отримує всі DeFi opportunities
func (r *DeFiRepositoryImpl) GetAll(limit, offset int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Order("apy DESC").
		Limit(limit).
		Offset(offset).
		Find(&opps).Error
	return opps, err
}

// GetByChain отримує DeFi opportunities за chain
func (r *DeFiRepositoryImpl) GetByChain(chain string, limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("chain = ? AND is_active = ?", chain, true).
		Order("apy DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetByProtocol отримує DeFi opportunities за protocol
func (r *DeFiRepositoryImpl) GetByProtocol(protocol string, limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("protocol = ? AND is_active = ?", protocol, true).
		Order("apy DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetByRiskLevel отримує DeFi opportunities за risk level
func (r *DeFiRepositoryImpl) GetByRiskLevel(riskLevel string, limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("risk_level = ? AND is_active = ?", riskLevel, true).
		Order("apy DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetHighAPY отримує DeFi opportunities з високим APY
func (r *DeFiRepositoryImpl) GetHighAPY(minAPY float64, limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("apy >= ? AND is_active = ?", minAPY, true).
		Order("apy DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetHighTVL отримує DeFi opportunities з високим TVL
func (r *DeFiRepositoryImpl) GetHighTVL(minTVL float64, limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("tvl >= ? AND is_active = ?", minTVL, true).
		Order("tvl DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetStablePairs отримує стабільні пари (низький IL)
func (r *DeFiRepositoryImpl) GetStablePairs(maxIL float64, limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("il_risk <= ? AND is_active = ?", maxIL, true).
		Order("apy DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetAudited отримує тільки аудовані протоколи
func (r *DeFiRepositoryImpl) GetAudited(limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("audit_status IN (?, ?) AND is_active = ?", "audited", "verified", true).
		Order("apy DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetByFilters отримує DeFi opportunities за комплексними фільтрами
func (r *DeFiRepositoryImpl) GetByFilters(filters DeFiFilters, limit int) ([]*models.DeFiOpportunity, error) {
	query := r.db.Where("is_active = ?", true)

	// Chains filter
	if len(filters.Chains) > 0 {
		query = query.Where("chain IN ?", filters.Chains)
	}

	// Protocols filter
	if len(filters.Protocols) > 0 {
		query = query.Where("protocol IN ?", filters.Protocols)
	}

	// Risk levels filter
	if len(filters.RiskLevels) > 0 {
		query = query.Where("risk_level IN ?", filters.RiskLevels)
	}

	// APY range
	if filters.MinAPY > 0 {
		query = query.Where("apy >= ?", filters.MinAPY)
	}
	if filters.MaxAPY > 0 {
		query = query.Where("apy <= ?", filters.MaxAPY)
	}

	// TVL range
	if filters.MinTVL > 0 {
		query = query.Where("tvl >= ?", filters.MinTVL)
	}
	if filters.MaxTVL > 0 {
		query = query.Where("tvl <= ?", filters.MaxTVL)
	}

	// IL risk
	if filters.MaxIL > 0 {
		query = query.Where("il_risk <= ?", filters.MaxIL)
	}

	// Deposit range
	if filters.MinDeposit > 0 {
		query = query.Where("min_deposit >= ?", filters.MinDeposit)
	}
	if filters.MaxDeposit > 0 {
		query = query.Where("min_deposit <= ?", filters.MaxDeposit)
	}

	// Only audited
	if filters.OnlyAudited {
		query = query.Where("audit_status IN (?, ?)", "audited", "verified")
	}

	// No lockup
	if filters.NoLockup {
		query = query.Where("lock_period = ?", 0)
	}

	var opps []*models.DeFiOpportunity
	err := query.Order("apy DESC").Limit(limit).Find(&opps).Error
	return opps, err
}

// GetTopByAPY отримує топ за APY
func (r *DeFiRepositoryImpl) GetTopByAPY(limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("is_active = ?", true).
		Order("apy DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// GetTopByTVL отримує топ за TVL
func (r *DeFiRepositoryImpl) GetTopByTVL(limit int) ([]*models.DeFiOpportunity, error) {
	var opps []*models.DeFiOpportunity
	err := r.db.Where("is_active = ?", true).
		Order("tvl DESC").
		Limit(limit).
		Find(&opps).Error
	return opps, err
}

// DeleteOld видаляє старі записи
func (r *DeFiRepositoryImpl) DeleteOld(before time.Time) error {
	return r.db.Where("last_checked < ?", before).
		Delete(&models.DeFiOpportunity{}).Error
}

// MarkInactive позначає opportunities як неактивні
func (r *DeFiRepositoryImpl) MarkInactive(externalIDs []string) error {
	return r.db.Model(&models.DeFiOpportunity{}).
		Where("external_id IN ?", externalIDs).
		Update("is_active", false).Error
}

// CountActive рахує активні opportunities
func (r *DeFiRepositoryImpl) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&models.DeFiOpportunity{}).
		Where("is_active = ?", true).
		Count(&count).Error
	return count, err
}
