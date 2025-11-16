package websocket

import (
	"context"
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// BinanceManager управляє WebSocket з'єднанням з Binance
type BinanceManager struct {
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

// NewBinanceManager створює новий Binance WebSocket Manager
func NewBinanceManager() *BinanceManager {
	return &BinanceManager{
		wsURL:             "wss://stream.binance.com:9443/ws",
		orderbooks:        make(map[string]*models.OrderBook),
		reconnectInterval: 5 * time.Second,
		pingInterval:      20 * time.Second,
	}
}

// GetExchange повертає назву біржі
func (m *BinanceManager) GetExchange() string {
	return "binance"
}

// Connect підключається до Binance WebSocket
func (m *BinanceManager) Connect(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	conn, _, err := websocket.DefaultDialer.Dial(m.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Binance WS: %w", err)
	}

	m.conn = conn
	m.setConnected(true)
	log.Printf("✅ Connected to Binance WebSocket")

	// Start message handler
	go m.handleMessages()

	// Start ping/pong
	go m.ping()

	// Start connection watcher
	go m.watchConnection()

	return nil
}

// Disconnect від'єднується від WebSocket
func (m *BinanceManager) Disconnect() error {
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
func (m *BinanceManager) Subscribe(symbols []string) error {
	m.mu.Lock()
	m.symbols = symbols
	m.mu.Unlock()

	if !m.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// Create subscription message
	streams := []string{}

	for _, symbol := range symbols {
		symbolLower := m.formatSymbol(symbol)

		// Subscribe to depth (orderbook) - 20 levels, 100ms updates
		streams = append(streams, fmt.Sprintf("%s@depth20@100ms", symbolLower))

		// Subscribe to ticker (24h stats)
		streams = append(streams, fmt.Sprintf("%s@ticker", symbolLower))
	}

	subscribeMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": streams,
		"id":     time.Now().Unix(),
	}

	log.Printf("🔔 Subscribing to %d streams on Binance", len(streams))

	return m.conn.WriteJSON(subscribeMsg)
}

// Unsubscribe відписується від символів
func (m *BinanceManager) Unsubscribe(symbols []string) error {
	if !m.IsConnected() {
		return fmt.Errorf("not connected")
	}

	streams := []string{}
	for _, symbol := range symbols {
		symbolLower := m.formatSymbol(symbol)
		streams = append(streams, fmt.Sprintf("%s@depth20@100ms", symbolLower))
		streams = append(streams, fmt.Sprintf("%s@ticker", symbolLower))
	}

	unsubscribeMsg := map[string]interface{}{
		"method": "UNSUBSCRIBE",
		"params": streams,
		"id":     time.Now().Unix(),
	}

	return m.conn.WriteJSON(unsubscribeMsg)
}

// GetOrderBook отримує OrderBook для символу
func (m *BinanceManager) GetOrderBook(symbol string) *models.OrderBook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.orderbooks[symbol]
}

// OnOrderBookUpdate встановлює callback для оновлень OrderBook
func (m *BinanceManager) OnOrderBookUpdate(callback OrderBookCallback) {
	m.onOrderBookUpdate = callback
}

// OnTicker встановлює callback для оновлень ticker
func (m *BinanceManager) OnTicker(callback TickerCallback) {
	m.onTicker = callback
}

// IsConnected перевіряє чи з'єднання активне
func (m *BinanceManager) IsConnected() bool {
	m.connMu.RLock()
	defer m.connMu.RUnlock()
	return m.connected
}

func (m *BinanceManager) setConnected(status bool) {
	m.connMu.Lock()
	defer m.connMu.Unlock()
	m.connected = status
}

// handleMessages обробляє вхідні WebSocket повідомлення
func (m *BinanceManager) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Recovered in handleMessages: %v", r)
		}
	}()

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			_, message, err := m.conn.ReadMessage()
			if err != nil {
				log.Printf("❌ WebSocket read error (Binance): %v", err)
				m.setConnected(false)
				go m.reconnect()
				return
			}

			m.processMessage(message)
		}
	}
}

// processMessage обробляє окреме повідомлення
func (m *BinanceManager) processMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	// Check message type
	stream, ok := msg["stream"].(string)
	if !ok {
		// Може бути subscription confirmation
		return
	}

	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return
	}

	if strings.Contains(stream, "@depth") {
		m.processDepthUpdate(stream, data)
	} else if strings.Contains(stream, "@ticker") {
		m.processTickerUpdate(stream, data)
	}
}

// processDepthUpdate обробляє оновлення ордербуку
func (m *BinanceManager) processDepthUpdate(stream string, data map[string]interface{}) {
	// Parse symbol from stream
	symbol := m.extractSymbol(stream)
	if symbol == "" {
		return
	}

	// Parse bids and asks
	bidsRaw, ok1 := data["bids"].([]interface{})
	asksRaw, ok2 := data["asks"].([]interface{})

	if !ok1 || !ok2 {
		return
	}

	bids := m.parsePriceLevels(bidsRaw)
	asks := m.parsePriceLevels(asksRaw)

	// Get or create orderbook
	m.mu.Lock()
	orderbook, exists := m.orderbooks[symbol]
	if !exists {
		orderbook = models.NewOrderBook("binance", symbol)
		m.orderbooks[symbol] = orderbook
	}
	m.mu.Unlock()

	// Update orderbook
	updateID := int64(parseFloat(data["lastUpdateId"]))
	orderbook.Update(bids, asks, updateID)

	// Trigger callback
	if m.onOrderBookUpdate != nil {
		go m.onOrderBookUpdate("binance", symbol, orderbook)
	}
}

