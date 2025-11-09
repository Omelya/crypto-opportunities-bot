package models

type UserPreferences struct {
	BaseModel

	UserID             uint        `gorm:"uniqueIndex;not null" json:"user_id"`
	User               User        `gorm:"foreignKey:UserID" json:"-"`
	OpportunityTypes   StringArray `gorm:"type:jsonb;serializer:json;default:'[]'" json:"opportunity_types"`
	Exchanges          StringArray `gorm:"type:jsonb;serializer:json;default:'[]'" json:"exchanges"`
	MinROI             float64     `gorm:"default:0.3" json:"min_roi"`            // Мінімальний ROI %
	MaxInvestment      int         `gorm:"default:0" json:"max_investment"`       // 0 = без ліміту
	NotifyArbitrage    bool        `gorm:"default:false" json:"notify_arbitrage"` // Premium
	NotifyLaunchpool   bool        `gorm:"default:true" json:"notify_launchpool"`
	NotifyAirdrop      bool        `gorm:"default:true" json:"notify_airdrop"`
	NotifyLearnEarn    bool        `gorm:"default:true" json:"notify_learn_earn"`
	NotifyDeFi         bool        `gorm:"default:false" json:"notify_defi"`   // Premium
	NotifyWhales       bool        `gorm:"default:false" json:"notify_whales"` // Premium
	DailyDigestEnabled bool        `gorm:"default:true" json:"daily_digest_enabled"`
	DailyDigestTime    string      `gorm:"default:'09:00'" json:"daily_digest_time"` // HH:MM format
}

func (*UserPreferences) TableName() string {
	return "user_preferences"
}
