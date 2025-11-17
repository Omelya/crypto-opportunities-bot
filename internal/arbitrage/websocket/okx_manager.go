package websocket

import (
	"context"
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// OKXManager управляє WebSocket з'єднанням з OKX
type OKXManager struct {
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

// NewOKXManager створює новий OKX WebSocket Manager
func NewOKXManager() *OKXManager {
	return &OKXManager{
		wsURL:             "wss://ws.okx.com:8443/ws/v5/public",
		orderbooks:        make(map[string]*models.OrderBook),
		reconnectInterval: 5 * time.Second,
		pingInterval:      20 * time.Second,
	}
}

// GetExchange повертає назву біржі
func (m *OKXManager) GetExchange() string {
	return "okx"
}

// Connect підключається до OKX WebSocket
func (m *OKXManager) Connect(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	conn, _, err := websocket.DefaultDialer.Dial(m.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to OKX WS: %w", err)
	}

	m.conn = conn
	m.setConnected(true)
	log.Printf("✅ Connected to OKX WebSocket")

	// Start message handler
	go m.handleMessages()

	// Start ping/pong
	go m.ping()

	// Start connection watcher
	go m.watchConnection()

	return nil
}

// Disconnect від'єднується від WebSocket
func (m *OKXManager) Disconnect() error {
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
func (m *OKXManager) Subscribe(symbols []string) error {
	m.mu.Lock()
	m.symbols = symbols
	m.mu.Unlock()

	if !m.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// OKX subscription format
	args := make([]map[string]string, 0, len(symbols)*2)
	for _, symbol := range symbols {
		// Normalize symbol: BTC/USDT -> BTC-USDT
		normalized := normalizeOKXSymbol(symbol)

		// Subscribe to orderbook (books50-l2-tbt = top 50 levels, tick-by-tick)
		args = append(args, map[string]string{
			"channel": "books50-l2-tbt",
			"instId":  normalized,
		})

		// Subscribe to ticker
		args = append(args, map[string]string{
			"channel": "tickers",
			"instId":  normalized,
		})
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

	log.Printf("📡 Subscribed to %d symbols on OKX", len(symbols))
	return nil
}

// Unsubscribe відписується від символів
func (m *OKXManager) Unsubscribe(symbols []string) error {
	if !m.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// OKX unsubscription format
	args := make([]map[string]string, 0, len(symbols)*2)
	for _, symbol := range symbols {
		normalized := normalizeOKXSymbol(symbol)

		args = append(args, map[string]string{
			"channel": "books50-l2-tbt",
			"instId":  normalized,
		})

		args = append(args, map[string]string{
			"channel": "tickers",
			"instId":  normalized,
		})
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
func (m *OKXManager) GetOrderBook(symbol string) *models.OrderBook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.orderbooks[symbol]
}

// OnOrderBookUpdate встановлює callback для оновлень OrderBook
func (m *OKXManager) OnOrderBookUpdate(callback OrderBookCallback) {
	m.onOrderBookUpdate = callback
}

// OnTicker встановлює callback для ticker updates
func (m *OKXManager) OnTicker(callback TickerCallback) {
	m.onTicker = callback
}

// IsConnected перевіряє статус з'єднання
func (m *OKXManager) IsConnected() bool {
	m.connMu.RLock()
	defer m.connMu.RUnlock()
	return m.connected
}

// setConnected встановлює статус з'єднання
func (m *OKXManager) setConnected(connected bool) {
	m.connMu.Lock()
	m.connected = connected
	m.connMu.Unlock()
}

// handleMessages обробляє вхідні повідомлення
func (m *OKXManager) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ OKX message handler panic: %v", r)
		}
	}()

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			_, message, err := m.conn.ReadMessage()
			if err != nil {
				log.Printf("⚠️ OKX read error: %v", err)
				m.setConnected(false)
				return
			}

			m.processMessage(message)
		}
	}
}

// processMessage обробляє отримане повідомлення
func (m *OKXManager) processMessage(message []byte) {
	// Check if it's pong response
	if string(message) == "pong" {
		return
	}

	var baseMsg struct {
		Event string `json:"event"`
		Arg   struct {
			Channel string `json:"channel"`
			InstID  string `json:"instId"`
		} `json:"arg"`
		Data []json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return
	}

	// Skip subscription confirmations
	if baseMsg.Event == "subscribe" || baseMsg.Event == "error" {
		return
	}

	// Check channel
	switch baseMsg.Arg.Channel {
	case "books50-l2-tbt":
		m.handleOrderBookUpdate(message)
	case "tickers":
		m.handleTickerUpdate(message)
	}
}

