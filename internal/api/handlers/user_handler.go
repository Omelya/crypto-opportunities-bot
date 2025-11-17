package handlers

import (
	"crypto-opportunities-bot/internal/repository"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// UserHandler обробляє запити пов'язані з користувачами
type UserHandler struct {
	userRepo   repository.UserRepository
	actionRepo repository.UserActionRepository
	notifRepo  repository.NotificationRepository
}

// NewUserHandler створює новий UserHandler
func NewUserHandler(
	userRepo repository.UserRepository,
	actionRepo repository.UserActionRepository,
	notifRepo repository.NotificationRepository,
) *UserHandler {
	return &UserHandler{
		userRepo:   userRepo,
		actionRepo: actionRepo,
		notifRepo:  notifRepo,
	}
}

// ListUsers повертає список користувачів з пагінацією та фільтрами
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	if limit > 100 {
		limit = 100 // Max 100 per page
	}

	offset := (page - 1) * limit

	// Parse filters
	tier := r.URL.Query().Get("tier")          // free, premium
	isActive := r.URL.Query().Get("is_active") // true, false

	// Get total count (TODO: add filter support to CountAll)
	total, err := h.userRepo.CountAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count users")
		return
	}

	// Fetch users
	users, err := h.userRepo.List(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	// Apply client-side filtering (TODO: move to repository for efficiency)
	filteredUsers := users
	if tier != "" || isActive != "" {
		filteredUsers = make([]*repository.User, 0)
		for _, user := range users {
			// Filter by tier
			if tier != "" && user.SubscriptionTier != tier {
				continue
			}
			// Filter by is_active
			if isActive != "" {
				activeFilter, _ := strconv.ParseBool(isActive)
				if user.IsActive != activeFilter {
					continue
				}
			}
			filteredUsers = append(filteredUsers, user)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": filteredUsers,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetUser повертає інформацію про конкретного користувача
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// UpdateUser оновлює користувача
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Decode update request
	var updateReq struct {
		IsBlocked        *bool   `json:"is_blocked"`
		SubscriptionTier *string `json:"subscription_tier"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Apply updates
	if updateReq.IsBlocked != nil {
		user.IsBlocked = *updateReq.IsBlocked
	}
	if updateReq.SubscriptionTier != nil {
		user.SubscriptionTier = *updateReq.SubscriptionTier
	}

	// Save
	if err := h.userRepo.Update(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// DeleteUser видаляє користувача (soft delete)
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Soft delete
	if err := h.userRepo.Delete(uint(id)); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User deleted successfully",
		"user_id": user.ID,
	})
}

// GetUserStats повертає статистику користувача
func (h *UserHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Count notifications
	sentCount, _ := h.notifRepo.CountByUserAndStatus(user.ID, "sent")
	failedCount, _ := h.notifRepo.CountByUserAndStatus(user.ID, "failed")

	// Count actions
	actionsCount, _ := h.actionRepo.CountByUser(user.ID)

	stats := map[string]interface{}{
		"user_id":       user.ID,
		"telegram_id":   user.TelegramID,
		"username":      user.Username,
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"notifications": map[string]interface{}{
			"sent":   sentCount,
			"failed": failedCount,
			"total":  sentCount + failedCount,
		},
		"actions_count":     actionsCount,
		"subscription_tier": user.SubscriptionTier,
		"is_premium":        user.IsPremium(),
		"is_active":         user.IsActive,
		"is_blocked":        user.IsBlocked,
		"capital_range":     user.CapitalRange,
		"risk_profile":      user.RiskProfile,
		"created_at":        user.CreatedAt,
		"updated_at":        user.UpdatedAt,
	}

	if user.SubscriptionExpiresAt != nil {
		stats["subscription_expires_at"] = user.SubscriptionExpiresAt
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetUserActions повертає історію дій користувача
func (h *UserHandler) GetUserActions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Check if user exists
	_, err = h.userRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Parse pagination
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 50)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Fetch actions
	actions, err := h.actionRepo.GetByUserID(uint(id), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch user actions")
		return
	}

	// Count total
	total, _ := h.actionRepo.CountByUser(uint(id))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"actions": actions,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// Helper functions

func parseIntQuery(r *http.Request, key string, defaultValue int) int {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return value
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}
