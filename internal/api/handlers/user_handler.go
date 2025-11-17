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
	userRepo repository.UserRepository
}

// NewUserHandler створює новий UserHandler
func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

// ListUsers повертає список користувачів з пагінацією
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// TODO: Add pagination, filters
	// For now, return first 100 users

	users, err := h.userRepo.List(100, 0)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": len(users),
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

	// TODO: Fetch actual stats from database
	stats := map[string]interface{}{
		"user_id":              user.ID,
		"telegram_id":          user.TelegramID,
		"notifications_sent":   0, // TODO: Count from notifications
		"opportunities_viewed": 0, // TODO: Count from user actions
		"subscription_tier":    user.SubscriptionTier,
		"is_premium":           user.IsPremium(),
		"is_active":            user.IsActive,
		"created_at":           user.CreatedAt,
	}

	respondJSON(w, http.StatusOK, stats)
}

// Helper functions

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