// handleOrderBookUpdate обробляє оновлення OrderBook
func (m *OKXManager) handleOrderBookUpdate(message []byte) {
	var update struct {
		Arg struct {
			Channel string `json:"channel"`
			InstID  string `json:"instId"`
		} `json:"arg"`
		Data []struct {
			Asks      [][]string `json:"asks"`
			Bids      [][]string `json:"bids"`
			Timestamp string     `json:"ts"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &update); err != nil {
		return
	}

	if len(update.Data) == 0 {
		return
	}

	data := update.Data[0]

	// Parse symbol (BTC-USDT -> BTC/USDT)
	symbol := denormalizeOKXSymbol(update.Arg.InstID)

	// Parse timestamp
	ts := parseFloat(data.Timestamp)
	timestamp := time.UnixMilli(int64(ts))

	// Create OrderBook
	ob := &models.OrderBook{
		Exchange:   "okx",
		Symbol:     symbol,
		LastUpdate: timestamp,
		Bids:       make([]models.PriceLevel, 0),
		Asks:       make([]models.PriceLevel, 0),
	}

	// Parse bids
	for _, bid := range data.Bids {
		if len(bid) < 2 {
			continue
		}

		price := parseFloat(bid[0])
		quantity := parseFloat(bid[1])

		ob.Bids = append(ob.Bids, models.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Parse asks
	for _, ask := range data.Asks {
		if len(ask) < 2 {
			continue
		}

		price := parseFloat(ask[0])

		quantity := parseFloat(ask[1])

		ob.Asks = append(ob.Asks, models.PriceLevel{
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
		m.onOrderBookUpdate("okx", symbol, ob)
	}
}

// handleTickerUpdate обробляє оновлення ticker
func (m *OKXManager) handleTickerUpdate(message []byte) {
	var update struct {
		Arg struct {
			Channel string `json:"channel"`
			InstID  string `json:"instId"`
		} `json:"arg"`
		Data []struct {
			InstID    string `json:"instId"`
			Open24h   string `json:"open24h"`
			Last      string `json:"last"`
			Vol24h    string `json:"vol24h"`
			VolCcy24h string `json:"volCcy24h"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &update); err != nil {
		return
	}

	if len(update.Data) == 0 {
		return
	}

	data := update.Data[0]

	// Parse symbol
	symbol := denormalizeOKXSymbol(data.InstID)

	if m.onTicker != nil {
		// Parse values
		lastPrice := parseFloat(data.Last)
		volume24h := parseFloat(data.VolCcy24h) // Volume in quote currency (USDT)
		open24h := parseFloat(data.Open24h)
		turnover24h := parseFloat(data.Vol24h)
		priceChange := calculatePriceChange(lastPrice, open24h)

		tickerData := &TickerData{
			Symbol:         symbol,
			LastPrice:      lastPrice,
			Volume24h:      volume24h * lastPrice,
			PriceChange:    priceChange,
			PriceChange24h: turnover24h,
			Timestamp:      time.Time{},
		}

		m.onTicker("okx", symbol, tickerData)
	}
}

// ping надсилає ping повідомлення
func (m *OKXManager) ping() {
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

			// OKX uses plain text "ping" message
			if err := m.conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				log.Printf("⚠️ OKX ping error: %v", err)
			}
		}
	}
}

// watchConnection відслідковує стан з'єднання
func (m *OKXManager) watchConnection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if !m.IsConnected() {
				log.Printf("⚠️ OKX connection lost, reconnecting...")
				time.Sleep(m.reconnectInterval)

				if err := m.reconnect(); err != nil {
					log.Printf("❌ OKX reconnect failed: %v", err)
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
					log.Printf("⚠️ OKX: %d/%d orderbooks stale, reconnecting...", staleCount, total)
					m.reconnect()
				}
			}
		}
	}
}

// reconnect перепідключається до WebSocket
func (m *OKXManager) reconnect() error {
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

// normalizeOKXSymbol нормалізує символ для OKX (BTC/USDT -> BTC-USDT)
func normalizeOKXSymbol(symbol string) string {
	return strings.ReplaceAll(symbol, "/", "-")
}

// denormalizeOKXSymbol денормалізує символ з OKX (BTC-USDT -> BTC/USDT)
func denormalizeOKXSymbol(symbol string) string {
	return strings.ReplaceAll(symbol, "-", "/")
}

func calculatePriceChange(lastPrice float64, openPrice float64) float64 {
	change := (lastPrice - openPrice) / openPrice * 100
	return math.Round(change*100) / 100
}
