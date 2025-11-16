package websocket

import (
	"context"
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// BybitManager управляє WebSocket з'єднанням з Bybit
type BybitManager struct {
	wsURL             string
	conn              *websocket.Conn
	symbols           []string
	orderbooks        map[string]*models.OrderBook
	mu                sync.RWMutex
	reconnectInterval time.Duration
	pingInterval      time.Duration

	// Callbacks
	onOrderBookUpdate OrderBookCallback
	onTicker          TickerCallback

	ctx    context.Context
	cancel context.CancelFunc

	connected bool
	connMu    sync.RWMutex
}

// NewBybitManager створює новий Bybit WebSocket Manager
func NewBybitManager() *BybitManager {
	return &BybitManager{
		wsURL:             "wss://stream.bybit.com/v5/public/spot",
		orderbooks:        make(map[string]*models.OrderBook),
		reconnectInterval: 5 * time.Second,
		pingInterval:      20 * time.Second,
	}
}

// GetExchange повертає назву біржі
func (m *BybitManager) GetExchange() string {
	return "bybit"
}

// Connect підключається до Bybit WebSocket
func (m *BybitManager) Connect(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	conn, _, err := websocket.DefaultDialer.Dial(m.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Bybit WS: %w", err)
	}

	m.conn = conn
	m.setConnected(true)
	log.Printf("✅ Connected to Bybit WebSocket")

	// Start message handler
	go m.handleMessages()

	// Start ping/pong
	go m.ping()

	// Start connection watcher
	go m.watchConnection()

	return nil
}

// Disconnect від'єднується від WebSocket
func (m *BybitManager) Disconnect() error {
	if m.cancel != nil {
		m.cancel()
	}

	m.setConnected(false)

	if m.conn != nil {
		return m.conn.Close()
	}

	return nil
}

// Subscribe підписується на символи
func (m *BybitManager) Subscribe(symbols []string) error {
	m.mu.Lock()
	m.symbols = symbols
	m.mu.Unlock()

	if !m.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// Bybit subscription format
	args := make([]string, 0, len(symbols)*2)
	for _, symbol := range symbols {
		// Normalize symbol: BTC/USDT -> BTCUSDT
		normalized := normalizeBybitSymbol(symbol)

		// Subscribe to orderbook and ticker
		args = append(args, fmt.Sprintf("orderbook.50.%s", normalized))
		args = append(args, fmt.Sprintf("tickers.%s", normalized))
	}

	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": args,
	}

	data, err := json.Marshal(subMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe message: %w", err)
	}

	if err := m.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	log.Printf("📡 Subscribed to %d symbols on Bybit", len(symbols))
	return nil
}

// Unsubscribe відписується від символів
func (m *BybitManager) Unsubscribe(symbols []string) error {
	if !m.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// Bybit unsubscription format
	args := make([]string, 0, len(symbols)*2)
	for _, symbol := range symbols {
		normalized := normalizeBybitSymbol(symbol)
		args = append(args, fmt.Sprintf("orderbook.50.%s", normalized))
		args = append(args, fmt.Sprintf("tickers.%s", normalized))
	}

	unsubMsg := map[string]interface{}{
		"op":   "unsubscribe",
		"args": args,
	}

	data, err := json.Marshal(unsubMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscribe message: %w", err)
	}

	if err := m.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send unsubscribe message: %w", err)
	}

	return nil
}

// GetOrderBook отримує OrderBook для символу
func (m *BybitManager) GetOrderBook(symbol string) *models.OrderBook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.orderbooks[symbol]
}

// OnOrderBookUpdate встановлює callback для оновлень OrderBook
func (m *BybitManager) OnOrderBookUpdate(callback OrderBookCallback) {
	m.onOrderBookUpdate = callback
}

// OnTicker встановлює callback для ticker updates
func (m *BybitManager) OnTicker(callback TickerCallback) {
	m.onTicker = callback
}

// IsConnected перевіряє статус з'єднання
func (m *BybitManager) IsConnected() bool {
	m.connMu.RLock()
	defer m.connMu.RUnlock()
	return m.connected
}

// setConnected встановлює статус з'єднання
func (m *BybitManager) setConnected(connected bool) {
	m.connMu.Lock()
	m.connected = connected
	m.connMu.Unlock()
}

// handleMessages обробляє вхідні повідомлення
func (m *BybitManager) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Bybit message handler panic: %v", r)
		}
	}()

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			_, message, err := m.conn.ReadMessage()
			if err != nil {
				log.Printf("⚠️ Bybit read error: %v", err)
				m.setConnected(false)
				return
			}

			m.processMessage(message)
		}
	}
}

// processMessage обробляє отримане повідомлення
func (m *BybitManager) processMessage(message []byte) {
	var baseMsg map[string]interface{}
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return
	}

	// Check message type
	topic, ok := baseMsg["topic"].(string)
	if !ok {
		// Could be subscription confirmation or pong
		return
	}

	// OrderBook update
	if strings.HasPrefix(topic, "orderbook.") {
		m.handleOrderBookUpdate(message)
	}

	// Ticker update
	if strings.HasPrefix(topic, "tickers.") {
		m.handleTickerUpdate(message)
	}
}

