package models

import "time"

const (
	ReferralRewardTypePremium  = "premium_month"   // 1 month free Premium
	ReferralRewardTypeDiscount = "discount_percent" // Discount percentage
)

const (
	ReferralStatusPending   = "pending"   // Referral registered, not yet activated
	ReferralStatusActive    = "active"    // Friend subscribed/activated
	ReferralStatusCompleted = "completed" // Reward issued
	ReferralStatusExpired   = "expired"   // Referral expired
)

// Referral represents a referral relationship between users
type Referral struct {
	BaseModel

	ReferrerID uint   `gorm:"index;not null" json:"referrer_id"` // User who referred
	ReferredID uint   `gorm:"index;not null" json:"referred_id"` // User who was referred
	Code       string `gorm:"index" json:"code"`                 // Optional: specific referral code used

	Status string `gorm:"index;default:'pending'" json:"status"` // pending, active, completed, expired

	// Reward tracking
	RewardType     string     `json:"reward_type"`      // Type of reward for referrer
	RewardIssued   bool       `gorm:"default:false" json:"reward_issued"`
	RewardIssuedAt *time.Time `json:"reward_issued_at,omitempty"`

	// Friend benefit tracking
	FriendBenefitType   string `json:"friend_benefit_type"`   // Type of benefit for referred friend
	FriendBenefitIssued bool   `gorm:"default:false" json:"friend_benefit_issued"`

	// Activation tracking
	ActivatedAt *time.Time `json:"activated_at,omitempty"` // When friend became premium/active
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`   // Referral expiration

	Metadata JSONMap `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

func (*Referral) TableName() string {
	return "referrals"
}

func (r *Referral) IsActive() bool {
	return r.Status == ReferralStatusActive
}

func (r *Referral) IsExpired() bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*r.ExpiresAt)
}

// ReferralCode represents custom referral codes for influencers/partners
type ReferralCode struct {
	BaseModel

	Code        string `gorm:"uniqueIndex;not null" json:"code"` // Unique code (e.g., "CRYPTO2025")
	OwnerID     uint   `gorm:"index;not null" json:"owner_id"`   // User who owns this code
	Description string `gorm:"type:text" json:"description"`     // Description of the code

	// Usage limits
	MaxUses      int  `gorm:"default:0" json:"max_uses"`      // 0 = unlimited
	CurrentUses  int  `gorm:"default:0" json:"current_uses"`  // How many times used
	IsActive     bool `gorm:"default:true" json:"is_active"`  // Can be used
	IsPublic     bool `gorm:"default:false" json:"is_public"` // Can be shared publicly

	// Rewards
	ReferrerRewardType string `json:"referrer_reward_type"` // What referrer gets
	ReferrerRewardValue int   `json:"referrer_reward_value"` // Value (days, percentage, etc.)

	FriendBenefitType  string `json:"friend_benefit_type"`  // What friend gets
	FriendBenefitValue int    `json:"friend_benefit_value"` // Value (days, percentage, etc.)

	// Validity
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`

	Metadata JSONMap `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

func (*ReferralCode) TableName() string {
	return "referral_codes"
}

func (rc *ReferralCode) CanUse() bool {
	if !rc.IsActive {
		return false
	}

	// Check usage limit
	if rc.MaxUses > 0 && rc.CurrentUses >= rc.MaxUses {
		return false
	}

	// Check validity period
	now := time.Now()

	if rc.ValidFrom != nil && now.Before(*rc.ValidFrom) {
		return false
	}

	if rc.ValidTo != nil && now.After(*rc.ValidTo) {
		return false
	}

	return true
}

func (rc *ReferralCode) IncrementUses() {
	rc.CurrentUses++
}

// ReferralReward represents rewards issued through referral system
type ReferralReward struct {
	BaseModel

	UserID     uint   `gorm:"index;not null" json:"user_id"`     // User receiving reward
	ReferralID uint   `gorm:"index" json:"referral_id"`          // Associated referral
	Type       string `gorm:"index;not null" json:"type"`        // premium_month, discount_percent
	Value      int    `json:"value"`                             // Value of reward (days, percentage)
	Status     string `gorm:"index;default:'pending'" json:"status"` // pending, issued, claimed, expired

	IssuedAt   *time.Time `json:"issued_at,omitempty"`
	ClaimedAt  *time.Time `json:"claimed_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`

	Metadata JSONMap `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

func (*ReferralReward) TableName() string {
	return "referral_rewards"
}

func (rr *ReferralReward) IsClaimed() bool {
	return rr.Status == "claimed" && rr.ClaimedAt != nil
}

func (rr *ReferralReward) IsExpired() bool {
	if rr.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*rr.ExpiresAt)
}

// ReferralStats represents aggregated referral statistics for a user
type ReferralStats struct {
	UserID uint `json:"user_id"`

	TotalReferrals    int `json:"total_referrals"`    // Total referrals made
	ActiveReferrals   int `json:"active_referrals"`   // Currently active referrals
	CompletedReferrals int `json:"completed_referrals"` // Completed (rewarded) referrals

	TotalRewardsEarned int `json:"total_rewards_earned"` // Total rewards received
	PendingRewards     int `json:"pending_rewards"`      // Pending rewards

	CustomCode        string `json:"custom_code,omitempty"`        // User's custom referral code if any
	CustomCodeUses    int    `json:"custom_code_uses,omitempty"`   // Uses of custom code
}
