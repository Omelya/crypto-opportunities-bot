package models

// OpportunityStats stores statistics for each opportunity
type OpportunityStats struct {
	BaseModel

	OpportunityID uint        `gorm:"uniqueIndex;not null" json:"opportunity_id"`
	Opportunity   Opportunity `gorm:"foreignKey:OpportunityID" json:"-"`

	// View metrics
	TotalViews  int `gorm:"default:0" json:"total_views"`
	UniqueViews int `gorm:"default:0" json:"unique_views"` // Unique users who viewed

	// Click metrics
	TotalClicks  int `gorm:"default:0" json:"total_clicks"`
	UniqueClicks int `gorm:"default:0" json:"unique_clicks"` // Unique users who clicked

	// Participation metrics
	TotalParticipations  int `gorm:"default:0" json:"total_participations"`
	UniqueParticipations int `gorm:"default:0" json:"unique_participations"` // Unique users who participated

	// Ignore metrics
	TotalIgnores  int `gorm:"default:0" json:"total_ignores"`
	UniqueIgnores int `gorm:"default:0" json:"unique_ignores"`

	// Conversion metrics
	ViewToClickRate        float64 `gorm:"type:decimal(5,2);default:0" json:"view_to_click_rate"`
	ClickToParticipateRate float64 `gorm:"type:decimal(5,2);default:0" json:"click_to_participate_rate"`
	OverallConversionRate  float64 `gorm:"type:decimal(5,2);default:0" json:"overall_conversion_rate"`

	// Time metrics (in seconds)
	AvgTimeToClick       int `gorm:"default:0" json:"avg_time_to_click"`
	AvgTimeToParticipate int `gorm:"default:0" json:"avg_time_to_participate"`

	// Demographic breakdown
	FreeUserViews    int `gorm:"default:0" json:"free_user_views"`
	PremiumUserViews int `gorm:"default:0" json:"premium_user_views"`

	// Performance score (0-100)
	PerformanceScore float64 `gorm:"type:decimal(5,2);default:0" json:"performance_score"`
}

func (*OpportunityStats) TableName() string {
	return "opportunity_stats"
}

// CalculateConversionRates recalculates all conversion rates
func (os *OpportunityStats) CalculateConversionRates() {
	if os.UniqueViews > 0 {
		os.ViewToClickRate = float64(os.UniqueClicks) / float64(os.UniqueViews) * 100
		os.OverallConversionRate = float64(os.UniqueParticipations) / float64(os.UniqueViews) * 100
	}

	if os.UniqueClicks > 0 {
		os.ClickToParticipateRate = float64(os.UniqueParticipations) / float64(os.UniqueClicks) * 100
	}
}

// CalculatePerformanceScore calculates performance based on engagement and conversion
func (os *OpportunityStats) CalculatePerformanceScore() {
	// Weighted score:
	// 30% - view to click rate
	// 40% - overall conversion rate
	// 20% - total unique views
	// 10% - click to participate rate

	viewScore := os.ViewToClickRate * 0.3
	conversionScore := os.OverallConversionRate * 0.4
	participateScore := os.ClickToParticipateRate * 0.1

	// Normalize unique views (cap at 1000 views = 100%)
	viewCount := float64(os.UniqueViews)
	if viewCount > 1000 {
		viewCount = 1000
	}
	viewCountScore := (viewCount / 1000) * 20

	os.PerformanceScore = viewScore + conversionScore + participateScore + viewCountScore

	// Cap at 100
	if os.PerformanceScore > 100 {
		os.PerformanceScore = 100
	}
}

// IsPopular determines if opportunity is popular (>100 unique views)
func (os *OpportunityStats) IsPopular() bool {
	return os.UniqueViews >= 100
}

// IsHighPerforming determines if opportunity has high conversion
func (os *OpportunityStats) IsHighPerforming() bool {
	return os.OverallConversionRate >= 10.0
}