// handleOrderBookUpdate обробляє оновлення OrderBook
func (m *BybitManager) handleOrderBookUpdate(message []byte) {
	var update struct {
		Topic string `json:"topic"`
		Type  string `json:"type"`
		Data  struct {
			Symbol string          `json:"s"`
			Bids   [][]interface{} `json:"b"`
			Asks   [][]interface{} `json:"a"`
			UpdateID int64        `json:"u"`
			Timestamp int64       `json:"t"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &update); err != nil {
		return
	}

	// Parse symbol (BTCUSDT -> BTC/USDT)
	symbol := denormalizeSymbol(update.Data.Symbol)

	// Create OrderBook
	ob := &models.OrderBook{
		Exchange:  "bybit",
		Symbol:    symbol,
		Timestamp: time.UnixMilli(update.Data.Timestamp),
		Bids:      make([]*models.PriceLevel, 0),
		Asks:      make([]*models.PriceLevel, 0),
	}

	// Parse bids
	for _, bid := range update.Data.Bids {
		if len(bid) < 2 {
			continue
		}

		price, err := parseFloat(bid[0])
		if err != nil {
			continue
		}

		quantity, err := parseFloat(bid[1])
		if err != nil {
			continue
		}

		ob.Bids = append(ob.Bids, &models.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Parse asks
	for _, ask := range update.Data.Asks {
		if len(ask) < 2 {
			continue
		}

		price, err := parseFloat(ask[0])
		if err != nil {
			continue
		}

		quantity, err := parseFloat(ask[1])
		if err != nil {
			continue
		}

		ob.Asks = append(ob.Asks, &models.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Store OrderBook
	m.mu.Lock()
	m.orderbooks[symbol] = ob
	m.mu.Unlock()

	// Trigger callback
	if m.onOrderBookUpdate != nil {
		m.onOrderBookUpdate("bybit", symbol, ob)
	}
}

// handleTickerUpdate обробляє оновлення ticker
func (m *BybitManager) handleTickerUpdate(message []byte) {
	var update struct {
		Topic string `json:"topic"`
		Data  struct {
			Symbol       string `json:"symbol"`
			LastPrice    string `json:"lastPrice"`
			Volume24h    string `json:"volume24h"`
			TurnOver24h  string `json:"turnover24h"`
			PriceChange  string `json:"price24hPcnt"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &update); err != nil {
		return
	}

	// Parse symbol
	symbol := denormalizeSymbol(update.Data.Symbol)

	if m.onTicker != nil {
		// Parse values
		lastPrice, _ := parseFloat(update.Data.LastPrice)
		volume24h, _ := parseFloat(update.Data.TurnOver24h) // Turnover in USDT

		m.onTicker("bybit", symbol, lastPrice, volume24h)
	}
}

// ping надсилає ping повідомлення
func (m *BybitManager) ping() {
	ticker := time.NewTicker(m.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if !m.IsConnected() {
				continue
			}

			pingMsg := map[string]string{
				"op": "ping",
			}

			data, _ := json.Marshal(pingMsg)
			if err := m.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("⚠️ Bybit ping error: %v", err)
			}
		}
	}
}

// watchConnection відслідковує стан з'єднання
func (m *BybitManager) watchConnection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if !m.IsConnected() {
				log.Printf("⚠️ Bybit connection lost, reconnecting...")
				time.Sleep(m.reconnectInterval)

				if err := m.reconnect(); err != nil {
					log.Printf("❌ Bybit reconnect failed: %v", err)
				}
			} else {
				// Check for stale data
				m.mu.RLock()
				staleCount := 0
				for _, ob := range m.orderbooks {
					if ob.IsStale(30 * time.Second) {
						staleCount++
					}
				}
				total := len(m.orderbooks)
				m.mu.RUnlock()

				// If more than 50% orderbooks are stale, reconnect
				if total > 0 && staleCount > total/2 {
					log.Printf("⚠️ Bybit: %d/%d orderbooks stale, reconnecting...", staleCount, total)
					m.reconnect()
				}
			}
		}
	}
}

// reconnect перепідключається до WebSocket
func (m *BybitManager) reconnect() error {
	m.Disconnect()
	time.Sleep(2 * time.Second)

	if err := m.Connect(m.ctx); err != nil {
		return err
	}

	// Re-subscribe to symbols
	m.mu.RLock()
	symbols := m.symbols
	m.mu.RUnlock()

	if len(symbols) > 0 {
		return m.Subscribe(symbols)
	}

	return nil
}

// normalizeBybitSymbol нормалізує символ для Bybit (BTC/USDT -> BTCUSDT)
func normalizeBybitSymbol(symbol string) string {
	return strings.ReplaceAll(symbol, "/", "")
}

// denormalizeSymbol денормалізує символ (BTCUSDT -> BTC/USDT)
func denormalizeSymbol(symbol string) string {
	// Common quote currencies
	quoteCurrencies := []string{"USDT", "USDC", "BTC", "ETH", "BNB"}

	for _, quote := range quoteCurrencies {
		if strings.HasSuffix(symbol, quote) {
			base := strings.TrimSuffix(symbol, quote)
			return base + "/" + quote
		}
	}

	// Fallback: can't determine, return as is
	return symbol
}

// parseFloat парсить float значення
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}
