package handlers

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// OpportunityHandler обробляє запити пов'язані з opportunities
type OpportunityHandler struct {
	oppRepo repository.OpportunityRepository
}

// NewOpportunityHandler створює новий OpportunityHandler
func NewOpportunityHandler(oppRepo repository.OpportunityRepository) *OpportunityHandler {
	return &OpportunityHandler{
		oppRepo: oppRepo,
	}
}

// ListOpportunities повертає список opportunities з фільтрами
func (h *OpportunityHandler) ListOpportunities(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Parse filters
	exchange := r.URL.Query().Get("exchange")      // binance, bybit, okx
	oppType := r.URL.Query().Get("type")           // launchpool, airdrop, staking, etc.
	isActive := r.URL.Query().Get("is_active")     // true, false

	// Fetch opportunities
	opps, err := h.oppRepo.List(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch opportunities")
		return
	}

	// Apply filters (TODO: move to repository query)
	filteredOpps := opps
	if exchange != "" || oppType != "" || isActive != "" {
		filteredOpps = make([]*models.Opportunity, 0)
		for _, opp := range opps {
			if exchange != "" && opp.Exchange != exchange {
				continue
			}
			if oppType != "" && opp.Type != oppType {
				continue
			}
			if isActive != "" {
				activeFilter, _ := strconv.ParseBool(isActive)
				if opp.IsActive != activeFilter {
					continue
				}
			}
			filteredOpps = append(filteredOpps, opp)
		}
	}

	// Count total
	total, _ := h.oppRepo.CountAll()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"opportunities": filteredOpps,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetOpportunity повертає конкретний opportunity
func (h *OpportunityHandler) GetOpportunity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid opportunity ID")
		return
	}

	opp, err := h.oppRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "Opportunity not found")
		return
	}

	respondJSON(w, http.StatusOK, opp)
}

// CreateOpportunity створює новий opportunity вручну
func (h *OpportunityHandler) CreateOpportunity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExternalID   string    `json:"external_id"`
		Exchange     string    `json:"exchange"`
		Type         string    `json:"type"`
		Title        string    `json:"title"`
		Description  string    `json:"description"`
		Reward       string    `json:"reward"`
		EstimatedROI float64   `json:"estimated_roi"`
		StartDate    *time.Time `json:"start_date"`
		EndDate      *time.Time `json:"end_date"`
		URL          string    `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Title == "" || req.Exchange == "" || req.Type == "" {
		respondError(w, http.StatusBadRequest, "Title, exchange, and type are required")
		return
	}

	// Create opportunity
	opp := &models.Opportunity{
		ExternalID:   req.ExternalID,
		Exchange:     req.Exchange,
		Type:         req.Type,
		Title:        req.Title,
		Description:  req.Description,
		Reward:       req.Reward,
		EstimatedROI: req.EstimatedROI,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		URL:          req.URL,
		IsActive:     true,
	}

	if err := h.oppRepo.Create(opp); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create opportunity")
		return
	}

	respondJSON(w, http.StatusCreated, opp)
}

// UpdateOpportunity оновлює opportunity
func (h *OpportunityHandler) UpdateOpportunity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid opportunity ID")
		return
	}

	opp, err := h.oppRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "Opportunity not found")
		return
	}

	var updateReq struct {
		Title        *string    `json:"title"`
		Description  *string    `json:"description"`
		Reward       *string    `json:"reward"`
		EstimatedROI *float64   `json:"estimated_roi"`
		EndDate      *time.Time `json:"end_date"`
		IsActive     *bool      `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Apply updates
	if updateReq.Title != nil {
		opp.Title = *updateReq.Title
	}
	if updateReq.Description != nil {
		opp.Description = *updateReq.Description
	}
	if updateReq.Reward != nil {
		opp.Reward = *updateReq.Reward
	}
	if updateReq.EstimatedROI != nil {
		opp.EstimatedROI = *updateReq.EstimatedROI
	}
	if updateReq.EndDate != nil {
		opp.EndDate = updateReq.EndDate
	}
	if updateReq.IsActive != nil {
		opp.IsActive = *updateReq.IsActive
	}

	if err := h.oppRepo.Update(opp); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update opportunity")
		return
	}

	respondJSON(w, http.StatusOK, opp)
}

// DeactivateOpportunity деактивує opportunity
func (h *OpportunityHandler) DeactivateOpportunity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid opportunity ID")
		return
	}

	opp, err := h.oppRepo.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "Opportunity not found")
		return
	}

	opp.IsActive = false
	if err := h.oppRepo.Update(opp); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to deactivate opportunity")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Opportunity deactivated successfully",
		"opportunity": opp,
	})
}

// DeleteOpportunity видаляє opportunity (soft delete)
func (h *OpportunityHandler) DeleteOpportunity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid opportunity ID")
		return
	}

	if err := h.oppRepo.Delete(uint(id)); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete opportunity")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Opportunity deleted successfully",
	})
}
