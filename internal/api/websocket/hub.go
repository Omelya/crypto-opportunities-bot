package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Hub manages all active WebSocket connections
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to all clients
	broadcast chan *Message

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256), // Buffered channel
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("🔌 WebSocket client connected. Total clients: %d", len(h.clients))

			// Send welcome message
			h.sendToClient(client, &Message{
				Type:      "connected",
				Data:      map[string]interface{}{"message": "Connected to monitoring hub"},
				Timestamp: time.Now(),
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("🔌 WebSocket client disconnected. Total clients: %d", len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
					// Message sent successfully
				default:
					// Client's send buffer is full, close connection
					close(client.send)
					delete(h.clients, client)
					log.Printf("⚠️ Client send buffer full, disconnecting")
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(messageType string, data interface{}) {
	message := &Message{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
		// Message queued successfully
	default:
		log.Printf("⚠️ Broadcast channel full, message dropped")
	}
}

// BroadcastWithMetadata sends a message with metadata to all clients
func (h *Hub) BroadcastWithMetadata(messageType string, data interface{}, metadata map[string]interface{}) {
	message := &Message{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	select {
	case h.broadcast <- message:
		// Message queued successfully
	default:
		log.Printf("⚠️ Broadcast channel full, message dropped")
	}
}

// sendToClient sends a message to a specific client
func (h *Hub) sendToClient(client *Client, message *Message) {
	select {
	case client.send <- message:
		// Message sent
	default:
		// Client buffer full, skip
		log.Printf("⚠️ Client buffer full, message skipped")
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BroadcastSystemMetrics broadcasts system metrics to all clients
func (h *Hub) BroadcastSystemMetrics(metrics interface{}) {
	h.Broadcast("system.metrics", metrics)
}

// BroadcastNotification broadcasts a notification event
func (h *Hub) BroadcastNotification(event string, notification interface{}) {
	h.Broadcast("notification."+event, notification)
}

// BroadcastScraperEvent broadcasts a scraper event
func (h *Hub) BroadcastScraperEvent(event string, data interface{}) {
	h.Broadcast("scraper."+event, data)
}

// BroadcastUserAction broadcasts a user action event
func (h *Hub) BroadcastUserAction(action string, data interface{}) {
	h.Broadcast("user."+action, data)
}

// BroadcastOpportunity broadcasts an opportunity event
func (h *Hub) BroadcastOpportunity(event string, opportunity interface{}) {
	h.Broadcast("opportunity."+event, opportunity)
}

// MarshalMessage converts a message to JSON
func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: m.Timestamp.Format(time.RFC3339),
		Alias:     (*Alias)(m),
	})
}
