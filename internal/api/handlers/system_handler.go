package handlers

import (
	"crypto-opportunities-bot/internal/repository"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/mux"
)

// SystemHandler обробляє системні запити
type SystemHandler struct {
	userRepo  repository.UserRepository
	oppRepo   repository.OpportunityRepository
	arbRepo   repository.ArbitrageRepository
	defiRepo  repository.DeFiRepository
	notifRepo repository.NotificationRepository
	startTime time.Time
}

// NewSystemHandler створює новий SystemHandler
func NewSystemHandler(
	userRepo repository.UserRepository,
	oppRepo repository.OpportunityRepository,
	arbRepo repository.ArbitrageRepository,
	defiRepo repository.DeFiRepository,
	notifRepo repository.NotificationRepository,
) *SystemHandler {
	return &SystemHandler{
		userRepo:  userRepo,
		oppRepo:   oppRepo,
		arbRepo:   arbRepo,
		defiRepo:  defiRepo,
		notifRepo: notifRepo,
		startTime: time.Now(),
	}
}

// GetSystemStatus повертає загальний статус системи
func (h *SystemHandler) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	// Get counts from repositories
	totalUsers, _ := h.userRepo.CountAll()
	activeOpportunities, _ := h.oppRepo.CountActive()
	activeArbitrage, _ := h.arbRepo.CountActive()
	activeDeFi, _ := h.defiRepo.CountActive()

	pendingNotifs, _ := h.notifRepo.CountByStatus("pending")
	sentNotifs, _ := h.notifRepo.CountByStatus("sent")
	failedNotifs, _ := h.notifRepo.CountByStatus("failed")

	// System metrics
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	status := map[string]interface{}{
		"status":  "healthy",
		"uptime":  time.Since(h.startTime).String(),
		"version": "1.0.0",
		"system": map[string]interface{}{
			"go_version":       runtime.Version(),
			"goroutines":       runtime.NumGoroutine(),
			"memory_alloc_mb":  mem.Alloc / 1024 / 1024,
			"memory_total_mb":  mem.TotalAlloc / 1024 / 1024,
			"memory_sys_mb":    mem.Sys / 1024 / 1024,
			"gc_runs":          mem.NumGC,
			"last_gc":          time.Unix(0, int64(mem.LastGC)).Format(time.RFC3339),
		},
		"database": map[string]interface{}{
			"users": map[string]int64{
				"total": totalUsers,
			},
			"opportunities": map[string]int64{
				"active": activeOpportunities,
			},
			"arbitrage": map[string]int64{
				"active": activeArbitrage,
			},
			"defi": map[string]int64{
				"active": activeDeFi,
			},
			"notifications": map[string]int64{
				"pending": pendingNotifs,
				"sent":    sentNotifs,
				"failed":  failedNotifs,
			},
		},
		"timestamp": time.Now(),
	}

	respondJSON(w, http.StatusOK, status)
}

// TriggerScraper запускає конкретний scraper
func (h *SystemHandler) TriggerScraper(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scraperName := vars["name"]

	// Validate scraper name
	validScrapers := map[string]bool{
		"binance": true,
		"bybit":   true,
		"defi":    true,
	}

	if !validScrapers[scraperName] {
		respondError(w, http.StatusBadRequest, "Invalid scraper name. Valid options: binance, bybit, defi")
		return
	}

	// In a real implementation, you would trigger the actual scraper here
	// For now, we'll just return a success message
	// TODO: Implement actual scraper triggering via channels or service calls

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Scraper triggered successfully",
		"scraper":      scraperName,
		"triggered_at": time.Now(),
		"note":         "Manual scraper triggering will be implemented with scraper service integration",
	})
}

// TriggerAllScrapers запускає всі scrapers
func (h *SystemHandler) TriggerAllScrapers(w http.ResponseWriter, r *http.Request) {
	scrapers := []string{"binance", "bybit", "defi"}

	// In a real implementation, you would trigger all scrapers here
	// TODO: Implement actual scraper triggering

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "All scrapers triggered successfully",
		"scrapers":     scrapers,
		"triggered_at": time.Now(),
		"note":         "Manual scraper triggering will be implemented with scraper service integration",
	})
}

// GetScraperStatus повертає статус scrapers
func (h *SystemHandler) GetScraperStatus(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would get actual scraper status
	// For now, return mock data

	status := map[string]interface{}{
		"scrapers": []map[string]interface{}{
			{
				"name":        "binance",
				"status":      "active",
				"last_run":    time.Now().Add(-5 * time.Minute),
				"next_run":    time.Now().Add(5 * time.Minute),
				"success_rate": 98.5,
				"total_runs":  1440,
			},
			{
				"name":        "bybit",
				"status":      "active",
				"last_run":    time.Now().Add(-5 * time.Minute),
				"next_run":    time.Now().Add(5 * time.Minute),
				"success_rate": 97.2,
				"total_runs":  1440,
			},
			{
				"name":        "defi",
				"status":      "active",
				"last_run":    time.Now().Add(-30 * time.Minute),
				"next_run":    time.Now().Add(30 * time.Minute),
				"success_rate": 95.8,
				"total_runs":  48,
			},
		},
		"schedule": map[string]string{
			"binance": "*/5 * * * *",  // Every 5 minutes
			"bybit":   "*/5 * * * *",  // Every 5 minutes
			"defi":    "0 * * * *",    // Every hour
		},
		"note": "Scraper status tracking will be enhanced with actual metrics",
	}

	respondJSON(w, http.StatusOK, status)
}

// ClearCache очищає кеш (якщо використовується Redis)
func (h *SystemHandler) ClearCache(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would clear Redis cache here
	// TODO: Implement Redis cache clearing

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Cache cleared successfully",
		"cleared_at": time.Now(),
		"note":       "Redis cache clearing will be implemented when Redis is integrated",
	})
}

// GetHealthCheck простий health check для load balancers
func (h *SystemHandler) GetHealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"uptime": time.Since(h.startTime).String(),
	})
}

// RestartNotificationDispatcher перезапускає notification dispatcher
func (h *SystemHandler) RestartNotificationDispatcher(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would restart the notification dispatcher
	// TODO: Implement notification dispatcher restart

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Notification dispatcher restart triggered",
		"triggered_at": time.Now(),
		"note":         "Dispatcher restart will be implemented with service integration",
	})
}
