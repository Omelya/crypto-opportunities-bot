package models

import "time"

const (
	NotificationStatusPending = "pending"
	NotificationStatusSent    = "sent"
	NotificationStatusFailed  = "failed"
)

const (
	NotificationPriorityLow    = "low"
	NotificationPriorityNormal = "normal"
	NotificationPriorityHigh   = "high"
)

type Notification struct {
	BaseModel

	UserID        uint         `gorm:"index;not null" json:"user_id"`
	User          User         `gorm:"foreignKey:UserID" json:"-"`
	OpportunityID *uint        `gorm:"index" json:"opportunity_id,omitempty"`
	Opportunity   *Opportunity `gorm:"foreignKey:OpportunityID" json:"-"`
	Type          string       `gorm:"index;not null" json:"type"` // opportunity_type або "daily_digest", "system"
	Priority      string       `gorm:"default:'normal'" json:"priority"`
	Status        string       `gorm:"index;default:'pending'" json:"status"`
	Message       string       `gorm:"type:text;not null" json:"message"`
	MessageData   JSONMap      `gorm:"type:jsonb;serializer:json" json:"message_data,omitempty"` // Для форматування
	ScheduledFor  *time.Time   `gorm:"index" json:"scheduled_for,omitempty"`                     // Для відкладених повідомлень
	SentAt        *time.Time   `json:"sent_at,omitempty"`
	ErrorMessage  string       `gorm:"type:text" json:"error_message,omitempty"`
	RetryCount    int          `gorm:"default:0" json:"retry_count"`
	MaxRetries    int          `gorm:"default:3" json:"max_retries"`
}

func (*Notification) TableName() string {
	return "notifications"
}

func (n *Notification) IsPending() bool {
	return n.Status == NotificationStatusPending
}

func (n *Notification) IsSent() bool {
	return n.Status == NotificationStatusSent
}

func (n *Notification) CanRetry() bool {
	return n.Status == NotificationStatusFailed && n.RetryCount < n.MaxRetries
}

func (n *Notification) IsScheduled() bool {
	if n.ScheduledFor == nil {
		return false
	}
	return time.Now().Before(*n.ScheduledFor)
}

func (n *Notification) ShouldSend() bool {
	if n.Status != NotificationStatusPending {
		return false
	}

	if n.ScheduledFor != nil {
		return time.Now().After(*n.ScheduledFor)
	}

	return true
}

func (n *Notification) MarkAsSent() {
	now := time.Now()
	n.Status = NotificationStatusSent
	n.SentAt = &now
}

func (n *Notification) MarkAsFailed(errorMsg string) {
	n.Status = NotificationStatusFailed
	n.ErrorMessage = errorMsg
	n.RetryCount++
}
