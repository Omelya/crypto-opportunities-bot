package handlers

import (
	"crypto-opportunities-bot/internal/repository"
	"net/http"
)

// StatsHandler обробляє запити статистики
type StatsHandler struct {
	userRepo  repository.UserRepository
	oppRepo   repository.OpportunityRepository
	arbRepo   repository.ArbitrageRepository
	defiRepo  repository.DeFiRepository
	notifRepo repository.NotificationRepository
}

// NewStatsHandler створює новий StatsHandler
func NewStatsHandler(
	userRepo repository.UserRepository,
	oppRepo repository.OpportunityRepository,
	arbRepo repository.ArbitrageRepository,
	defiRepo repository.DeFiRepository,
	notifRepo repository.NotificationRepository,
) *StatsHandler {
	return &StatsHandler{
		userRepo:  userRepo,
		oppRepo:   oppRepo,
		arbRepo:   arbRepo,
		defiRepo:  defiRepo,
		notifRepo: notifRepo,
	}
}

// Dashboard повертає загальну статистику для дашборду
func (h *StatsHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Збираємо статистику з різних джерел
	totalUsers, _ := h.userRepo.CountAll()
	activeUsers, _ := h.userRepo.CountActive()
	premiumUsers, _ := h.userRepo.CountPremium()

	activeOpps, _ := h.oppRepo.CountActive()
	activeArbs, _ := h.arbRepo.CountActive()
	activeDeFi, _ := h.defiRepo.CountActive()

	pendingNotifs, _ := h.notifRepo.CountPending()
	sentNotifs, _ := h.notifRepo.CountSent()
	failedNotifs, _ := h.notifRepo.CountFailed()

	stats := map[string]interface{}{
		"users": map[string]interface{}{
			"total":   totalUsers,
			"active":  activeUsers,
			"premium": premiumUsers,
			"free":    totalUsers - premiumUsers,
		},
		"opportunities": map[string]interface{}{
			"active":    activeOpps,
			"arbitrage": activeArbs,
			"defi":      activeDeFi,
		},
		"notifications": map[string]interface{}{
			"pending": pendingNotifs,
			"sent":    sentNotifs,
			"failed":  failedNotifs,
			"total":   pendingNotifs + sentNotifs + failedNotifs,
		},
	}

	respondJSON(w, http.StatusOK, stats)
}

// UserStats повертає статистику користувачів
func (h *StatsHandler) UserStats(w http.ResponseWriter, r *http.Request) {
	totalUsers, _ := h.userRepo.CountAll()
	activeUsers, _ := h.userRepo.CountActive()
	premiumUsers, _ := h.userRepo.CountPremium()

	stats := map[string]interface{}{
		"total":   totalUsers,
		"active":  activeUsers,
		"premium": premiumUsers,
		"free":    totalUsers - premiumUsers,
		// TODO: Add more detailed stats
		// - Users by capital range
		// - Users by risk profile
		// - Growth over time
		// - Retention rates
	}

	respondJSON(w, http.StatusOK, stats)
}
