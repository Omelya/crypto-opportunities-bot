package handlers

import (
	"crypto-opportunities-bot/internal/api/middleware"
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// ClientAPIHandler обробляє API запити для Premium Client
type ClientAPIHandler struct {
	sessionRepo repository.ClientSessionRepository
	tradeRepo   repository.ClientTradeRepository
	statsRepo   repository.ClientStatisticsRepository
	userRepo    repository.UserRepository
}

// NewClientAPIHandler створює новий ClientAPIHandler
func NewClientAPIHandler(
	sessionRepo repository.ClientSessionRepository,
	tradeRepo repository.ClientTradeRepository,
	statsRepo repository.ClientStatisticsRepository,
	userRepo repository.UserRepository,
) *ClientAPIHandler {
	return &ClientAPIHandler{
		sessionRepo: sessionRepo,
		tradeRepo:   tradeRepo,
		statsRepo:   statsRepo,
		userRepo:    userRepo,
	}
}

// CreateTradeRequest структура запиту для створення трейду
type CreateTradeRequest struct {
	OpportunityID uint    `json:"opportunity_id"`
	Pair          string  `json:"pair"`
	BuyExchange   string  `json:"buy_exchange"`
	SellExchange  string  `json:"sell_exchange"`
	Amount        float64 `json:"amount"`
	BuyPrice      float64 `json:"buy_price"`
	SellPrice     float64 `json:"sell_price"`
	ExpectedProfit float64 `json:"expected_profit"`
}

// UpdateTradeRequest структура запиту для оновлення трейду
type UpdateTradeRequest struct {
	Status              string  `json:"status"`
	BuyOrderID          string  `json:"buy_order_id,omitempty"`
	SellOrderID         string  `json:"sell_order_id,omitempty"`
	ActualProfit        float64 `json:"actual_profit,omitempty"`
	ActualProfitPercent float64 `json:"actual_profit_percent,omitempty"`
	ExecutionTimeMs     int     `json:"execution_time_ms,omitempty"`
	Error               string  `json:"error,omitempty"`
}

// CreateTrade створює новий трейд
func (h *ClientAPIHandler) CreateTrade(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Parse request
	var req CreateTradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if req.OpportunityID == 0 || req.Pair == "" || req.Amount <= 0 {
		respondError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	// Create trade
	trade := &models.ClientTrade{
		UserID:         claims.UserID,
		OpportunityID:  req.OpportunityID,
		Pair:           req.Pair,
		BuyExchange:    req.BuyExchange,
		SellExchange:   req.SellExchange,
		Amount:         req.Amount,
		BuyPrice:       req.BuyPrice,
		SellPrice:      req.SellPrice,
		ExpectedProfit: req.ExpectedProfit,
		Status:         models.TradeStatusPending,
	}

	if err := h.tradeRepo.Create(trade); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create trade")
		return
	}

	respondJSON(w, http.StatusCreated, trade)
}

// UpdateTrade оновлює статус трейду
func (h *ClientAPIHandler) UpdateTrade(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Get trade ID from URL
	vars := mux.Vars(r)
	tradeID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid trade ID")
		return
	}

	// Get trade
	trade, err := h.tradeRepo.GetByID(uint(tradeID))
	if err != nil {
		respondError(w, http.StatusNotFound, "Trade not found")
		return
	}

	// Verify ownership
	if trade.UserID != claims.UserID {
		respondError(w, http.StatusForbidden, "Not authorized to update this trade")
		return
	}

	// Parse request
	var req UpdateTradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update fields
	if req.Status != "" {
		trade.Status = req.Status
	}
	if req.BuyOrderID != "" {
		trade.BuyOrderID = req.BuyOrderID
	}
	if req.SellOrderID != "" {
		trade.SellOrderID = req.SellOrderID
	}
	if req.ActualProfit != 0 {
		trade.ActualProfit = req.ActualProfit
	}
	if req.ActualProfitPercent != 0 {
		trade.ActualProfitPercent = req.ActualProfitPercent
	}
	if req.ExecutionTimeMs > 0 {
		trade.ExecutionTimeMs = req.ExecutionTimeMs
	}
	if req.Error != "" {
		trade.Error = req.Error
	}

	// Set completed time if status is completed or failed
	if req.Status == models.TradeStatusCompleted || req.Status == models.TradeStatusFailed {
		now := time.Now()
		trade.CompletedAt = &now

		// Update statistics
		go func() {
			if err := h.statsRepo.UpdateFromTrade(trade); err != nil {
				// Log error but don't fail the request
				// log.Printf("Failed to update stats: %v", err)
			}
		}()
	}

	// Save
	if err := h.tradeRepo.Update(trade); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update trade")
		return
	}

	respondJSON(w, http.StatusOK, trade)
}

// GetTrades повертає список трейдів користувача
func (h *ClientAPIHandler) GetTrades(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Parse parameters
	limit := parseIntQuery(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	// Get trades
	trades, err := h.tradeRepo.GetByUserID(claims.UserID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get trades")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"trades": trades,
		"total":  len(trades),
	})
}

// GetStatistics повертає статистику користувача
func (h *ClientAPIHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Get or create statistics
	stats, err := h.statsRepo.GetOrCreate(claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get statistics")
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetLeaderboard повертає топ користувачів
func (h *ClientAPIHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	// Parse parameters
	limit := parseIntQuery(r, "limit", 10)
	if limit > 50 {
		limit = 50
	}

	sortBy := r.URL.Query().Get("sort_by") // "profit", "win_rate"
	if sortBy == "" {
		sortBy = "profit"
	}

	var stats []*models.ClientStatistics
	var err error

	switch sortBy {
	case "win_rate":
		stats, err = h.statsRepo.GetLeaderboardByWinRate(limit)
	default:
		stats, err = h.statsRepo.GetLeaderboardByProfit(limit)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get leaderboard")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"leaderboard": stats,
		"sort_by":     sortBy,
		"total":       len(stats),
	})
}

// GetSessions повертає історію сесій користувача
func (h *ClientAPIHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	sessions, err := h.sessionRepo.ListByUserID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get sessions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// GetActiveSession повертає активну сесію користувача
func (h *ClientAPIHandler) GetActiveSession(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	session, err := h.sessionRepo.GetActiveByUserID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get active session")
		return
	}

	if session == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"active": false,
			"session": nil,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"active":  true,
		"session": session,
	})
}
