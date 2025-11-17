package arbitrage

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"log"
	"strings"
	"time"
)

// Detector виявляє арбітражні можливості event-driven
type Detector struct {
	obManager    *OrderBookManager
	calculator   *Calculator
	arbRepo      repository.ArbitrageRepository
	deduplicator *Deduplicator

	// Configuration
	minProfitPercent float64
	minVolume24h     float64
	maxSpreadPercent float64
	maxSlippage      float64

	onOpportunity OpportunityCallback
}

// OpportunityCallback викликається при знаходженні можливості
type OpportunityCallback func(*models.ArbitrageOpportunity)

// NewDetector створює новий Detector
func NewDetector(
	obManager *OrderBookManager,
	calc *Calculator,
	deduplicator *Deduplicator,
	arbRepo repository.ArbitrageRepository,
	minProfitPercent float64,
	minVolume24h float64,
	maxSpreadPercent float64,
	maxSlippage float64,
) *Detector {
	return &Detector{
		obManager:        obManager,
		calculator:       calc,
		deduplicator:     deduplicator,
		arbRepo:          arbRepo,
		minProfitPercent: minProfitPercent,
		minVolume24h:     minVolume24h,
		maxSpreadPercent: maxSpreadPercent,
		maxSlippage:      maxSlippage,
	}
}

// Start запускає detector (підписується на оновлення orderbook)
func (d *Detector) Start() {
	// Підписуємось на оновлення OrderBook
	d.obManager.OnUpdate(func(exchange, symbol string, ob *models.OrderBook) {
		// Кожен раз коли оновлюється orderbook - перевіряємо арбітраж
		go d.checkArbitrage(symbol)
	})

	log.Println("✅ Arbitrage detector started (event-driven)")
}

// checkArbitrage перевіряє арбітражну можливість для символу
func (d *Detector) checkArbitrage(symbol string) {
	// Отримати найкращі ціни з усіх бірж
	bestPrices := d.obManager.GetBestPrices(symbol)
	if bestPrices == nil {
		return
	}

	// Перевірити чи є арбітраж
	if !bestPrices.HasArbitrage() {
		return
	}

	buyExchange := bestPrices.BestAsk.Exchange
	sellExchange := bestPrices.BestBid.Exchange

	if buyExchange == sellExchange {
		return // Та сама біржа
	}

	// Отримати повні orderbook для розрахунку slippage
	buyOB := d.obManager.GetOrderBook(buyExchange, symbol)
	sellOB := d.obManager.GetOrderBook(sellExchange, symbol)

	if buyOB == nil || sellOB == nil {
		return
	}

	// Розрахувати з урахуванням slippage
	opp := d.calculateWithSlippage(symbol, buyExchange, buyOB, sellExchange, sellOB)

	if opp == nil {
		return
	}

	// Фільтрація
	if !d.shouldCreate(opp) {
		return
	}

	// Deduplication
	if d.deduplicator.IsDuplicate(opp.ExternalID) {
		return
	}

	// Зберегти в БД
	if err := d.arbRepo.Create(opp); err != nil {
		log.Printf("❌ Error creating arbitrage: %v", err)
		return
	}

	d.deduplicator.Add(opp.ExternalID)

	log.Printf("🔥 NEW ARBITRAGE: %s | %s→%s | %.2f%% net profit | $%.2f on $1000",
		opp.Pair, opp.ExchangeBuy, opp.ExchangeSell, opp.NetProfitPercent, opp.NetProfitUSD)

	// Callback для створення нотифікацій
	if d.onOpportunity != nil {
		go d.onOpportunity(opp)
	}
}

