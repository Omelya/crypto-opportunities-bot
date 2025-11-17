package handlers

import (
	"crypto-opportunities-bot/internal/repository"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// DeFiHandler обробляє запити пов'язані з DeFi opportunities
type DeFiHandler struct {
	defiRepo repository.DeFiRepository
}

// NewDeFiHandler створює новий DeFiHandler
func NewDeFiHandler(defiRepo repository.DeFiRepository) *DeFiHandler {
	return &DeFiHandler{
		defiRepo: defiRepo,
	}
}

// ListDeFi повертає список DeFi opportunities
func (h *DeFiHandler) ListDeFi(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Parse filters
	chain := r.URL.Query().Get("chain")             // Ethereum, BSC, Polygon
	protocol := r.URL.Query().Get("protocol")       // Uniswap, Aave, Curve
	minAPY := r.URL.Query().Get("min_apy")          // minimum APY %
	riskLevel := r.URL.Query().Get("risk_level")    // low, medium, high

	// Fetch DeFi opportunities
	defis, err := h.defiRepo.List(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch DeFi opportunities")
		return
	}

	// Apply filters
	filteredDefis := defis
	if chain != "" || protocol != "" || minAPY != "" || riskLevel != "" {
		filteredDefis = make([]*repository.DeFiOpportunity, 0)
		minAPYFloat := 0.0
		if minAPY != "" {
			minAPYFloat, _ = strconv.ParseFloat(minAPY, 64)
		}

		for _, defi := range defis {
			if chain != "" && defi.Chain != chain {
				continue
			}
			if protocol != "" && defi.Protocol != protocol {
				continue
			}
			if minAPY != "" && defi.APY < minAPYFloat {
				continue
			}
			if riskLevel != "" && defi.RiskLevel != riskLevel {
				continue
			}
			filteredDefis = append(filteredDefis, defi)
		}
	}

	// Count total
	total, _ := h.defiRepo.CountAll()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"defi_opportunities": filteredDefis,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetDeFi повертає конкретний DeFi opportunity
func (h *DeFiHandler) GetDeFi(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid DeFi ID")
		return
	}

	defi, err := h.defiRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "DeFi opportunity not found")
		return
	}

	respondJSON(w, http.StatusOK, defi)
}

// GetDeFiStats повертає статистику DeFi
func (h *DeFiHandler) GetDeFiStats(w http.ResponseWriter, r *http.Request) {
	// Count active DeFi opportunities
	activeCount, _ := h.defiRepo.CountActive()

	// Count total
	totalCount, _ := h.defiRepo.CountAll()

	// Get top by APY for stats
	topByAPY, _ := h.defiRepo.GetTopByAPY(10)

	stats := map[string]interface{}{
		"active_count": activeCount,
		"total_count":  totalCount,
		"top_count":    len(topByAPY),
	}

	// Calculate average APY if we have data
	if len(topByAPY) > 0 {
		totalAPY := 0.0
		for _, defi := range topByAPY {
			totalAPY += defi.APY
		}
		stats["average_apy"] = totalAPY / float64(len(topByAPY))
		stats["max_apy"] = topByAPY[0].APY
	}

	respondJSON(w, http.StatusOK, stats)
}

// TriggerDeFiScrape запускає scraping DeFi вручну
// TODO: This requires access to the DeFi scraper which runs in the bot
func (h *DeFiHandler) TriggerDeFiScrape(w http.ResponseWriter, r *http.Request) {
	// This would require access to the running DeFi scraper
	// which is in the bot process, not the API process

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Manual DeFi scraping requires integration with bot process",
		"note":    "This endpoint will be implemented when API and bot are integrated",
	})
}

// GetProtocols повертає список протоколів
func (h *DeFiHandler) GetProtocols(w http.ResponseWriter, r *http.Request) {
	protocols, err := h.defiRepo.GetUniqueProtocols()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch protocols")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"protocols": protocols,
		"count":     len(protocols),
	})
}

// GetChains повертає список chains
func (h *DeFiHandler) GetChains(w http.ResponseWriter, r *http.Request) {
	chains, err := h.defiRepo.GetUniqueChains()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch chains")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"chains": chains,
		"count":  len(chains),
	})
}
