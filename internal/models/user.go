package models

import (
	"time"
)

type User struct {
	BaseModel

	TelegramID            int64      `gorm:"uniqueIndex;not null" json:"telegram_id"`
	Username              string     `gorm:"index" json:"username"`
	FirstName             string     `json:"first_name"`
	LastName              string     `json:"last_name"`
	LanguageCode          string     `gorm:"default:'uk'" json:"language_code"`
	Timezone              string     `gorm:"default:'Europe/Kyiv'" json:"timezone"`
	SubscriptionTier      string     `gorm:"default:'free'" json:"subscription_tier"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at,omitempty"`
	SubscriptionStripeID  string     `json:"subscription_stripe_id,omitempty"`
	CapitalRange          string     `json:"capital_range,omitempty"`
	RiskProfile           string     `json:"risk_profile,omitempty"`
	ReferralCode          string     `gorm:"uniqueIndex" json:"referral_code,omitempty"` // User's personal referral code
	ReferredByID          *uint      `gorm:"index" json:"referred_by_id,omitempty"`      // ID of user who referred this user
	IsActive              bool       `gorm:"default:true" json:"is_active"`
	IsBlocked             bool       `gorm:"default:false" json:"is_blocked"`
	LastActiveAt          *time.Time `json:"last_active_at,omitempty"`
}

func (*User) TableName() string {
	return "users"
}

func (u *User) IsPremium() bool {
	return u.IsSubscriptionActive() && u.SubscriptionTier == "premium"
}

func (u *User) IsSubscriptionActive() bool {
	if u.SubscriptionExpiresAt == nil {
		return false
	}

	return time.Now().Before(*u.SubscriptionExpiresAt)
}
