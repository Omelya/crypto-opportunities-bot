package models

import (
	"sort"
	"sync"
	"time"
)

// OrderBook представляє стан ордербуку біржі
type OrderBook struct {
	Exchange     string
	Symbol       string
	Bids         []PriceLevel // Від найкращого до гіршого (ціна спадає)
	Asks         []PriceLevel // Від найкращого до гіршого (ціна зростає)
	LastUpdateID int64
	LastUpdate   time.Time
	mu           sync.RWMutex
}

// PriceLevel - рівень ціни в ордербуці
type PriceLevel struct {
	Price    float64 // Ціна
	Quantity float64 // Обсяг на цьому рівні
}

// NewOrderBook створює новий OrderBook
func NewOrderBook(exchange, symbol string) *OrderBook {
	return &OrderBook{
		Exchange:   exchange,
		Symbol:     symbol,
		Bids:       []PriceLevel{},
		Asks:       []PriceLevel{},
		LastUpdate: time.Now(),
	}
}

// GetBestBid найкраща ціна купівлі (найвища bid)
func (ob *OrderBook) GetBestBid() *PriceLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.Bids) == 0 {
		return nil
	}
	return &ob.Bids[0]
}

// GetBestAsk найкраща ціна продажу (найнижча ask)
func (ob *OrderBook) GetBestAsk() *PriceLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.Asks) == 0 {
		return nil
	}
	return &ob.Asks[0]
}

// GetMidPrice середня ціна між bid і ask
func (ob *OrderBook) GetMidPrice() float64 {
	bid := ob.GetBestBid()
	ask := ob.GetBestAsk()

	if bid == nil || ask == nil {
		return 0
	}

	return (bid.Price + ask.Price) / 2
}

// GetSpread спред між bid і ask
func (ob *OrderBook) GetSpread() float64 {
	bid := ob.GetBestBid()
	ask := ob.GetBestAsk()

	if bid == nil || ask == nil {
		return 0
	}

	return ask.Price - bid.Price
}

// GetSpreadPercent спред у відсотках
func (ob *OrderBook) GetSpreadPercent() float64 {
	spread := ob.GetSpread()
	midPrice := ob.GetMidPrice()

	if midPrice == 0 {
		return 0
	}

	return (spread / midPrice) * 100
}

// CalculateSlippage розраховує slippage для заданого обсягу в USD
// side: "buy" або "sell"
// amountUSD: сума в доларах США
func (ob *OrderBook) CalculateSlippage(side string, amountUSD float64) *SlippageResult {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var levels []PriceLevel
	if side == "buy" {
		levels = ob.Asks // Купуємо по ask (продавці)
	} else {
		levels = ob.Bids // Продаємо по bid (покупці)
	}

	if len(levels) == 0 {
		return &SlippageResult{
			Success:               false,
			InsufficientLiquidity: true,
		}
	}

	var totalCost float64
	var totalQuantity float64
	var levelsUsed int

	remainingUSD := amountUSD

	// Проходимось по рівнях ордербука
	for i, level := range levels {
		if remainingUSD <= 0 {
			break
		}

		// Вартість усього обсягу на цьому рівні
		levelCost := level.Price * level.Quantity

		if levelCost >= remainingUSD {
			// Частково використовуємо цей рівень
			qty := remainingUSD / level.Price
			totalCost += remainingUSD
			totalQuantity += qty
			remainingUSD = 0
			levelsUsed = i + 1
			break
		} else {
			// Використовуємо весь рівень
			totalCost += levelCost
			totalQuantity += level.Quantity
			remainingUSD -= levelCost
			levelsUsed = i + 1
		}
	}

	// Якщо залишились гроші - недостатня ліквідність
	if remainingUSD > 0 {
		return &SlippageResult{
			Success:               false,
			InsufficientLiquidity: true,
			AvailableLiquidityUSD: totalCost,
		}
	}

	// Середня ціна виконання
	avgPrice := totalCost / totalQuantity
	bestPrice := levels[0].Price

	// Розрахунок slippage
	slippagePercent := ((avgPrice - bestPrice) / bestPrice) * 100

	// Для продажу slippage негативний (отримуємо гіршу ціну)
	if side == "sell" {
		slippagePercent = -slippagePercent
	}

	return &SlippageResult{
		Success:         true,
		AveragePrice:    avgPrice,
		BestPrice:       bestPrice,
		SlippagePercent: slippagePercent,
		TotalQuantity:   totalQuantity,
		LevelsUsed:      levelsUsed,
		TotalCost:       totalCost,
	}
}