// processTickerUpdate обробляє оновлення тікера
func (m *BinanceManager) processTickerUpdate(stream string, data map[string]interface{}) {
	symbol := m.extractSymbol(stream)
	if symbol == "" {
		return
	}

	ticker := &TickerData{
		Symbol:         symbol,
		LastPrice:      parseFloat(data["c"]),
		Volume24h:      parseFloat(data["v"]) * parseFloat(data["c"]), // volume * price
		PriceChange:    parseFloat(data["p"]),
		PriceChange24h: parseFloat(data["P"]),
		Timestamp:      time.Now(),
	}

	// Trigger callback
	if m.onTicker != nil {
		go m.onTicker("binance", symbol, ticker)
	}
}

// parsePriceLevels парсить масив [price, quantity] в PriceLevel
func (m *BinanceManager) parsePriceLevels(levels []interface{}) []models.PriceLevel {
	result := make([]models.PriceLevel, 0, len(levels))

	for _, level := range levels {
		levelArr, ok := level.([]interface{})
		if !ok || len(levelArr) < 2 {
			continue
		}

		price := parseFloat(levelArr[0])
		quantity := parseFloat(levelArr[1])

		if price > 0 && quantity > 0 {
			result = append(result, models.PriceLevel{
				Price:    price,
				Quantity: quantity,
			})
		}
	}

	return result
}

// extractSymbol витягує символ зі stream name
func (m *BinanceManager) extractSymbol(stream string) string {
	// "btcusdt@depth20@100ms" -> "BTC/USDT"
	parts := strings.Split(stream, "@")
	if len(parts) == 0 {
		return ""
	}

	symbolLower := parts[0]

	// Convert btcusdt -> BTC/USDT
	return m.parseSymbol(symbolLower)
}

// formatSymbol конвертує "BTC/USDT" -> "btcusdt"
func (m *BinanceManager) formatSymbol(symbol string) string {
	return strings.ToLower(strings.ReplaceAll(symbol, "/", ""))
}

// parseSymbol конвертує "btcusdt" -> "BTC/USDT"
func (m *BinanceManager) parseSymbol(symbolLower string) string {
	// Список популярних quote currencies
	quotes := []string{"usdt", "busd", "usdc", "btc", "eth", "bnb"}

	for _, quote := range quotes {
		if strings.HasSuffix(symbolLower, quote) {
			base := strings.ToUpper(strings.TrimSuffix(symbolLower, quote))
			return base + "/" + strings.ToUpper(quote)
		}
	}

	return ""
}

// ping відправляє ping повідомлення для підтримки з'єднання
func (m *BinanceManager) ping() {
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

			if err := m.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("⚠️ Ping error (Binance): %v", err)
				m.setConnected(false)
				go m.reconnect()
				return
			}
		}
	}
}

// watchConnection слідкує за станом з'єднання
func (m *BinanceManager) watchConnection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// Check if orderbooks are fresh
			m.mu.RLock()
			staleCount := 0
			totalCount := len(m.orderbooks)

			for _, ob := range m.orderbooks {
				if ob.IsStale(60 * time.Second) {
					staleCount++
				}
			}
			m.mu.RUnlock()

			// If more than half orderbooks are stale - reconnect
			if totalCount > 0 && staleCount > totalCount/2 {
				log.Printf("⚠️ Too many stale orderbooks (%d/%d), reconnecting...", staleCount, totalCount)
				m.setConnected(false)
				go m.reconnect()
			}
		}
	}
}

// reconnect перепідключається до WebSocket
func (m *BinanceManager) reconnect() {
	if m.IsConnected() {
		return // Already connected
	}

	log.Printf("🔄 Reconnecting to Binance WebSocket...")

	m.Disconnect()

	time.Sleep(m.reconnectInterval)

	if err := m.Connect(m.ctx); err != nil {
		log.Printf("❌ Reconnection failed (Binance): %v", err)
		// Retry after interval
		time.Sleep(m.reconnectInterval)
		go m.reconnect()
		return
	}

	// Resubscribe to symbols
	m.mu.RLock()
	symbols := m.symbols
	m.mu.RUnlock()

	if len(symbols) > 0 {
		if err := m.Subscribe(symbols); err != nil {
			log.Printf("❌ Resubscription failed (Binance): %v", err)
		}
	}

	log.Printf("✅ Reconnected to Binance WebSocket")
}

// parseFloat парсить interface{} в float64
func parseFloat(val interface{}) float64 {
	switch v := val.(type) {
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}
