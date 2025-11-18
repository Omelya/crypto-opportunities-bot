package models

import "time"

// UserAnalytics stores aggregated user activity metrics
type UserAnalytics struct {
	BaseModel

	UserID uint `gorm:"uniqueIndex;not null" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"-"`

	// Activity metrics
	ViewedOpportunities      int `gorm:"default:0" json:"viewed_opportunities"`
	ClickedOpportunities     int `gorm:"default:0" json:"clicked_opportunities"`
	ParticipatedOpportunities int `gorm:"default:0" json:"participated_opportunities"`
	IgnoredOpportunities     int `gorm:"default:0" json:"ignored_opportunities"`

	// Engagement metrics
	TotalSessions         int       `gorm:"default:0" json:"total_sessions"`
	TotalTimeSpent        int       `gorm:"default:0" json:"total_time_spent"` // in seconds
	AverageSessionTime    int       `gorm:"default:0" json:"average_session_time"` // in seconds
	LastActivityAt        *time.Time `json:"last_activity_at,omitempty"`
	DaysSinceRegistration int       `gorm:"default:0" json:"days_since_registration"`

	// Conversion metrics
	ViewToClickRate        float64 `gorm:"type:decimal(5,2);default:0" json:"view_to_click_rate"`
	ClickToParticipateRate float64 `gorm:"type:decimal(5,2);default:0" json:"click_to_participate_rate"`
	OverallConversionRate  float64 `gorm:"type:decimal(5,2);default:0" json:"overall_conversion_rate"`

	// Preferences insights
	FavoriteExchanges JSONArray `gorm:"type:jsonb;serializer:json" json:"favorite_exchanges,omitempty"`
	FavoriteTypes     JSONArray `gorm:"type:jsonb;serializer:json" json:"favorite_types,omitempty"`

	// Notification metrics
	NotificationsReceived int `gorm:"default:0" json:"notifications_received"`
	NotificationsOpened   int `gorm:"default:0" json:"notifications_opened"`

	// Revenue metrics (for premium users)
	TotalRevenue      float64    `gorm:"type:decimal(10,2);default:0" json:"total_revenue"`
	LastPaymentAt     *time.Time `json:"last_payment_at,omitempty"`
	SubscriptionDays  int        `gorm:"default:0" json:"subscription_days"`
}

func (*UserAnalytics) TableName() string {
	return "user_analytics"
}

// CalculateConversionRates recalculates all conversion rates
func (ua *UserAnalytics) CalculateConversionRates() {
	if ua.ViewedOpportunities > 0 {
		ua.ViewToClickRate = float64(ua.ClickedOpportunities) / float64(ua.ViewedOpportunities) * 100
		ua.OverallConversionRate = float64(ua.ParticipatedOpportunities) / float64(ua.ViewedOpportunities) * 100
	}

	if ua.ClickedOpportunities > 0 {
		ua.ClickToParticipateRate = float64(ua.ParticipatedOpportunities) / float64(ua.ClickedOpportunities) * 100
	}

	if ua.TotalSessions > 0 && ua.TotalTimeSpent > 0 {
		ua.AverageSessionTime = ua.TotalTimeSpent / ua.TotalSessions
	}
}

// IsActiveUser determines if user is active (activity in last 7 days)
func (ua *UserAnalytics) IsActiveUser() bool {
	if ua.LastActivityAt == nil {
		return false
	}
	return time.Since(*ua.LastActivityAt) <= 7*24*time.Hour
}

// IsEngaged determines if user is highly engaged
func (ua *UserAnalytics) IsEngaged() bool {
	return ua.TotalSessions >= 5 && ua.ParticipatedOpportunities >= 3
}
