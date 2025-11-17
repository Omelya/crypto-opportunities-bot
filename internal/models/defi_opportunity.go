package models

import (
	"crypto/md5"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// DeFiOpportunity представляє можливість в DeFi протоколі
type DeFiOpportunity struct {
	BaseModel

	// Identification
	ExternalID string `gorm:"uniqueIndex;size:32;not null"` // MD5: protocol:chain:poolID
	Protocol   string `gorm:"index;size:50;not null"`       // uniswap-v3, pancakeswap, aave-v3, etc.
	Chain      string `gorm:"index;size:50;not null"`       // ethereum, bsc, polygon, arbitrum
	PoolID     string `gorm:"size:100;not null"`            // Protocol-specific pool ID

	// Pool Details
	PoolName string `gorm:"size:200"` // e.g., "USDC-ETH 0.3%"
	Token0   string `gorm:"size:20"`  // USDC
	Token1   string `gorm:"size:20"`  // ETH (empty for single-asset pools)
	PoolType string `gorm:"size:50"`  // liquidity, lending, staking, vault

	// Profitability Metrics
	APY         float64 `gorm:"index"` // Annual Percentage Yield (%)
	APR         float64 // Annual Percentage Rate (%)
	APYBase     float64 // Base APY (from fees/interest)
	APYReward   float64 // Reward APY (from token emissions)
	DailyReturn float64 // Daily return (%)

	// Liquidity & Volume Metrics
	TVL        float64 `gorm:"index"` // Total Value Locked (USD)
	Volume24h  float64 // 24h volume (USD)
	Volume7d   float64 // 7d volume (USD)
	VolumeAPR  float64 // Volume-based APR

	// Risk Metrics
	RiskLevel   string  `gorm:"index;size:20"` // low, medium, high
	ILRisk      float64 // Impermanent Loss risk (%), 0 for stable pairs
	ILRisk7d    float64 // 7-day IL (%)
	AuditStatus string  `gorm:"size:50"` // audited, unaudited, verified, unknown

	// Requirements
	MinDeposit float64 // Minimum deposit (USD)
	LockPeriod int     // Lock period (days), 0 = no lock

	// Rewards
	RewardTokens pq.StringArray `gorm:"type:text[]"` // Array of reward token symbols

	// URLs
	PoolURL     string `gorm:"size:500"`
	ProtocolURL string `gorm:"size:500"`

	// Metadata
	PoolMeta JSONMap `gorm:"type:jsonb"` // Additional metadata

	// Status
	IsActive    bool      `gorm:"index;default:true"`
	LastChecked time.Time `gorm:"index"`
}

// TableName override
func (DeFiOpportunity) TableName() string {
	return "defi_opportunities"
}

// GenerateExternalID генерує унікальний ID для DeFi opportunity
func GenerateDeFiExternalID(protocol, chain, poolID string) string {
	data := fmt.Sprintf("%s:%s:%s", protocol, chain, poolID)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// IsHighAPY перевіряє чи APY вище порогу
func (d *DeFiOpportunity) IsHighAPY(threshold float64) bool {
	return d.APY >= threshold
}

// IsLowRisk перевіряє чи це low-risk opportunity
func (d *DeFiOpportunity) IsLowRisk() bool {
	return d.RiskLevel == "low"
}

// IsMediumRisk перевіряє чи це medium-risk opportunity
func (d *DeFiOpportunity) IsMediumRisk() bool {
	return d.RiskLevel == "medium"
}

// IsHighRisk перевіряє чи це high-risk opportunity
func (d *DeFiOpportunity) IsHighRisk() bool {
	return d.RiskLevel == "high"
}

// HasLockPeriod перевіряє чи є lock period
func (d *DeFiOpportunity) HasLockPeriod() bool {
	return d.LockPeriod > 0
}

// IsStable перевіряє чи це стабільна пара (низький IL)
func (d *DeFiOpportunity) IsStable() bool {
	return d.ILRisk < 2.0 // Less than 2% IL risk
}

// IsAudited перевіряє чи протокол аудований
func (d *DeFiOpportunity) IsAudited() bool {
	return d.AuditStatus == "audited" || d.AuditStatus == "verified"
}

// DailyReturnUSD розраховує щоденний return в USD для заданої суми
func (d *DeFiOpportunity) DailyReturnUSD(amount float64) float64 {
	return amount * (d.DailyReturn / 100)
}

// MonthlyReturnUSD розраховує місячний return в USD
func (d *DeFiOpportunity) MonthlyReturnUSD(amount float64) float64 {
	return amount * (d.APY / 100 / 12)
}

// YearlyReturnUSD розраховує річний return в USD
func (d *DeFiOpportunity) YearlyReturnUSD(amount float64) float64 {
	return amount * (d.APY / 100)
}

// CalculateRiskScore розраховує загальний risk score (0-100)
func (d *DeFiOpportunity) CalculateRiskScore() float64 {
	score := 0.0

	// Risk level weight (40 points)
	switch d.RiskLevel {
	case "low":
		score += 0
	case "medium":
		score += 20
	case "high":
		score += 40
	}

	// IL risk weight (30 points)
	if d.ILRisk > 20 {
		score += 30
	} else if d.ILRisk > 10 {
		score += 20
	} else if d.ILRisk > 5 {
		score += 10
	}

	// TVL weight (15 points) - lower TVL = higher risk
	if d.TVL < 100000 {
		score += 15
	} else if d.TVL < 1000000 {
		score += 10
	} else if d.TVL < 10000000 {
		score += 5
	}

	// Audit weight (15 points)
	if !d.IsAudited() {
		score += 15
	}

	return score
}

// GetDisplayName повертає відображуване ім'я
func (d *DeFiOpportunity) GetDisplayName() string {
	if d.PoolName != "" {
		return d.PoolName
	}
	if d.Token1 != "" {
		return fmt.Sprintf("%s-%s", d.Token0, d.Token1)
	}
	return d.Token0
}

// IsStale перевіряє чи дані застаріли
func (d *DeFiOpportunity) IsStale(maxAge time.Duration) bool {
	return time.Since(d.LastChecked) > maxAge
}
