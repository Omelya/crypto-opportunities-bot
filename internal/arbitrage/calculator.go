package arbitrage

import (
	"fmt"
	"math"
	"strings"
)

// Calculator розраховує арбітражні можливості з урахуванням fees та slippage
type Calculator struct {
	feeTable *FeeTable
}

// FeeTable таблиця комісій для різних бірж
type FeeTable struct {
	TradingFees    map[string]float64              // exchange -> fee %
	WithdrawalFees map[string]map[string]float64   // exchange -> currency -> fee
}

// NewCalculator створює новий Calculator
func NewCalculator() *Calculator {
	return &Calculator{
		feeTable: &FeeTable{
			TradingFees: map[string]float64{
				"binance": 0.1,  // 0.1% (maker/taker average with BNB discount)
				"bybit":   0.1,  // 0.1%
				"okx":     0.08, // 0.08%
				"gateio":  0.15, // 0.15%
				"kraken":  0.16, // 0.16%
			},
			WithdrawalFees: map[string]map[string]float64{
				"binance": {
					"BTC":  0.0005,
					"ETH":  0.005,
					"BNB":  0.01,
					"SOL":  0.01,
					"XRP":  0.25,
					"ADA":  1.0,
					"AVAX": 0.01,
					"DOT":  0.1,
					"MATIC": 0.1,
					"LINK": 0.2,
					"USDT": 1.0,
					"USDC": 1.0,
				},
				"bybit": {
					"BTC":  0.0005,
					"ETH":  0.005,
					"SOL":  0.01,
					"XRP":  0.25,
					"ADA":  1.0,
					"USDT": 1.0,
				},
				"okx": {
					"BTC":  0.0004,
					"ETH":  0.004,
					"SOL":  0.01,
					"USDT": 0.8,
				},
			},
		},
	}
}

// ArbitrageCalculation результат розрахунку арбітражу
type ArbitrageCalculation struct {
	Pair          string
	BaseCurrency  string
	QuoteCurrency string

	BuyExchange string
	BuyPrice    float64
	BuyFee      float64 // %

	SellExchange string
	SellPrice    float64
	SellFee      float64 // %

	WithdrawalFee    float64 // в базовій валюті (BTC, ETH)
	WithdrawalFeeUSD float64 // в доларах

	GrossProfit      float64 // % без fees
	TotalFeesPercent float64 // % всіх комісій
	NetProfit        float64 // % після всіх комісій

	ProfitOn1000USD   float64 // прибуток на $1000
	RecommendedAmount float64 // рекомендована сума

	Volume24h     float64
	SpreadPercent float64
}

// Calculate розраховує арбітражну можливість
func (c *Calculator) Calculate(
	pair string,
	buyExchange string,
	buyPrice float64,
	sellExchange string,
	sellPrice float64,
	volume24h float64,
) (*ArbitrageCalculation, error) {

	// Валідація
	if buyPrice <= 0 || sellPrice <= 0 {
		return nil, fmt.Errorf("invalid prices")
	}

	if buyPrice >= sellPrice {
		return nil, fmt.Errorf("no arbitrage opportunity: buy price >= sell price")
	}

	// Parse pair
	baseCurrency, quoteCurrency := parsePair(pair)
	if baseCurrency == "" {
		return nil, fmt.Errorf("invalid pair format")
	}

	// Gross profit
	grossProfit := ((sellPrice - buyPrice) / buyPrice) * 100

	// Trading fees
	buyFee := c.getTradingFee(buyExchange)
	sellFee := c.getTradingFee(sellExchange)

	// Withdrawal fee
	withdrawalFee := c.getWithdrawalFee(buyExchange, baseCurrency)
	withdrawalFeeUSD := withdrawalFee * buyPrice

	// Total fees в % (approximation на $1000)
	// Buy: 0.1%, Sell: 0.1%, Withdrawal: ~$X на $1000
	totalFeesPercent := buyFee + sellFee + ((withdrawalFeeUSD / 1000) * 100)

	// Net profit
	netProfit := grossProfit - totalFeesPercent

	// Profit на $1000
	profitOn1000 := (netProfit / 100) * 1000

	// Recommended amount (залежить від ліквідності та profit)
	recommendedAmount := c.calculateRecommendedAmount(volume24h, netProfit)

	// Spread
	midPrice := (buyPrice + sellPrice) / 2
	spreadPercent := ((sellPrice - buyPrice) / midPrice) * 100

	return &ArbitrageCalculation{
		Pair:              pair,
		BaseCurrency:      baseCurrency,
		QuoteCurrency:     quoteCurrency,
		BuyExchange:       buyExchange,
		BuyPrice:          buyPrice,
		BuyFee:            buyFee,
		SellExchange:      sellExchange,
		SellPrice:         sellPrice,
		SellFee:           sellFee,
		WithdrawalFee:     withdrawalFee,
		WithdrawalFeeUSD:  withdrawalFeeUSD,
		GrossProfit:       grossProfit,
		TotalFeesPercent:  totalFeesPercent,
		NetProfit:         netProfit,
		ProfitOn1000USD:   profitOn1000,
		RecommendedAmount: recommendedAmount,
		Volume24h:         volume24h,
		SpreadPercent:     spreadPercent,
	}, nil
}