// calculateWithSlippage розраховує можливість з урахуванням slippage
func (d *Detector) calculateWithSlippage(
	symbol string,
	buyExchange string,
	buyOB *models.OrderBook,
	sellExchange string,
	sellOB *models.OrderBook,
) *models.ArbitrageOpportunity {

	testAmount := 1000.0 // Default test amount in USD

	// Розрахувати slippage для купівлі
	buySlippage := buyOB.CalculateSlippage("buy", testAmount)
	if buySlippage == nil || !buySlippage.Success {
		return nil // Недостатня ліквідність
	}

	// Розрахувати slippage для продажу
	sellSlippage := sellOB.CalculateSlippage("sell", testAmount)
	if sellSlippage == nil || !sellSlippage.Success {
		return nil // Недостатня ліквідність
	}

	// Перевірити slippage limits
	if buySlippage.SlippagePercent > d.maxSlippage ||
		sellSlippage.SlippagePercent > d.maxSlippage {
		return nil // Занадто великий slippage
	}

	// Використовуємо реальні ціни з урахуванням slippage
	buyPrice := buySlippage.AveragePrice
	sellPrice := sellSlippage.AveragePrice

	// Estimate volume (приблизний)
	estimatedVolume := (buySlippage.TotalCost + sellSlippage.TotalCost) / 2

	// Розрахунок через Calculator
	calc, err := d.calculator.CalculateWithSlippage(
		symbol,
		buyExchange,
		buyPrice,
		buySlippage.SlippagePercent,
		sellExchange,
		sellPrice,
		sellSlippage.SlippagePercent,
		estimatedVolume,
	)

	if err != nil {
		return nil
	}

	// Перевірити чи ще прибутково після slippage
	if calc.NetProfit < d.minProfitPercent {
		return nil
	}

	// Створити ArbitrageOpportunity
	now := time.Now()
	ttl := 5 * time.Minute // Default TTL for arbitrage opportunities

	return &models.ArbitrageOpportunity{
		Pair:             symbol,
		BaseCurrency:     calc.BaseCurrency,
		QuoteCurrency:    calc.QuoteCurrency,
		ExchangeBuy:      buyExchange,
		PriceBuy:         buyPrice,
		VolumeBuy:        buySlippage.TotalQuantity,
		ExchangeSell:     sellExchange,
		PriceSell:        sellPrice,
		VolumeSell:       sellSlippage.TotalQuantity,
		ProfitPercent:    calc.GrossProfit,
		ProfitUSD:        calc.ProfitOn1000USD,
		TradingFeeBuy:    calc.BuyFee,
		TradingFeeSell:   calc.SellFee,
		WithdrawalFee:    calc.WithdrawalFee,
		WithdrawalFeeUSD: calc.WithdrawalFeeUSD,
		TotalFeesPercent: calc.TotalFeesPercent,
		SlippageBuy:      buySlippage.SlippagePercent,
		SlippageSell:     sellSlippage.SlippagePercent,
		NetProfitPercent: calc.NetProfit,
		NetProfitUSD:     calc.ProfitOn1000USD,
		Volume24h:        estimatedVolume,
		SpreadPercent:    calc.SpreadPercent,
		MinTradeAmount:   100,
		MaxTradeAmount:   min(buySlippage.AvailableLiquidityUSD, sellSlippage.AvailableLiquidityUSD),
		RecommendedAmount: calc.RecommendedAmount,
		DetectedAt:       now,
		ExpiresAt:        now.Add(ttl),
		IsNotified:       false,
		ExternalID:       GenerateArbitrageID(symbol, buyExchange, sellExchange, now),
	}
}

// shouldCreate фільтрує можливості перед створенням
func (d *Detector) shouldCreate(opp *models.ArbitrageOpportunity) bool {
	// Min profit
	if opp.NetProfitPercent < d.minProfitPercent {
		return false
	}

	// Min volume
	if d.minVolume24h > 0 && opp.Volume24h < d.minVolume24h {
		log.Printf("⚠️ Low volume for %s: $%.0f", opp.Pair, opp.Volume24h)
		return false
	}

	// Max spread (підозріло якщо занадто великий)
	if opp.SpreadPercent > d.maxSpreadPercent {
		log.Printf("⚠️ Suspicious spread for %s: %.2f%%", opp.Pair, opp.SpreadPercent)
		return false
	}

	// Recommended amount > 0
	if opp.RecommendedAmount < 100 {
		return false
	}

	return true
}

// OnOpportunity встановлює callback для нових можливостей
func (d *Detector) OnOpportunity(callback OpportunityCallback) {
	d.onOpportunity = callback
}

// OnArbitrageDetected alias для OnOpportunity (для сумісності)
func (d *Detector) OnArbitrageDetected(callback OpportunityCallback) {
	d.OnOpportunity(callback)
}

// GetStats повертає статистику детектора
func (d *Detector) GetStats() *DetectorStats {
	activeCount, _ := d.arbRepo.CountActive()

	return &DetectorStats{
		ActiveOpportunities: int(activeCount),
		CachedIDs:          d.deduplicator.Size(),
		MinProfit:          d.minProfitPercent,
		MinVolume:          d.minVolume24h,
	}
}

// Stop зупиняє detector
func (d *Detector) Stop() {
	// Cleanup if needed
	log.Println("🛑 Arbitrage detector stopped")
}

// DetectorStats статистика детектора
type DetectorStats struct {
	ActiveOpportunities int
	CachedIDs          int
	MinProfit          float64
	MinVolume          float64
}

// min helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// FormatPair форматує пару для orderbook lookup
func FormatPair(pair string) string {
	return strings.ToUpper(pair)
}
