package websocket

import (
	"context"
	"crypto-opportunities-bot/internal/models"
	"time"
)

// Manager інтерфейс для WebSocket з'єднань з біржами
type Manager interface {
	// Connect підключається до WebSocket
	Connect(ctx context.Context) error

	// Disconnect від'єднується від WebSocket
	Disconnect() error

	// Subscribe підписується на символи
	Subscribe(symbols []string) error

	// Unsubscribe відписується від символів
	Unsubscribe(symbols []string) error

	// IsConnected перевіряє чи активне з'єднання
	IsConnected() bool

	// GetOrderBook отримує OrderBook для символу
	GetOrderBook(symbol string) *models.OrderBook

	// OnOrderBookUpdate callback при оновленні OrderBook
	OnOrderBookUpdate(callback OrderBookCallback)

	// OnTicker callback при оновленні ticker
	OnTicker(callback TickerCallback)

	// GetExchange назва біржі
	GetExchange() string
}

// OrderBookCallback функція для обробки оновлень OrderBook
type OrderBookCallback func(exchange, symbol string, orderbook *models.OrderBook)

// TickerCallback функція для обробки оновлень ticker
type TickerCallback func(exchange, symbol string, ticker *TickerData)

// TickerData дані тікера
type TickerData struct {
	Symbol       string
	LastPrice    float64
	Volume24h    float64
	PriceChange  float64
	PriceChange24h float64
	Timestamp    time.Time
}
