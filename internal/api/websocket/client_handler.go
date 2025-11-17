package websocket

import (
	"crypto-opportunities-bot/internal/api/auth"
	"crypto-opportunities-bot/internal/api/middleware"
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var premiumUpgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: In production, check against allowed origins
		return true
	},
}

// ClientHandler handles Premium Client WebSocket connections
type ClientHandler struct {
	clientHub   *ClientHub
	jwtManager  *auth.JWTManager
	sessionRepo repository.ClientSessionRepository
	userRepo    repository.UserRepository
}

// NewClientHandler creates a new ClientHandler
func NewClientHandler(
	clientHub *ClientHub,
	jwtManager *auth.JWTManager,
	sessionRepo repository.ClientSessionRepository,
	userRepo repository.UserRepository,
) *ClientHandler {
	return &ClientHandler{
		clientHub:   clientHub,
		jwtManager:  jwtManager,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

// ServePremiumClient handles Premium Client WebSocket upgrade requests
func (h *ClientHandler) ServePremiumClient(w http.ResponseWriter, r *http.Request) {
	// Extract user claims from context (set by JWT middleware)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := h.userRepo.GetByID(claims.UserID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check premium status
	if !user.IsPremium() {
		http.Error(w, "Premium subscription required", http.StatusForbidden)
		return
	}

	// Check if user already has an active session
	existingSession, err := h.sessionRepo.GetActiveByUserID(user.ID)
	if err == nil && existingSession != nil {
		log.Printf("⚠️ User %d already has active session, will disconnect old one", user.ID)
		// Old session will be disconnected when new one registers
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := premiumUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		return
	}

	// Generate unique session ID
	sessionID := uuid.New().String()

	// Extract client metadata
	platform := r.URL.Query().Get("platform") // "windows", "linux", "macos"
	if platform == "" {
		platform = "unknown"
	}

	clientVersion := r.URL.Query().Get("version")
	if clientVersion == "" {
		clientVersion = "1.0.0"
	}

	ipAddress := r.RemoteAddr

	// Create session in database
	now := time.Now()
	session := &models.ClientSession{
		UserID:        user.ID,
		SessionID:     sessionID,
		ConnectionID:  sessionID, // Same for now
		ClientVersion: clientVersion,
		Platform:      platform,
		IPAddress:     ipAddress,
		IsActive:      true,
		LastHeartbeat: now,
		ConnectedAt:   now,
	}

	if err := h.sessionRepo.Create(session); err != nil {
		log.Printf("❌ Failed to create session: %v", err)
		conn.Close()
		return
	}

	// Create PremiumClient
	client := &PremiumClient{
		SessionID:     sessionID,
		UserID:        user.ID,
		User:          user,
		Hub:           h.clientHub,
		Conn:          conn,
		Send:          make(chan *ClientMessage, 256),
		Platform:      platform,
		ClientVersion: clientVersion,
		IPAddress:     ipAddress,
		ConnectedAt:   now,
		LastHeartbeat: now,
	}

	// Register client with hub
	h.clientHub.RegisterClient(client)

	// Start client goroutines
	go h.writePump(client, conn)
	go h.readPump(client, conn)

	log.Printf("✅ Premium client connected: User=%d, Session=%s, Platform=%s, Version=%s",
		user.ID, sessionID[:8], platform, clientVersion)
}

// readPump reads messages from the WebSocket connection
func (h *ClientHandler) readPump(client *PremiumClient, conn *websocket.Conn) {
	defer func() {
		h.clientHub.UnregisterClient(client)
		conn.Close()

		// Update session in DB
		if err := h.sessionRepo.Disconnect(client.SessionID); err != nil {
			log.Printf("❌ Failed to disconnect session: %v", err)
		}
	}()

	// Configure WebSocket read settings
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		var msg ClientMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("⚠️ WebSocket error (User=%d): %v", client.UserID, err)
			}
			break
		}

		// Handle different message types
		h.handleClientMessage(client, &msg)
	}
}

// writePump writes messages to the WebSocket connection
func (h *ClientHandler) writePump(client *PremiumClient, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteJSON(message); err != nil {
				log.Printf("❌ Write error (User=%d): %v", client.UserID, err)
				return
			}

		case <-ticker.C:
			// Send ping
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage handles messages from Premium Client
func (h *ClientHandler) handleClientMessage(client *PremiumClient, msg *ClientMessage) {
	switch msg.Type {
	case "heartbeat":
		// Update heartbeat in memory
		client.UpdateHeartbeat()

		// Update heartbeat in database (async)
		go func() {
			if err := h.sessionRepo.UpdateHeartbeat(client.SessionID); err != nil {
				log.Printf("⚠️ Failed to update heartbeat: %v", err)
			}
		}()

		// Send heartbeat response
		client.Send <- &ClientMessage{
			Type:      "heartbeat_ack",
			Timestamp: time.Now(),
		}

	case "trade_executed", "trade_failed":
		// Trade result from client
		log.Printf("📊 Trade result from User=%d: %s", client.UserID, msg.Type)
		// This will be handled by API endpoint, not WebSocket
		// Just log for now

	case "ping":
		// Simple ping-pong
		client.Send <- &ClientMessage{
			Type:      "pong",
			Timestamp: time.Now(),
		}

	default:
		log.Printf("⚠️ Unknown message type from User=%d: %s", client.UserID, msg.Type)
	}
}

// GetHub returns the ClientHub
func (h *ClientHandler) GetHub() *ClientHub {
	return h.clientHub
}

// CleanupStaleSessions cleanup goroutine for stale sessions
func (h *ClientHandler) CleanupStaleSessions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Cleanup sessions with no heartbeat for > 2 minutes
		if err := h.sessionRepo.CleanupStale(2 * time.Minute); err != nil {
			log.Printf("⚠️ Failed to cleanup stale sessions: %v", err)
		}
	}
}
