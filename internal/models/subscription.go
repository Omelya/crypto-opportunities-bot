package models

import (
	"time"
)

const (
	SubscriptionStatusPending  = "pending"  // Створено, чекає оплату
	SubscriptionStatusActive   = "active"   // Активна підписка
	SubscriptionStatusCanceled = "canceled" // Скасована користувачем
	SubscriptionStatusExpired  = "expired"  // Закінчилась
	SubscriptionStatusFailed   = "failed"   // Помилка оплати
)

const (
	PaymentProviderMonobank = "monobank"
	PaymentProviderPayPal   = "paypal"
	PaymentProviderCrypto   = "crypto"
)

type Subscription struct {
	BaseModel

	// User relation
	UserID uint `gorm:"index;not null" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"-"`

	// Payment provider
	Provider string `gorm:"index;not null" json:"provider"` // monobank, paypal, crypto

	// Subscription details
	Plan   string `gorm:"not null" json:"plan"` // premium_monthly, premium_weekly, premium_yearly
	Status string `gorm:"index;not null;default:'pending'" json:"status"`

	// Pricing
	Amount   int64  `gorm:"not null" json:"amount"`           // Сума в мінімальних одиницях (копійки)
	Currency string `gorm:"not null;default:'UAH'" json:"currency"` // UAH, USD, EUR

	// Periods
	CurrentPeriodStart time.Time  `gorm:"not null" json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `gorm:"not null" json:"current_period_end"`
	TrialEnd           *time.Time `json:"trial_end,omitempty"`

	// Cancellation
	CancelAtPeriodEnd bool       `gorm:"default:false" json:"cancel_at_period_end"`
	CanceledAt        *time.Time `json:"canceled_at,omitempty"`
	CancelReason      string     `gorm:"type:text" json:"cancel_reason,omitempty"`

	// Monobank specific
	MonobankInvoiceID string `gorm:"index" json:"monobank_invoice_id,omitempty"`
	MonobankWalletID  string `json:"monobank_wallet_id,omitempty"` // Для recurring
	MonobankReference string `gorm:"uniqueIndex" json:"monobank_reference,omitempty"` // Унікальний reference

	// PayPal specific (для майбутнього)
	PayPalSubscriptionID string `gorm:"index" json:"paypal_subscription_id,omitempty"`
	PayPalCustomerID     string `json:"paypal_customer_id,omitempty"`

	// Auto-renewal
	AutoRenew      bool       `gorm:"default:true" json:"auto_renew"`
	NextBillingAt  *time.Time `json:"next_billing_at,omitempty"`
	LastBillingAt  *time.Time `json:"last_billing_at,omitempty"`
	FailedAttempts int        `gorm:"default:0" json:"failed_attempts"`

	// Metadata
	Metadata JSONMap `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

func (*Subscription) TableName() string {
	return "subscriptions"
}

// IsActive перевіряє чи активна підписка
func (s *Subscription) IsActive() bool {
	if s.Status != SubscriptionStatusActive {
		return false
	}

	now := time.Now()
	return now.Before(s.CurrentPeriodEnd) && now.After(s.CurrentPeriodStart)
}

// IsExpired перевіряє чи закінчилась підписка
func (s *Subscription) IsExpired() bool {
	return time.Now().After(s.CurrentPeriodEnd)
}

// IsInTrial перевіряє чи підписка в trial періоді
func (s *Subscription) IsInTrial() bool {
	if s.TrialEnd == nil {
		return false
	}
	return time.Now().Before(*s.TrialEnd)
}

// DaysLeft кількість днів до закінчення підписки
func (s *Subscription) DaysLeft() int {
	if s.IsExpired() {
		return 0
	}

	diff := s.CurrentPeriodEnd.Sub(time.Now())
	days := int(diff.Hours() / 24)

	if days < 0 {
		return 0
	}

	return days
}

// ShouldRenew перевіряє чи потрібно продовжити підписку
func (s *Subscription) ShouldRenew() bool {
	if !s.AutoRenew {
		return false
	}

	if s.Status != SubscriptionStatusActive {
		return false
	}

	if s.CancelAtPeriodEnd {
		return false
	}

	// Спробувати продовжити за 2 дні до закінчення
	return time.Now().After(s.CurrentPeriodEnd.Add(-48 * time.Hour))
}

// MarkAsActive позначає підписку як активну
func (s *Subscription) MarkAsActive(walletID string) {
	s.Status = SubscriptionStatusActive
	s.MonobankWalletID = walletID
	now := time.Now()
	s.LastBillingAt = &now
	s.FailedAttempts = 0

	// Розрахувати наступний період
	s.calculateNextPeriod()
}

// MarkAsCanceled скасовує підписку
func (s *Subscription) MarkAsCanceled(reason string, immediately bool) {
	now := time.Now()
	s.CanceledAt = &now
	s.CancelReason = reason
	s.AutoRenew = false

	if immediately {
		s.Status = SubscriptionStatusCanceled
		s.CurrentPeriodEnd = now
	} else {
		s.CancelAtPeriodEnd = true
	}
}

// MarkAsExpired позначає підписку як закінчену
func (s *Subscription) MarkAsExpired() {
	s.Status = SubscriptionStatusExpired
}

// MarkAsFailed позначає помилку оплати
func (s *Subscription) MarkAsFailed() {
	s.FailedAttempts++
	if s.FailedAttempts >= 3 {
		s.Status = SubscriptionStatusFailed
		s.AutoRenew = false
	}
}

// calculateNextPeriod розраховує наступний період оплати
func (s *Subscription) calculateNextPeriod() {
	// Імпортуємо з monobank package
	durations := map[string]time.Duration{
		"premium_monthly": 30 * 24 * time.Hour,
		"premium_weekly":  7 * 24 * time.Hour,
		"premium_yearly":  365 * 24 * time.Hour,
	}

	duration, ok := durations[s.Plan]
	if !ok {
		duration = 30 * 24 * time.Hour // Default
	}

	nextBilling := s.CurrentPeriodEnd.Add(duration)
	s.NextBillingAt = &nextBilling
}

// Payment історія платежів
type Payment struct {
	BaseModel

	// Relations
	UserID         uint          `gorm:"index;not null" json:"user_id"`
	User           User          `gorm:"foreignKey:UserID" json:"-"`
	SubscriptionID *uint         `gorm:"index" json:"subscription_id,omitempty"`
	Subscription   *Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`

	// Payment details
	Provider      string `gorm:"index;not null" json:"provider"` // monobank, paypal, crypto
	TransactionID string `gorm:"uniqueIndex" json:"transaction_id"` // Invoice ID або Transaction ID
	Reference     string `gorm:"uniqueIndex" json:"reference"` // Наш внутрішній reference

	// Amount
	Amount   int64  `gorm:"not null" json:"amount"`
	Currency string `gorm:"not null;default:'UAH'" json:"currency"`

	// Status
	Status        string     `gorm:"index;not null" json:"status"` // pending, success, failed, refunded
	FailureReason string     `gorm:"type:text" json:"failure_reason,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	RefundedAt    *time.Time `json:"refunded_at,omitempty"`

	// Metadata
	Metadata JSONMap `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

func (*Payment) TableName() string {
	return "payments"
}

const (
	PaymentStatusPending  = "pending"
	PaymentStatusSuccess  = "success"
	PaymentStatusFailed   = "failed"
	PaymentStatusRefunded = "refunded"
)

// MarkAsSuccess позначає платіж як успішний
func (p *Payment) MarkAsSuccess() {
	now := time.Now()
	p.Status = PaymentStatusSuccess
	p.PaidAt = &now
}

// MarkAsFailed позначає платіж як невдалий
func (p *Payment) MarkAsFailed(reason string) {
	p.Status = PaymentStatusFailed
	p.FailureReason = reason
}
