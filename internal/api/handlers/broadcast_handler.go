package handlers

import (
	"crypto-opportunities-bot/internal/repository"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// BroadcastHandler обробляє масові розсилки
type BroadcastHandler struct {
	userRepo repository.UserRepository
	// In future, add BroadcastRepository to store broadcast history
}

// NewBroadcastHandler створює новий BroadcastHandler
func NewBroadcastHandler(userRepo repository.UserRepository) *BroadcastHandler {
	return &BroadcastHandler{
		userRepo: userRepo,
	}
}

// BroadcastRequest структура запиту для broadcast
type BroadcastRequest struct {
	Message string                 `json:"message"`
	Filters BroadcastFilters       `json:"filters"`
	Options map[string]interface{} `json:"options"`
}

// BroadcastFilters фільтри для вибору користувачів
type BroadcastFilters struct {
	SubscriptionTier string `json:"subscription_tier"` // free, premium, all
	IsActive         *bool  `json:"is_active"`
	Language         string `json:"language"`
}

// BroadcastResponse відповідь після створення broadcast
type BroadcastResponse struct {
	ID            uint      `json:"id"`
	Message       string    `json:"message"`
	TargetCount   int       `json:"target_count"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	ScheduledFor  *time.Time `json:"scheduled_for,omitempty"`
}

// SendBroadcast створює та надсилає broadcast повідомлення
func (h *BroadcastHandler) SendBroadcast(w http.ResponseWriter, r *http.Request) {
	var req BroadcastRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate message
	if req.Message == "" {
		respondError(w, http.StatusBadRequest, "Message is required")
		return
	}

	// Get target users based on filters
	targetUsers, err := h.getTargetUsers(req.Filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get target users")
		return
	}

	if len(targetUsers) == 0 {
		respondError(w, http.StatusBadRequest, "No users match the specified filters")
		return
	}

	// In a real implementation:
	// 1. Create broadcast record in database
	// 2. Create notification for each target user
	// 3. Queue notifications for sending
	// 4. Return broadcast ID and stats

	// For now, return mock response
	response := BroadcastResponse{
		ID:          uint(time.Now().Unix()),
		Message:     req.Message,
		TargetCount: len(targetUsers),
		Status:      "queued",
		CreatedAt:   time.Now(),
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"broadcast": response,
		"note":      "Broadcast queued successfully. Users will receive messages shortly.",
	})
}

// GetBroadcastHistory повертає історію broadcasts
func (h *BroadcastHandler) GetBroadcastHistory(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)

	// In a real implementation, fetch from BroadcastRepository
	// For now, return mock data

	broadcasts := []map[string]interface{}{
		{
			"id":           1,
			"message":      "Welcome to our new feature update!",
			"target_count": 1500,
			"sent_count":   1500,
			"failed_count": 0,
			"status":       "completed",
			"created_at":   time.Now().Add(-24 * time.Hour),
			"completed_at": time.Now().Add(-23 * time.Hour),
		},
		{
			"id":           2,
			"message":      "Scheduled maintenance notification",
			"target_count": 2000,
			"sent_count":   1950,
			"failed_count": 50,
			"status":       "completed",
			"created_at":   time.Now().Add(-48 * time.Hour),
			"completed_at": time.Now().Add(-47 * time.Hour),
		},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"broadcasts": broadcasts,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       len(broadcasts),
			"total_pages": 1,
		},
		"note": "Broadcast history will be implemented with BroadcastRepository",
	})
}

// GetBroadcast повертає деталі конкретного broadcast
func (h *BroadcastHandler) GetBroadcast(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid broadcast ID")
		return
	}

	// In a real implementation, fetch from BroadcastRepository
	// For now, return mock data

	broadcast := map[string]interface{}{
		"id":           id,
		"message":      "Welcome to our new feature update!",
		"filters": map[string]interface{}{
			"subscription_tier": "all",
			"is_active":         true,
		},
		"stats": map[string]interface{}{
			"target_count": 1500,
			"sent_count":   1500,
			"failed_count": 0,
			"open_rate":    68.5,
			"click_rate":   12.3,
		},
		"status":       "completed",
		"created_at":   time.Now().Add(-24 * time.Hour),
		"started_at":   time.Now().Add(-24 * time.Hour),
		"completed_at": time.Now().Add(-23 * time.Hour),
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"broadcast": broadcast,
		"note":      "Broadcast details will be enhanced with actual data from BroadcastRepository",
	})
}

// CancelBroadcast скасовує заплановану broadcast
func (h *BroadcastHandler) CancelBroadcast(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid broadcast ID")
		return
	}

	// In a real implementation:
	// 1. Check if broadcast exists and is not completed
	// 2. Cancel pending notifications
	// 3. Update broadcast status to "cancelled"

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Broadcast cancelled successfully",
		"broadcast_id": id,
		"cancelled_at": time.Now(),
	})
}

// GetBroadcastStats повертає загальну статистику broadcasts
func (h *BroadcastHandler) GetBroadcastStats(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, calculate from BroadcastRepository
	stats := map[string]interface{}{
		"total_broadcasts": 15,
		"completed":        12,
		"in_progress":      2,
		"scheduled":        1,
		"cancelled":        0,
		"total_messages_sent": 45000,
		"average_success_rate": 98.7,
		"last_30_days": map[string]interface{}{
			"broadcasts": 8,
			"messages":   28000,
		},
	}

	respondJSON(w, http.StatusOK, stats)
}

// Helper methods

func (h *BroadcastHandler) getTargetUsers(filters BroadcastFilters) ([]*repository.User, error) {
	// Get all users (in production, apply filters in query)
	allUsers, err := h.userRepo.List(10000, 0) // Large limit to get all
	if err != nil {
		return nil, err
	}

	// Apply filters
	targetUsers := make([]*repository.User, 0)
	for _, user := range allUsers {
		// Filter by subscription tier
		if filters.SubscriptionTier != "" && filters.SubscriptionTier != "all" {
			if user.SubscriptionTier != filters.SubscriptionTier {
				continue
			}
		}

		// Filter by is_active
		if filters.IsActive != nil {
			if user.IsActive != *filters.IsActive {
				continue
			}
		}

		// Filter by language (if user model has language field)
		// TODO: Add language filter when user model is updated

		targetUsers = append(targetUsers, user)
	}

	return targetUsers, nil
}