// SlippageResult результат розрахунку slippage
type SlippageResult struct {
	Success               bool
	AveragePrice          float64 // Середня ціна виконання
	BestPrice             float64 // Найкраща ціна (перший рівень)
	SlippagePercent       float64 // Slippage в %
	TotalQuantity         float64 // Скільки монет куплено/продано
	LevelsUsed            int     // Скільки рівнів використано
	TotalCost             float64 // Загальна вартість в USD
	InsufficientLiquidity bool    // Чи достатньо ліквідності
	AvailableLiquidityUSD float64 // Доступна ліквідність (якщо недостатньо)
}

// Update оновлює ордербук (full snapshot)
func (ob *OrderBook) Update(bids, asks []PriceLevel, updateID int64) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.Bids = bids
	ob.Asks = asks
	ob.LastUpdateID = updateID
	ob.LastUpdate = time.Now()
}

// ApplyDelta застосовує delta update (incremental updates)
func (ob *OrderBook) ApplyDelta(bids, asks []PriceLevel, updateID int64) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Apply bid updates
	for _, update := range bids {
		ob.updateLevel(&ob.Bids, update, true)
	}

	// Apply ask updates
	for _, update := range asks {
		ob.updateLevel(&ob.Asks, update, false)
	}

	// Sort after updates
	ob.sortLevels()

	ob.LastUpdateID = updateID
	ob.LastUpdate = time.Now()
}

// updateLevel оновлює окремий рівень ціни
func (ob *OrderBook) updateLevel(levels *[]PriceLevel, update PriceLevel, isBid bool) {
	// Знайти рівень з такою ціною
	for i, level := range *levels {
		if level.Price == update.Price {
			if update.Quantity == 0 {
				// Видалити рівень (quantity = 0 означає видалення)
				*levels = append((*levels)[:i], (*levels)[i+1:]...)
			} else {
				// Оновити quantity
				(*levels)[i].Quantity = update.Quantity
			}
			return
		}
	}

	// Якщо не знайшли і quantity > 0, додати новий рівень
	if update.Quantity > 0 {
		*levels = append(*levels, update)
	}
}

// sortLevels сортує bid і ask рівні
func (ob *OrderBook) sortLevels() {
	// Sort bids: найвища ціна першою (спадання)
	sort.Slice(ob.Bids, func(i, j int) bool {
		return ob.Bids[i].Price > ob.Bids[j].Price
	})

	// Sort asks: найнижча ціна першою (зростання)
	sort.Slice(ob.Asks, func(i, j int) bool {
		return ob.Asks[i].Price < ob.Asks[j].Price
	})
}

// IsStale перевіряє чи ордербук застарілий
func (ob *OrderBook) IsStale(maxAge time.Duration) bool {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return time.Since(ob.LastUpdate) > maxAge
}

// GetDepth повертає кількість рівнів
func (ob *OrderBook) GetDepth() (bidDepth, askDepth int) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return len(ob.Bids), len(ob.Asks)
}

// GetLiquidity розраховує доступну ліквідність в USD
func (ob *OrderBook) GetLiquidity(side string, maxLevels int) float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var levels []PriceLevel
	if side == "buy" {
		levels = ob.Asks
	} else {
		levels = ob.Bids
	}

	if maxLevels > 0 && len(levels) > maxLevels {
		levels = levels[:maxLevels]
	}

	var totalLiquidity float64
	for _, level := range levels {
		totalLiquidity += level.Price * level.Quantity
	}

	return totalLiquidity
}
