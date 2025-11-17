package handlers

import (
	"crypto-opportunities-bot/internal/repository"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// ArbitrageHandler обробляє запити пов'язані з arbitrage
type ArbitrageHandler struct {
	arbRepo repository.ArbitrageRepository
}

// NewArbitrageHandler створює новий ArbitrageHandler
func NewArbitrageHandler(arbRepo repository.ArbitrageRepository) *ArbitrageHandler {
	return &ArbitrageHandler{
		arbRepo: arbRepo,
	}
}

// ListArbitrage повертає список arbitrage opportunities
func (h *ArbitrageHandler) ListArbitrage(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Parse filters
	pair := r.URL.Query().Get("pair")                 // BTC/USDT, ETH/USDT
	minProfit := r.URL.Query().Get("min_profit")      // minimum profit %
	exchangeBuy := r.URL.Query().Get("exchange_buy")  // binance, bybit, okx
	exchangeSell := r.URL.Query().Get("exchange_sell")

	// Fetch arbitrage opportunities
	arbs, err := h.arbRepo.List(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch arbitrage opportunities")
		return
	}

	// Apply filters
	filteredArbs := arbs
	if pair != "" || minProfit != "" || exchangeBuy != "" || exchangeSell != "" {
		filteredArbs = make([]*repository.ArbitrageOpportunity, 0)
		minProfitFloat := 0.0
		if minProfit != "" {
			minProfitFloat, _ = strconv.ParseFloat(minProfit, 64)
		}

		for _, arb := range arbs {
			if pair != "" && arb.Pair != pair {
				continue
			}
			if minProfit != "" && arb.NetProfitPercent < minProfitFloat {
				continue
			}
			if exchangeBuy != "" && arb.ExchangeBuy != exchangeBuy {
				continue
			}
			if exchangeSell != "" && arb.ExchangeSell != exchangeSell {
				continue
			}
			filteredArbs = append(filteredArbs, arb)
		}
	}

	// Count total
	total, _ := h.arbRepo.CountAll()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"arbitrage_opportunities": filteredArbs,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetArbitrage повертає конкретний arbitrage opportunity
func (h *ArbitrageHandler) GetArbitrage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid arbitrage ID")
		return
	}

	arb, err := h.arbRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "Arbitrage opportunity not found")
		return
	}

	respondJSON(w, http.StatusOK, arb)
}

// GetArbitrageStats повертає статистику arbitrage
func (h *ArbitrageHandler) GetArbitrageStats(w http.ResponseWriter, r *http.Request) {
	// Count active arbitrage opportunities
	activeCount, _ := h.arbRepo.CountActive()

	// Count total
	totalCount, _ := h.arbRepo.CountAll()

	// Get latest opportunities for stats
	latest, _ := h.arbRepo.List(10, 0)

	stats := map[string]interface{}{
		"active_count": activeCount,
		"total_count":  totalCount,
		"latest_count": len(latest),
	}

	// Calculate average profit if we have data
	if len(latest) > 0 {
		totalProfit := 0.0
		for _, arb := range latest {
			totalProfit += arb.NetProfitPercent
		}
		stats["average_profit_percent"] = totalProfit / float64(len(latest))
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetExchangeStatus повертає статус підключених бірж
// TODO: This requires access to the arbitrage detector which runs in the bot
// For now, return placeholder data
func (h *ArbitrageHandler) GetExchangeStatus(w http.ResponseWriter, r *http.Request) {
	// This would require access to the running arbitrage detector
	// which is in the bot process, not the API process
	// For now, return basic info

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Exchange status requires integration with bot process",
		"note":    "This endpoint will be implemented when API and bot are integrated",
	})
}
