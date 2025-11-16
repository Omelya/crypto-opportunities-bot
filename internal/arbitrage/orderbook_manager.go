package arbitrage

import (
	"crypto-opportunities-bot/internal/arbitrage/websocket"
	"crypto-opportunities-bot/internal/models"
	"log"
	"sync"
	"time"
)

// OrderBookManager централізовано управляє orderbook'ами з усіх бірж
type OrderBookManager struct {
	wsManagers map[string]websocket.Manager                     // exchange -> WebSocket Manager
	orderbooks map[string]map[string]*models.OrderBook          // exchange -> symbol -> OrderBook
	mu         sync.RWMutex

	onUpdate OrderBookUpdateCallback
}

// OrderBookUpdateCallback викликається при оновленні OrderBook
type OrderBookUpdateCallback func(exchange, symbol string, orderbook *models.OrderBook)

// NewOrderBookManager створює новий OrderBook Manager
func NewOrderBookManager() *OrderBookManager {
	return &OrderBookManager{
		wsManagers: make(map[string]websocket.Manager),
		orderbooks: make(map[string]map[string]*models.OrderBook),
	}
}

// RegisterExchange реєструє WebSocket Manager для біржі
func (m *OrderBookManager) RegisterExchange(exchange string, manager websocket.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.wsManagers[exchange] = manager
	m.orderbooks[exchange] = make(map[string]*models.OrderBook)

	// Set callback для оновлень OrderBook
	manager.OnOrderBookUpdate(func(exch, symbol string, ob *models.OrderBook) {
		m.updateOrderBook(exch, symbol, ob)
	})

	log.Printf("✅ Registered exchange: %s", exchange)
}

// updateOrderBook оновлює OrderBook в пам'яті
func (m *OrderBookManager) updateOrderBook(exchange, symbol string, ob *models.OrderBook) {
	m.mu.Lock()
	if m.orderbooks[exchange] == nil {
		m.orderbooks[exchange] = make(map[string]*models.OrderBook)
	}
	m.orderbooks[exchange][symbol] = ob
	m.mu.Unlock()

	// Trigger callback
	if m.onUpdate != nil {
		go m.onUpdate(exchange, symbol, ob)
	}
}

// GetOrderBook отримує OrderBook для конкретної біржі та символу
func (m *OrderBookManager) GetOrderBook(exchange, symbol string) *models.OrderBook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if exchBooks, ok := m.orderbooks[exchange]; ok {
		return exchBooks[symbol]
	}

	return nil
}

// GetAllOrderBooks отримує всі OrderBook для символу з усіх бірж
func (m *OrderBookManager) GetAllOrderBooks(symbol string) map[string]*models.OrderBook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*models.OrderBook)

	for exchange, books := range m.orderbooks {
		if ob, ok := books[symbol]; ok {
			if !ob.IsStale(5 * time.Second) {
				result[exchange] = ob
			}
		}
	}

	return result
}

// OnUpdate встановлює callback для оновлень OrderBook
func (m *OrderBookManager) OnUpdate(callback OrderBookUpdateCallback) {
	m.onUpdate = callback
}

// GetBestPrices знаходить найкращі ціни buy/sell з усіх бірж
func (m *OrderBookManager) GetBestPrices(symbol string) *BestPrices {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var bestBid *ExchangePrice  // Найвища ціна купівлі (продаємо тут)
	var bestAsk *ExchangePrice  // Найнижча ціна продажу (купуємо тут)

	for exchange, books := range m.orderbooks {
		ob := books[symbol]
		if ob == nil || ob.IsStale(5*time.Second) {
			continue
		}

		// Best bid (де продавати)
		if bid := ob.GetBestBid(); bid != nil {
			if bestBid == nil || bid.Price > bestBid.Price {
				bestBid = &ExchangePrice{
					Exchange: exchange,
					Price:    bid.Price,
					Quantity: bid.Quantity,
				}
			}
		}

		// Best ask (де купувати)
		if ask := ob.GetBestAsk(); ask != nil {
			if bestAsk == nil || ask.Price < bestAsk.Price {
				bestAsk = &ExchangePrice{
					Exchange: exchange,
					Price:    ask.Price,
					Quantity: ask.Quantity,
				}
			}
		}
	}

	if bestBid == nil || bestAsk == nil {
		return nil
	}

	return &BestPrices{
		Symbol:  symbol,
		BestBid: bestBid,
		BestAsk: bestAsk,
	}
}

// BestPrices найкращі ціни з усіх бірж
type BestPrices struct {
	Symbol  string
	BestBid *ExchangePrice // Найвища ціна купівлі (продаємо тут)
	BestAsk *ExchangePrice // Найнижча ціна продажу (купуємо тут)
}

// HasArbitrage перевіряє чи є арбітражна можливість
func (bp *BestPrices) HasArbitrage() bool {
	if bp.BestBid == nil || bp.BestAsk == nil {
		return false
	}

	// Якщо можна купити дешевше ніж продати - є арбітраж
	return bp.BestAsk.Price < bp.BestBid.Price
}

// GrossProfit розраховує валовий прибуток в %
func (bp *BestPrices) GrossProfit() float64 {
	if !bp.HasArbitrage() {
		return 0
	}

	return ((bp.BestBid.Price - bp.BestAsk.Price) / bp.BestAsk.Price) * 100
}

// ExchangePrice ціна на конкретній біржі
type ExchangePrice struct {
	Exchange string
	Price    float64
	Quantity float64
}

// GetStats повертає статистику по всіх orderbook'ах
func (m *OrderBookManager) GetStats() *OrderBookStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &OrderBookStats{
		TotalExchanges: len(m.wsManagers),
		OrderBooks:     make(map[string]int),
		StaleBooks:     0,
		FreshBooks:     0,
	}

	for exchange, books := range m.orderbooks {
		stats.OrderBooks[exchange] = len(books)

		for _, ob := range books {
			if ob.IsStale(10 * time.Second) {
				stats.StaleBooks++
			} else {
				stats.FreshBooks++
			}
		}
	}

	return stats
}

// OrderBookStats статистика OrderBook Manager
type OrderBookStats struct {
	TotalExchanges int
	OrderBooks     map[string]int // exchange -> count
	FreshBooks     int
	StaleBooks     int
}

// GetExchanges повертає список зареєстрованих бірж
func (m *OrderBookManager) GetExchanges() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exchanges := make([]string, 0, len(m.wsManagers))
	for exchange := range m.wsManagers {
		exchanges = append(exchanges, exchange)
	}

	return exchanges
}

// IsExchangeConnected перевіряє чи біржа підключена
func (m *OrderBookManager) IsExchangeConnected(exchange string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	manager, ok := m.wsManagers[exchange]
	if !ok {
		return false
	}

	return manager.IsConnected()
}

// GetLiquidity розраховує загальну ліквідність для символу
func (m *OrderBookManager) GetLiquidity(symbol string, side string, maxLevels int) map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	liquidity := make(map[string]float64)

	for exchange, books := range m.orderbooks {
		ob := books[symbol]
		if ob == nil || ob.IsStale(5*time.Second) {
			continue
		}

		liquidity[exchange] = ob.GetLiquidity(side, maxLevels)
	}

	return liquidity
}
