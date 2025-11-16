package models

import (
	"time"
)

// ArbitrageOpportunity представляє арбітражну можливість між двома біржами
type ArbitrageOpportunity struct {
	BaseModel

	// Trading pair
	Pair          string `gorm:"index;not null" json:"pair"`           // 'BTC/USDT', 'ETH/USDT'
	BaseCurrency  string `gorm:"not null" json:"base_currency"`        // 'BTC', 'ETH'
	QuoteCurrency string `gorm:"not null" json:"quote_currency"`       // 'USDT', 'USD'

	// Buy side
	ExchangeBuy string  `gorm:"index;not null" json:"exchange_buy"`   // 'binance'
	PriceBuy    float64 `gorm:"type:decimal(20,8);not null" json:"price_buy"` // 67450.30
	VolumeBuy   float64 `gorm:"type:decimal(20,8)" json:"volume_buy"` // Available volume

	// Sell side
	ExchangeSell string  `gorm:"index;not null" json:"exchange_sell"`  // 'bybit'
	PriceSell    float64 `gorm:"type:decimal(20,8);not null" json:"price_sell"` // 67850.50
	VolumeSell   float64 `gorm:"type:decimal(20,8)" json:"volume_sell"` // Available volume

	// Profit calculations
	ProfitPercent float64 `gorm:"index;type:decimal(5,2);not null" json:"profit_percent"` // 0.59 (%)
	ProfitUSD     float64 `gorm:"type:decimal(12,2)" json:"profit_usd"` // Estimated on $1000

	// Fees breakdown
	TradingFeeBuy    float64 `gorm:"type:decimal(5,4)" json:"trading_fee_buy"`    // 0.1000 (0.1%)
	TradingFeeSell   float64 `gorm:"type:decimal(5,4)" json:"trading_fee_sell"`   // 0.1000 (0.1%)
	WithdrawalFee    float64 `gorm:"type:decimal(10,8)" json:"withdrawal_fee"`    // 0.0005 BTC
	WithdrawalFeeUSD float64 `gorm:"type:decimal(10,2)" json:"withdrawal_fee_usd"` // $33.72
	TotalFeesPercent float64 `gorm:"type:decimal(5,4)" json:"total_fees_percent"` // 0.2500 (0.25%)

	// Net profit (after fees and slippage)
	NetProfitPercent float64 `gorm:"type:decimal(5,2);not null" json:"net_profit_percent"` // 0.34 (%)
	NetProfitUSD     float64 `gorm:"type:decimal(12,2)" json:"net_profit_usd"` // $3.40 on $1000

	// Market data
	Volume24h     float64 `gorm:"type:decimal(15,2)" json:"volume_24h"`     // 24h volume (liquidity check)
	SpreadPercent float64 `gorm:"type:decimal(5,2)" json:"spread_percent"`  // Price difference %

	// Slippage
	SlippageBuy  float64 `gorm:"type:decimal(5,4)" json:"slippage_buy"`  // Slippage on buy side
	SlippageSell float64 `gorm:"type:decimal(5,4)" json:"slippage_sell"` // Slippage on sell side

	// Trade amounts
	MinTradeAmount    float64 `gorm:"type:decimal(12,2)" json:"min_trade_amount"`    // Minimum $50
	MaxTradeAmount    float64 `gorm:"type:decimal(12,2)" json:"max_trade_amount"`    // Maximum $5000
	RecommendedAmount float64 `gorm:"type:decimal(12,2)" json:"recommended_amount"`  // $500-2000

	// Timing
	DetectedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"detected_at"` // When detected
	ExpiresAt  time.Time `gorm:"index;not null" json:"expires_at"`                      // Valid until (+ 3-5 min)
	IsNotified bool      `gorm:"default:false" json:"is_notified"`                      // Sent to users?

	// Deduplication
	ExternalID string `gorm:"uniqueIndex;not null" json:"external_id"` // MD5 hash
}

func (*ArbitrageOpportunity) TableName() string {
	return "arbitrage_opportunities"
}

// IsActive перевіряє чи можливість ще актуальна
func (a *ArbitrageOpportunity) IsActive() bool {
	return time.Now().Before(a.ExpiresAt)
}

// TimeLeft скільки часу залишилось
func (a *ArbitrageOpportunity) TimeLeft() time.Duration {
	return time.Until(a.ExpiresAt)
}

// IsHighProfit перевіряє чи це висока можливість
func (a *ArbitrageOpportunity) IsHighProfit() bool {
	return a.NetProfitPercent >= 0.5
}

// IsVeryHighProfit перевіряє чи це дуже висока можливість
func (a *ArbitrageOpportunity) IsVeryHighProfit() bool {
	return a.NetProfitPercent >= 1.0
}
