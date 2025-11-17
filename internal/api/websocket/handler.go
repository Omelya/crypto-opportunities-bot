package websocket

import (
	"crypto-opportunities-bot/internal/api/auth"
	"crypto-opportunities-bot/internal/api/middleware"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, check against allowed origins
		// For now, allow all origins (should match CORS settings)
		return true
	},
}

// Handler handles WebSocket upgrade requests
type Handler struct {
	hub        *Hub
	jwtManager *auth.JWTManager
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		hub:        hub,
		jwtManager: jwtManager,
	}
}

// ServeMonitor handles WebSocket connections for real-time monitoring
func (h *Handler) ServeMonitor(w http.ResponseWriter, r *http.Request) {
	// Extract user claims from context (set by JWT middleware)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client with metadata
	metadata := map[string]interface{}{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
	}

	client := NewClient(conn, h.hub, claims.Username, metadata)

	// Register client with hub
	h.hub.register <- client

	// Start client pumps
	client.Run()

	log.Printf("✅ WebSocket client connected: %s (role: %s)", claims.Username, claims.Role)
}

// GetHub returns the WebSocket hub
func (h *Handler) GetHub() *Hub {
	return h.hub
}
