package handlers

import (
	"crypto-opportunities-bot/internal/analytics"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type AnalyticsHandler struct {
	analyticsService *analytics.Service
}

func NewAnalyticsHandler(analyticsService *analytics.Service) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetUserAnalytics returns analytics for a specific user
// GET /api/v1/analytics/users/:id
func (h *AnalyticsHandler) GetUserAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	analytics, err := h.analyticsService.GetUserAnalytics(uint(userID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if analytics == nil {
		http.Error(w, "User analytics not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// GetUserEngagement returns engagement history for a user
// GET /api/v1/analytics/users/:id/engagement?days=7
func (h *AnalyticsHandler) GetUserEngagement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	days := 7
	if daysParam := r.URL.Query().Get("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}

	engagements, err := h.analyticsService.GetUserEngagementHistory(uint(userID), days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":     userID,
		"days":        days,
		"engagements": engagements,
	})
}

// GetDailyStats returns daily statistics for a date range
// GET /api/v1/analytics/daily?from=2025-01-01&to=2025-01-31
func (h *AnalyticsHandler) GetDailyStats(w http.ResponseWriter, r *http.Request) {
	fromParam := r.URL.Query().Get("from")
	toParam := r.URL.Query().Get("to")

	// Default to last 30 days
	to := time.Now()
	from := to.AddDate(0, 0, -30)

	if fromParam != "" {
		if f, err := time.Parse("2006-01-02", fromParam); err == nil {
			from = f
		}
	}

	if toParam != "" {
		if t, err := time.Parse("2006-01-02", toParam); err == nil {
			to = t
		}
	}

	stats, err := h.analyticsService.GetDailyStatsRange(from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"from":  from.Format("2006-01-02"),
		"to":    to.Format("2006-01-02"),
		"stats": stats,
	})
}

// GetPlatformSummary returns overall platform statistics
// GET /api/v1/analytics/summary
func (h *AnalyticsHandler) GetPlatformSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.analyticsService.GetPlatformSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetTopOpportunities returns top performing opportunities
// GET /api/v1/analytics/opportunities/top?limit=10
func (h *AnalyticsHandler) GetTopOpportunities(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	opportunities, err := h.analyticsService.GetTopOpportunities(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"limit":         limit,
		"opportunities": opportunities,
	})
}

// GetTopUsers returns top users by different metrics
// GET /api/v1/analytics/users/top?order_by=participated&limit=10
func (h *AnalyticsHandler) GetTopUsers(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	orderBy := r.URL.Query().Get("order_by")
	if orderBy == "" {
		orderBy = "participated"
	}

	// Validate orderBy
	validOrders := map[string]bool{
		"viewed":       true,
		"participated": true,
		"conversion":   true,
		"engagement":   true,
	}

	if !validOrders[orderBy] {
		http.Error(w, "Invalid order_by parameter. Valid values: viewed, participated, conversion, engagement", http.StatusBadRequest)
		return
	}

	users, err := h.analyticsService.GetTopUsers(limit, orderBy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"limit":    limit,
		"order_by": orderBy,
		"users":    users,
	})
}

// TrackAction manually tracks a user action (for testing/debugging)
// POST /api/v1/analytics/track
func (h *AnalyticsHandler) TrackAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID        uint                   `json:"user_id"`
		ActionType    string                 `json:"action_type"`
		OpportunityID *uint                  `json:"opportunity_id,omitempty"`
		Metadata      map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 || req.ActionType == "" {
		http.Error(w, "user_id and action_type are required", http.StatusBadRequest)
		return
	}

	if err := h.analyticsService.TrackAction(req.UserID, req.ActionType, req.OpportunityID, req.Metadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Action tracked successfully",
	})
}

// RecordSession manually records a user session (for testing/debugging)
// POST /api/v1/analytics/session
func (h *AnalyticsHandler) RecordSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   uint `json:"user_id"`
		Duration int  `json:"duration"` // in seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 || req.Duration <= 0 {
		http.Error(w, "user_id and duration are required", http.StatusBadRequest)
		return
	}

	if err := h.analyticsService.RecordSession(req.UserID, req.Duration); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Session recorded successfully",
	})
}
