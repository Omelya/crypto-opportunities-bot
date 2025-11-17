package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Client represents a WebSocket client connection
type Client struct {
	// The WebSocket connection
	conn *websocket.Conn

	// The hub this client is registered to
	hub *Hub

	// Buffered channel of outbound messages
	send chan *Message

	// Client ID (can be user ID or session ID)
	id string

	// Client metadata (e.g., user role, permissions)
	metadata map[string]interface{}

	// Event subscriptions (event_type -> subscribed)
	subscriptions map[string]bool
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn, hub *Hub, id string, metadata map[string]interface{}) *Client {
	return &Client{
		conn:          conn,
		hub:           hub,
		send:          make(chan *Message, 256),
		id:            id,
		metadata:      metadata,
		subscriptions: make(map[string]bool),
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages (e.g., subscription requests)
		c.handleIncomingMessage(message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send message as JSON
			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshaling message: %v", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleIncomingMessage processes messages received from the client
func (c *Client) handleIncomingMessage(data []byte) {
	var msg struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	// Handle different message types
	switch msg.Type {
	case "ping":
		// Respond with pong
		c.send <- &Message{
			Type:      "pong",
			Data:      map[string]string{"status": "ok"},
			Timestamp: time.Now(),
		}

	case "subscribe":
		// Handle subscription to specific event types
		if event, ok := msg.Data["event"].(string); ok {
			c.subscriptions[event] = true
			c.send <- &Message{
				Type: "subscribed",
				Data: map[string]interface{}{
					"event":  event,
					"status": "success",
				},
				Timestamp: time.Now(),
			}
			log.Printf("Client %s subscribed to event: %s", c.id, event)
		}

	case "unsubscribe":
		// Handle unsubscription
		if event, ok := msg.Data["event"].(string); ok {
			delete(c.subscriptions, event)
			c.send <- &Message{
				Type: "unsubscribed",
				Data: map[string]interface{}{
					"event":  event,
					"status": "success",
				},
				Timestamp: time.Now(),
			}
			log.Printf("Client %s unsubscribed from event: %s", c.id, event)
		}

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// Run starts the client's read and write pumps
func (c *Client) Run() {
	go c.writePump()
	go c.readPump()
}
