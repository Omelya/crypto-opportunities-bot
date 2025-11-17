package websocket

import (
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// ClientHub manages all active Premium Client WebSocket connections
type ClientHub struct {
	// Registered clients (SessionID -> PremiumClient)
	clients map[string]*PremiumClient

	// User ID to SessionID mapping (for quick lookup)
	userSessions map[uint]string

	// Register requests from clients
	register chan *PremiumClient

	// Unregister requests from clients
	unregister chan *PremiumClient

	// Broadcast messages to all clients
	broadcast chan *ClientMessage

	// Send message to specific user
	sendToUser chan *UserMessage

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// PremiumClient represents a single Premium Client connection
type PremiumClient struct {
	SessionID string
	UserID    uint
	User      *models.User
	Hub       *ClientHub
	Conn      interface{} // *websocket.Conn (interface для тестування)
	Send      chan *ClientMessage
	mu        sync.Mutex

	// Metadata
	Platform      string
	ClientVersion string
	IPAddress     string
	ConnectedAt   time.Time
	LastHeartbeat time.Time
}

// ClientMessage represents a message for Premium Clients
type ClientMessage struct {
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UserMessage для відправки повідомлення конкретному користувачу
type UserMessage struct {
	UserID  uint
	Message *ClientMessage
}

// NewClientHub creates a new ClientHub
func NewClientHub() *ClientHub {
	return &ClientHub{
		clients:      make(map[string]*PremiumClient),
		userSessions: make(map[uint]string),
		register:     make(chan *PremiumClient),
		unregister:   make(chan *PremiumClient),
		broadcast:    make(chan *ClientMessage, 512), // Larger buffer for opportunities
		sendToUser:   make(chan *UserMessage, 256),
	}
}

// Run starts the hub's main loop
func (ch *ClientHub) Run() {
	log.Println("✅ Premium ClientHub started")

	for {
		select {
		case client := <-ch.register:
			ch.mu.Lock()

			// Disconnect previous session if exists
			if existingSessionID, exists := ch.userSessions[client.UserID]; exists {
				if existingClient, ok := ch.clients[existingSessionID]; ok {
					log.Printf("⚠️ Disconnecting previous session for user %d", client.UserID)
					close(existingClient.Send)
					delete(ch.clients, existingSessionID)
				}
			}

			// Register new client
			ch.clients[client.SessionID] = client
			ch.userSessions[client.UserID] = client.SessionID
			ch.mu.Unlock()

			log.Printf("🔌 Premium client connected: User=%d, Session=%s, Total=%d",
				client.UserID, client.SessionID[:8], len(ch.clients))

			// Send welcome message
			ch.sendToClient(client, &ClientMessage{
				Type:      "connected",
				Data:      map[string]interface{}{"message": "Connected to Premium Trading Hub"},
				Timestamp: time.Now(),
			})

		case client := <-ch.unregister:
			ch.mu.Lock()
			if _, ok := ch.clients[client.SessionID]; ok {
				delete(ch.clients, client.SessionID)
				delete(ch.userSessions, client.UserID)
				close(client.Send)

				log.Printf("🔌 Premium client disconnected: User=%d, Session=%s, Total=%d",
					client.UserID, client.SessionID[:8], len(ch.clients))
			}
			ch.mu.Unlock()

		case message := <-ch.broadcast:
			// Broadcast to all connected clients
			ch.mu.RLock()
			for _, client := range ch.clients {
				select {
				case client.Send <- message:
					// Message sent successfully
				default:
					// Client's send buffer is full
					log.Printf("⚠️ Client buffer full (User=%d), message dropped", client.UserID)
				}
			}
			ch.mu.RUnlock()

		case userMsg := <-ch.sendToUser:
			// Send to specific user
			ch.mu.RLock()
			if sessionID, exists := ch.userSessions[userMsg.UserID]; exists {
				if client, ok := ch.clients[sessionID]; ok {
					select {
					case client.Send <- userMsg.Message:
						// Sent
					default:
						log.Printf("⚠️ Failed to send to user %d, buffer full", userMsg.UserID)
					}
				}
			}
			ch.mu.RUnlock()
		}
	}
}

// RegisterClient registers a new Premium Client
func (ch *ClientHub) RegisterClient(client *PremiumClient) {
	ch.register <- client
}

// UnregisterClient unregisters a Premium Client
func (ch *ClientHub) UnregisterClient(client *PremiumClient) {
	ch.unregister <- client
}

// BroadcastArbitrage broadcasts arbitrage opportunity to all connected clients
func (ch *ClientHub) BroadcastArbitrage(opp *models.ArbitrageOpportunity) {
	message := &ClientMessage{
		Type:      "arbitrage_opportunity",
		Data:      opp,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"opportunity_id": opp.ID,
			"pair":           opp.Pair,
		},
	}

	select {
	case ch.broadcast <- message:
		log.Printf("📢 Broadcasting arbitrage to %d premium clients: %s (%.2f%%)",
			len(ch.clients), opp.Pair, opp.NetProfitPercent)
	default:
		log.Printf("⚠️ Broadcast channel full, arbitrage message dropped")
	}
}

// SendToUser sends a message to a specific user
func (ch *ClientHub) SendToUser(userID uint, messageType string, data interface{}) {
	message := &ClientMessage{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
	}

	ch.sendToUser <- &UserMessage{
		UserID:  userID,
		Message: message,
	}
}

// SendCommand sends a command to a specific user
func (ch *ClientHub) SendCommand(userID uint, command string, data interface{}) {
	ch.SendToUser(userID, "command", map[string]interface{}{
		"action": command,
		"data":   data,
	})
}

// sendToClient sends a message to a specific client (internal)
func (ch *ClientHub) sendToClient(client *PremiumClient, message *ClientMessage) {
	select {
	case client.Send <- message:
		// Message sent
	default:
		// Client buffer full, skip
		log.Printf("⚠️ Client buffer full (User=%d), message skipped", client.UserID)
	}
}

// GetConnectedClients returns the number of connected clients
func (ch *ClientHub) GetConnectedClients() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.clients)
}

