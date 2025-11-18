package models

import "time"

// DailyStats stores daily platform-wide statistics
type DailyStats struct {
	BaseModel

	Date time.Time `gorm:"uniqueIndex;not null" json:"date"`

	// User metrics
	ActiveUsers   int `gorm:"default:0" json:"active_users"`   // Users who performed any action
	NewUsers      int `gorm:"default:0" json:"new_users"`      // New registrations
	PremiumUsers  int `gorm:"default:0" json:"premium_users"`  // Active premium subscribers
	ReturnedUsers int `gorm:"default:0" json:"returned_users"` // Users who came back after 7+ days

	// Opportunity metrics
	TotalOpportunities  int `gorm:"default:0" json:"total_opportunities"`  // Total active opportunities
	NewOpportunities    int `gorm:"default:0" json:"new_opportunities"`    // New opportunities added
	ViewedOpportunities int `gorm:"default:0" json:"viewed_opportunities"` // Total views
	ClickedOpportunities int `gorm:"default:0" json:"clicked_opportunities"` // Total clicks
	ParticipatedOpportunities int `gorm:"default:0" json:"participated_opportunities"` // Total participations

	// Notification metrics
	NotificationsSent   int `gorm:"default:0" json:"notifications_sent"`
	NotificationsFailed int `gorm:"default:0" json:"notifications_failed"`
	NotificationsOpened int `gorm:"default:0" json:"notifications_opened"`

	// Engagement metrics
	TotalSessions      int `gorm:"default:0" json:"total_sessions"`
	AverageSessionTime int `gorm:"default:0" json:"average_session_time"` // in seconds

	// Conversion metrics
	ConversionRate float64 `gorm:"type:decimal(5,2);default:0" json:"conversion_rate"` // Viewed to participated
	RetentionRate  float64 `gorm:"type:decimal(5,2);default:0" json:"retention_rate"`  // 7-day retention

	// Revenue metrics
	DailyRevenue      float64 `gorm:"type:decimal(10,2);default:0" json:"daily_revenue"`
	NewSubscriptions  int     `gorm:"default:0" json:"new_subscriptions"`
	ChurnedUsers      int     `gorm:"default:0" json:"churned_users"`

	// Top performers
	TopExchange     string `json:"top_exchange,omitempty"`
	TopOpportunity  string `json:"top_opportunity,omitempty"`
	TopOpportunityType string `json:"top_opportunity_type,omitempty"`
}

func (*DailyStats) TableName() string {
	return "daily_stats"
}

// CalculateRates recalculates conversion and retention rates
func (ds *DailyStats) CalculateRates() {
	if ds.ViewedOpportunities > 0 {
		ds.ConversionRate = float64(ds.ParticipatedOpportunities) / float64(ds.ViewedOpportunities) * 100
	}

	if ds.ActiveUsers > 0 && ds.NewUsers > 0 {
		ds.RetentionRate = float64(ds.ActiveUsers-ds.NewUsers) / float64(ds.ActiveUsers) * 100
	}
}

// GetSuccessRate returns notification success rate
func (ds *DailyStats) GetSuccessRate() float64 {
	total := ds.NotificationsSent + ds.NotificationsFailed
	if total == 0 {
		return 0
	}
	return float64(ds.NotificationsSent) / float64(total) * 100
}
