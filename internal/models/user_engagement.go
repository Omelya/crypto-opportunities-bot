package models

import "time"

// UserEngagement tracks daily user engagement metrics
type UserEngagement struct {
	BaseModel

	UserID uint      `gorm:"index;not null" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"-"`
	Date   time.Time `gorm:"index;not null" json:"date"`

	// Session metrics
	SessionsCount int `gorm:"default:0" json:"sessions_count"`
	TimeSpent     int `gorm:"default:0" json:"time_spent"` // in seconds

	// Activity metrics
	ActionsCount           int       `gorm:"default:0" json:"actions_count"`
	OpportunitiesViewed    int       `gorm:"default:0" json:"opportunities_viewed"`
	OpportunitiesClicked   int       `gorm:"default:0" json:"opportunities_clicked"`
	OpportunitiesParticipated int    `gorm:"default:0" json:"opportunities_participated"`
	FirstActivityAt        *time.Time `json:"first_activity_at,omitempty"`
	LastActivityAt         *time.Time `json:"last_activity_at,omitempty"`

	// Commands used
	CommandsUsed JSONArray `gorm:"type:jsonb;serializer:json" json:"commands_used,omitempty"`

	// Engagement level
	EngagementLevel string `gorm:"index" json:"engagement_level"` // low, medium, high
}

func (*UserEngagement) TableName() string {
	return "user_engagement"
}

// CalculateEngagementLevel determines user's engagement level for the day
func (ue *UserEngagement) CalculateEngagementLevel() {
	score := 0

	// Sessions contribute to engagement
	if ue.SessionsCount >= 3 {
		score += 3
	} else {
		score += ue.SessionsCount
	}

	// Time spent (every 5 minutes = 1 point, max 5)
	timeScore := ue.TimeSpent / 300 // 300 seconds = 5 minutes
	if timeScore > 5 {
		timeScore = 5
	}
	score += timeScore

	// Actions (every 5 actions = 1 point, max 5)
	actionScore := ue.ActionsCount / 5
	if actionScore > 5 {
		actionScore = 5
	}
	score += actionScore

	// Participation adds significant score
	score += ue.OpportunitiesParticipated * 3

	// Determine level
	if score >= 15 {
		ue.EngagementLevel = "high"
	} else if score >= 7 {
		ue.EngagementLevel = "medium"
	} else {
		ue.EngagementLevel = "low"
	}
}

// GetAverageSessionTime returns average session duration
func (ue *UserEngagement) GetAverageSessionTime() int {
	if ue.SessionsCount == 0 {
		return 0
	}
	return ue.TimeSpent / ue.SessionsCount
}