// GetConnectedUsers returns list of connected user IDs
func (ch *ClientHub) GetConnectedUsers() []uint {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	users := make([]uint, 0, len(ch.userSessions))
	for userID := range ch.userSessions {
		users = append(users, userID)
	}
	return users
}

// IsUserConnected checks if a user is currently connected
func (ch *ClientHub) IsUserConnected(userID uint) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	_, exists := ch.userSessions[userID]
	return exists
}

// GetClientByUserID returns the client for a specific user
func (ch *ClientHub) GetClientByUserID(userID uint) *PremiumClient {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if sessionID, exists := ch.userSessions[userID]; exists {
		return ch.clients[sessionID]
	}
	return nil
}

// BroadcastTradeResult broadcasts a trade result notification
func (ch *ClientHub) BroadcastTradeResult(trade *models.ClientTrade) {
	message := &ClientMessage{
		Type:      "trade_result",
		Data:      trade,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"trade_id": trade.ID,
			"status":   trade.Status,
		},
	}

	select {
	case ch.broadcast <- message:
		// Broadcast queued
	default:
		log.Printf("⚠️ Broadcast channel full, trade result dropped")
	}
}

// BroadcastStatsUpdate broadcasts statistics update to a specific user
func (ch *ClientHub) BroadcastStatsUpdate(userID uint, stats *models.ClientStatistics) {
	ch.SendToUser(userID, "stats_update", stats)
}

// MarshalJSON converts ClientMessage to JSON
func (cm *ClientMessage) MarshalJSON() ([]byte, error) {
	type Alias ClientMessage
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: cm.Timestamp.Format(time.RFC3339),
		Alias:     (*Alias)(cm),
	})
}

// UpdateHeartbeat updates the last heartbeat time for a client
func (pc *PremiumClient) UpdateHeartbeat() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.LastHeartbeat = time.Now()
}

// IsAlive checks if the client is still alive (recent heartbeat)
func (pc *PremiumClient) IsAlive() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// If no heartbeat in last 2 minutes, consider dead
	return time.Since(pc.LastHeartbeat) < 2*time.Minute
}

// Duration returns how long the client has been connected
func (pc *PremiumClient) Duration() time.Duration {
	return time.Since(pc.ConnectedAt)
}
