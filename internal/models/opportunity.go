package models

import "time"

const (
	OpportunityTypeLaunchpool = "launchpool"
	OpportunityTypeLaunchpad  = "launchpad"
	OpportunityTypeAirdrop    = "airdrop"
	OpportunityTypeLearnEarn  = "learn_earn"
	OpportunityTypeStaking    = "staking"
	OpportunityTypeArbitrage  = "arbitrage" // Premium
	OpportunityTypeDeFi       = "defi"      // Premium
)

const (
	ExchangeBinance = "binance"
	ExchangeBybit   = "bybit"
	ExchangeOKX     = "okx"
	ExchangeGateIO  = "gateio"
	ExchangeKraken  = "kraken"
)

type Opportunity struct {
	BaseModel

	ExternalID    string     `gorm:"uniqueIndex;not null" json:"external_id"` // MD5 hash для унікальності
	Exchange      string     `gorm:"index;not null" json:"exchange"`
	Type          string     `gorm:"index;not null" json:"type"`
	Title         string     `gorm:"not null" json:"title"`
	Description   string     `gorm:"type:text" json:"description"`
	Reward        string     `json:"reward"`                     // "100 USDT", "5% APR"
	MinInvestment float64    `json:"min_investment"`             // Мінімальна сума
	EstimatedROI  float64    `gorm:"index" json:"estimated_roi"` // % ROI
	PoolSize      float64    `json:"pool_size"`                  // Для airdrop
	Requirements  string     `gorm:"type:text" json:"requirements"`
	Duration      string     `json:"duration"` // "7 days", "30 days"
	StartDate     *time.Time `json:"start_date"`
	EndDate       *time.Time `gorm:"index" json:"end_date"`
	URL           string     `json:"url"`
	ImageURL      string     `json:"image_url,omitempty"`
	IsActive      bool       `gorm:"index;default:true" json:"is_active"`
	IsFeatured    bool       `gorm:"default:false" json:"is_featured"` // Виділені можливості
	Metadata      JSONMap    `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

func (*Opportunity) TableName() string {
	return "opportunities"
}

func (o *Opportunity) IsExpired() bool {
	if o.EndDate == nil {
		return false
	}

	return time.Now().After(*o.EndDate)
}

func (o *Opportunity) DaysLeft() int {
	if o.EndDate == nil {
		return -1 // Необмежено
	}

	diff := o.EndDate.Sub(time.Now())
	days := int(diff.Hours() / 24)

	if days < 0 {
		return 0
	}

	return days
}

func (o *Opportunity) IsHighROI() bool {
	return o.EstimatedROI >= 5.0
}
