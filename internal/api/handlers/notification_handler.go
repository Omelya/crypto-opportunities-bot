package handlers

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// NotificationHandler обробляє запити пов'язані з нотифікаціями
type NotificationHandler struct {
	notifRepo repository.NotificationRepository
	userRepo  repository.UserRepository
	oppRepo   repository.OpportunityRepository
}

// NewNotificationHandler створює новий NotificationHandler
func NewNotificationHandler(
	notifRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	oppRepo repository.OpportunityRepository,
) *NotificationHandler {
	return &NotificationHandler{
		notifRepo: notifRepo,
		userRepo:  userRepo,
		oppRepo:   oppRepo,
	}
}

// ListNotifications повертає список нотифікацій з пагінацією та фільтрами
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 50)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Parse filters
	userIDStr := r.URL.Query().Get("user_id")
	status := r.URL.Query().Get("status") // pending, sent, failed
	oppIDStr := r.URL.Query().Get("opportunity_id")

	// Build query based on filters
	var notifications []*models.Notification
	var total int64
	var err error

	// Get notifications based on filters
	if status != "" {
		// Filter by status
		notifications, err = h.getNotificationsByStatus(status, limit, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch notifications")
			return
		}
		total, _ = h.notifRepo.CountByStatus(status)
	} else if userIDStr != "" {
		// Filter by user
		userID, _ := strconv.ParseUint(userIDStr, 10, 32)
		notifications, err = h.getNotificationsByUser(uint(userID), limit, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch user notifications")
			return
		}
		// Count total for user (we'll need to add this method)
		total = int64(len(notifications)) // Temporary
	} else if oppIDStr != "" {
		// Filter by opportunity
		oppID, _ := strconv.ParseUint(oppIDStr, 10, 32)
		notifications, err = h.getNotificationsByOpportunity(uint(oppID), limit, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch opportunity notifications")
			return
		}
		total = int64(len(notifications)) // Temporary
	} else {
		// Get all notifications (we'll need to add List method to repository)
		respondError(w, http.StatusBadRequest, "Please provide at least one filter (user_id or status)")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetNotification повертає деталі конкретної нотифікації
func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	notification, err := h.notifRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch notification")
		return
	}

	if notification == nil {
		respondError(w, http.StatusNotFound, "Notification not found")
		return
	}

	respondJSON(w, http.StatusOK, notification)
}

// RetryNotification повторно надсилає failed нотифікацію
func (h *NotificationHandler) RetryNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	notification, err := h.notifRepo.GetByID(uint(id))
	if err != nil || notification == nil {
		respondError(w, http.StatusNotFound, "Notification not found")
		return
	}

	// Check if notification is failed
	if notification.Status != models.NotificationStatusFailed {
		respondError(w, http.StatusBadRequest, "Can only retry failed notifications")
		return
	}

	// Reset status to pending
	notification.Status = models.NotificationStatusPending
	notification.ErrorMessage = ""

	if err := h.notifRepo.Update(notification); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update notification")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Notification queued for retry",
		"notification": notification,
	})
}

// DeleteNotification видаляє нотифікацію
func (h *NotificationHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	if err := h.notifRepo.Delete(uint(id)); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete notification")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Notification deleted successfully",
		"notification_id": id,
	})
}

// GetNotificationStats повертає статистику нотифікацій
func (h *NotificationHandler) GetNotificationStats(w http.ResponseWriter, r *http.Request) {
	pendingCount, _ := h.notifRepo.CountByStatus(models.NotificationStatusPending)
	sentCount, _ := h.notifRepo.CountByStatus(models.NotificationStatusSent)
	failedCount, _ := h.notifRepo.CountByStatus(models.NotificationStatusFailed)

	stats := map[string]interface{}{
		"pending": pendingCount,
		"sent":    sentCount,
		"failed":  failedCount,
		"total":   pendingCount + sentCount + failedCount,
		"by_status": map[string]int64{
			"pending": pendingCount,
			"sent":    sentCount,
			"failed":  failedCount,
		},
	}

	respondJSON(w, http.StatusOK, stats)
}

// RetryAllFailed повторює всі failed нотифікації
func (h *NotificationHandler) RetryAllFailed(w http.ResponseWriter, r *http.Request) {
	// Get limit from query param (default 100)
	limit := parseIntQuery(r, "limit", 100)

	// Get failed notifications
	notifications, err := h.notifRepo.GetFailed(limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch failed notifications")
		return
	}

	// Reset all to pending
	resetCount := 0
	for _, notif := range notifications {
		notif.Status = models.NotificationStatusPending
		notif.ErrorMessage = ""

		if err := h.notifRepo.Update(notif); err == nil {
			resetCount++
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Failed notifications queued for retry",
		"reset_count":  resetCount,
		"total_failed": len(notifications),
	})
}

// Helper methods

func (h *NotificationHandler) getNotificationsByStatus(status string, limit, offset int) ([]*models.Notification, error) {
	// This is a simplified implementation
	// In production, you'd want to add a proper List method to repository
	switch status {
	case models.NotificationStatusPending:
		return h.notifRepo.GetPending(limit)
	case models.NotificationStatusFailed:
		return h.notifRepo.GetFailed(limit)
	default:
		// For sent status or other, we'd need a new repository method
		return []*models.Notification{}, nil
	}
}

func (h *NotificationHandler) getNotificationsByUser(userID uint, limit, offset int) ([]*models.Notification, error) {
	// Use existing method, but it only gets pending
	// In production, add a proper ListByUser method to repository
	return h.notifRepo.GetPendingForUser(userID, limit)
}

func (h *NotificationHandler) getNotificationsByOpportunity(oppID uint, limit, offset int) ([]*models.Notification, error) {
	// This would require a new repository method
	// For now, return empty array
	return []*models.Notification{}, nil
}
