package models

import (
	"time"
)

// ClientTrade представляє трейд виконаний Premium клієнтом
type ClientTrade struct {
	BaseModel

	UserID uint  `gorm:"index;not null" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	OpportunityID uint                  `gorm:"index;not null" json:"opportunity_id"`
	Opportunity   *ArbitrageOpportunity `gorm:"foreignKey:OpportunityID" json:"opportunity,omitempty"`

	// Trading pair
	Pair         string `gorm:"index;not null" json:"pair"`          // "BTC/USDT"
	BuyExchange  string `gorm:"index;not null" json:"buy_exchange"`  // "binance"
	SellExchange string `gorm:"index;not null" json:"sell_exchange"` // "bybit"

	// Trade details
	Amount    float64 `gorm:"type:decimal(20,8);not null" json:"amount"`     // Кількість базової валюти
	BuyPrice  float64 `gorm:"type:decimal(20,8);not null" json:"buy_price"`  // Ціна купівлі
	SellPrice float64 `gorm:"type:decimal(20,8);not null" json:"sell_price"` // Ціна продажу

	// Order IDs (для відстеження на біржах)
	BuyOrderID  string `json:"buy_order_id,omitempty"`  // ID ордера на біржі купівлі
	SellOrderID string `json:"sell_order_id,omitempty"` // ID ордера на біржі продажу

	// Profit tracking
	ExpectedProfit      float64 `gorm:"type:decimal(12,2)" json:"expected_profit"`      // Очікуваний прибуток (USD)
	ActualProfit        float64 `gorm:"type:decimal(12,2)" json:"actual_profit"`        // Фактичний прибуток (USD)
	ActualProfitPercent float64 `gorm:"type:decimal(5,2)" json:"actual_profit_percent"` // Фактичний прибуток (%)

	// Status tracking
	Status string `gorm:"index;not null;default:'pending'" json:"status"` // pending, executing, completed, failed
	Error  string `gorm:"type:text" json:"error,omitempty"`               // Опис помилки якщо failed

	// Performance
	ExecutionTimeMs int `json:"execution_time_ms"` // Час виконання трейду (мс)

	// Timing
	CompletedAt *time.Time `json:"completed_at,omitempty"` // Час завершення
}

func (*ClientTrade) TableName() string {
	return "client_trades"
}

// IsSuccessful перевіряє чи трейд був успішним
func (ct *ClientTrade) IsSuccessful() bool {
	return ct.Status == "completed" && ct.ActualProfit > 0
}

// IsFailed перевіряє чи трейд провалився
func (ct *ClientTrade) IsFailed() bool {
	return ct.Status == "failed"
}

// IsPending перевіряє чи трейд ще виконується
func (ct *ClientTrade) IsPending() bool {
	return ct.Status == "pending" || ct.Status == "executing"
}

// ExecutionDuration скільки часу зайняв трейд
func (ct *ClientTrade) ExecutionDuration() time.Duration {
	if ct.CompletedAt == nil {
		return time.Since(ct.CreatedAt)
	}
	return ct.CompletedAt.Sub(ct.CreatedAt)
}

// ProfitDifference різниця між очікуваним та фактичним прибутком
func (ct *ClientTrade) ProfitDifference() float64 {
	return ct.ActualProfit - ct.ExpectedProfit
}

// Trade statuses
const (
	TradeStatusPending   = "pending"   // Очікує виконання
	TradeStatusExecuting = "executing" // Виконується
	TradeStatusCompleted = "completed" // Успішно завершено
	TradeStatusFailed    = "failed"    // Провалено
)
