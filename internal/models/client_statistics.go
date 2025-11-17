package models

import (
	"time"
)

// ClientStatistics представляє статистику торгівлі користувача
type ClientStatistics struct {
	BaseModel

	UserID uint  `gorm:"uniqueIndex;not null" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Trade counters
	TotalTrades      int `gorm:"default:0" json:"total_trades"`
	SuccessfulTrades int `gorm:"default:0" json:"successful_trades"`
	FailedTrades     int `gorm:"default:0" json:"failed_trades"`

	// Profit/Loss
	TotalProfit float64 `gorm:"type:decimal(12,2);default:0" json:"total_profit"` // Сума всіх прибуткових трейдів
	TotalLoss   float64 `gorm:"type:decimal(12,2);default:0" json:"total_loss"`   // Сума всіх збиткових трейдів
	NetProfit   float64 `gorm:"type:decimal(12,2);default:0" json:"net_profit"`   // Чистий прибуток (total_profit - total_loss)

	// Trade performance
	BestTrade  float64 `gorm:"type:decimal(12,2);default:0" json:"best_trade"`  // Найкращий трейд (USD)
	WorstTrade float64 `gorm:"type:decimal(12,2);default:0" json:"worst_trade"` // Найгірший трейд (USD)
	AvgProfit  float64 `gorm:"type:decimal(12,2);default:0" json:"avg_profit"`  // Середній прибуток на трейд

	// Win rate
	WinRate float64 `gorm:"type:decimal(5,2);default:0" json:"win_rate"` // Відсоток успішних трейдів (0-100)

	// Volume
	TotalVolume float64 `gorm:"type:decimal(15,2);default:0" json:"total_volume"` // Загальний обсяг торгівлі (USD)

	// Timing
	LastTradeAt  *time.Time `json:"last_trade_at,omitempty"`
	LastUpdateAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"last_update_at"`
}

func (*ClientStatistics) TableName() string {
	return "client_statistics"
}

// CalculateWinRate розраховує win rate
func (cs *ClientStatistics) CalculateWinRate() float64 {
	if cs.TotalTrades == 0 {
		return 0
	}
	return (float64(cs.SuccessfulTrades) / float64(cs.TotalTrades)) * 100
}

// CalculateAvgProfit розраховує середній прибуток
func (cs *ClientStatistics) CalculateAvgProfit() float64 {
	if cs.TotalTrades == 0 {
		return 0
	}
	return cs.NetProfit / float64(cs.TotalTrades)
}

// UpdateFromTrade оновлює статистику на основі нового трейду
func (cs *ClientStatistics) UpdateFromTrade(trade *ClientTrade) {
	now := time.Now()

	// Increment counters
	cs.TotalTrades++
	if trade.IsSuccessful() {
		cs.SuccessfulTrades++
		cs.TotalProfit += trade.ActualProfit
	} else if trade.IsFailed() {
		cs.FailedTrades++
		if trade.ActualProfit < 0 {
			cs.TotalLoss += -trade.ActualProfit // Зберігаємо як позитивне число
		}
	}

	// Update net profit
	cs.NetProfit = cs.TotalProfit - cs.TotalLoss

	// Update best/worst
	if trade.ActualProfit > cs.BestTrade {
		cs.BestTrade = trade.ActualProfit
	}
	if trade.ActualProfit < cs.WorstTrade || cs.WorstTrade == 0 {
		cs.WorstTrade = trade.ActualProfit
	}

	// Update average
	cs.AvgProfit = cs.CalculateAvgProfit()

	// Update win rate
	cs.WinRate = cs.CalculateWinRate()

	// Update volume (buy + sell)
	tradeVolume := trade.Amount * ((trade.BuyPrice + trade.SellPrice) / 2)
	cs.TotalVolume += tradeVolume

	// Update timing
	cs.LastTradeAt = &now
	cs.LastUpdateAt = now
}

// IsActive перевіряє чи користувач активно торгує
func (cs *ClientStatistics) IsActive() bool {
	if cs.LastTradeAt == nil {
		return false
	}
	// Активний якщо останній трейд був менше 24 годин тому
	return time.Since(*cs.LastTradeAt) < 24*time.Hour
}

// ProfitFactor співвідношення прибутку до збитку
func (cs *ClientStatistics) ProfitFactor() float64 {
	if cs.TotalLoss == 0 {
		if cs.TotalProfit > 0 {
			return 999.99 // Максимальне значення якщо немає збитків
		}
		return 0
	}
	return cs.TotalProfit / cs.TotalLoss
}

// ROI повертає ROI у відсотках (на основі volume)
func (cs *ClientStatistics) ROI() float64 {
	if cs.TotalVolume == 0 {
		return 0
	}
	return (cs.NetProfit / cs.TotalVolume) * 100
}
