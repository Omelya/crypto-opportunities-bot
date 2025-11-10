package models

const (
	ActionTypeViewed       = "viewed"
	ActionTypeClicked      = "clicked"
	ActionTypeParticipated = "participated"
	ActionTypeIgnored      = "ignored"
)

type UserAction struct {
	BaseModel

	UserID        uint         `gorm:"index;not null" json:"user_id"`
	User          User         `gorm:"foreignKey:UserID" json:"-"`
	OpportunityID *uint        `gorm:"index" json:"opportunity_id,omitempty"`
	Opportunity   *Opportunity `gorm:"foreignKey:OpportunityID" json:"-"`
	ActionType    string       `gorm:"index;not null" json:"action_type"`
	Metadata      JSONMap      `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

func (*UserAction) TableName() string {
	return "user_actions"
}