// CalculateWithSlippage розраховує з урахуванням slippage з orderbook
func (c *Calculator) CalculateWithSlippage(
	pair string,
	buyExchange string,
	buyPrice float64,
	buySlippage float64, // %
	sellExchange string,
	sellPrice float64,
	sellSlippage float64, // %
	volume24h float64,
) (*ArbitrageCalculation, error) {

	// Базовий розрахунок
	calc, err := c.Calculate(pair, buyExchange, buyPrice, sellExchange, sellPrice, volume24h)
	if err != nil {
		return nil, err
	}

	// Додати slippage до fees
	totalSlippage := math.Abs(buySlippage) + math.Abs(sellSlippage)
	calc.TotalFeesPercent += totalSlippage
	calc.NetProfit -= totalSlippage
	calc.ProfitOn1000USD = (calc.NetProfit / 100) * 1000

	return calc, nil
}

// getTradingFee отримує комісію для біржі
func (c *Calculator) getTradingFee(exchange string) float64 {
	if fee, ok := c.feeTable.TradingFees[exchange]; ok {
		return fee
	}
	return 0.2 // Default 0.2% якщо біржа невідома
}

// getWithdrawalFee отримує комісію виведення
func (c *Calculator) getWithdrawalFee(exchange, currency string) float64 {
	if exchFees, ok := c.feeTable.WithdrawalFees[exchange]; ok {
		if fee, ok := exchFees[currency]; ok {
			return fee
		}
	}

	// Default fees якщо невідомо
	defaults := map[string]float64{
		"BTC":  0.0005,
		"ETH":  0.005,
		"BNB":  0.01,
		"SOL":  0.01,
		"USDT": 1.0,
	}

	if fee, ok := defaults[currency]; ok {
		return fee
	}

	return 0
}

// calculateRecommendedAmount розраховує безпечну суму для торгівлі
func (c *Calculator) calculateRecommendedAmount(volume24h, netProfit float64) float64 {
	// Базові правила:
	// - Не більше 0.1% від 24h volume (щоб не рухати ринок)
	// - Мінімум $100
	// - Максимум $5000 (безпечна межа)

	maxSafe := math.Min(volume24h*0.001, 5000) // 0.1% від volume або $5000
	minSafe := 100.0

	if maxSafe < minSafe {
		return 0 // Недостатня ліквідність
	}

	// Якщо profit високий - можна більше
	if netProfit >= 1.0 {
		return math.Min(maxSafe, 2000)
	} else if netProfit >= 0.5 {
		return math.Min(maxSafe, 1000)
	} else {
		return math.Min(maxSafe, 500)
	}
}

// parsePair розбирає пару "BTC/USDT" на base та quote
func parsePair(pair string) (string, string) {
	parts := strings.Split(pair, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// UpdateTradingFee оновлює комісію для біржі
func (c *Calculator) UpdateTradingFee(exchange string, fee float64) {
	c.feeTable.TradingFees[exchange] = fee
}

// UpdateWithdrawalFee оновлює комісію виведення
func (c *Calculator) UpdateWithdrawalFee(exchange, currency string, fee float64) {
	if c.feeTable.WithdrawalFees[exchange] == nil {
		c.feeTable.WithdrawalFees[exchange] = make(map[string]float64)
	}
	c.feeTable.WithdrawalFees[exchange][currency] = fee
}

// GetTradingFee отримує поточну комісію біржі
func (c *Calculator) GetTradingFee(exchange string) float64 {
	return c.getTradingFee(exchange)
}

// GetWithdrawalFee отримує поточну комісію виведення
func (c *Calculator) GetWithdrawalFee(exchange, currency string) float64 {
	return c.getWithdrawalFee(exchange, currency)
}
