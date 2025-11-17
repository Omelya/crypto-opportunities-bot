package handlers

import (
	"context"
	"crypto-opportunities-bot/internal/command"
	"crypto-opportunities-bot/internal/repository"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

// SystemHandler обробляє системні запити
type SystemHandler struct {
	userRepo   repository.UserRepository
	oppRepo    repository.OpportunityRepository
	arbRepo    repository.ArbitrageRepository
	defiRepo   repository.DeFiRepository
	notifRepo  repository.NotificationRepository
	cmdService *command.Service
	redisClient *redis.Client
	startTime  time.Time
}

// NewSystemHandler створює новий SystemHandler
func NewSystemHandler(
	userRepo repository.UserRepository,
	oppRepo repository.OpportunityRepository,
	arbRepo repository.ArbitrageRepository,
	defiRepo repository.DeFiRepository,
	notifRepo repository.NotificationRepository,
	cmdService *command.Service,
	redisClient *redis.Client,
) *SystemHandler {
	return &SystemHandler{
		userRepo:    userRepo,
		oppRepo:     oppRepo,
		arbRepo:     arbRepo,
		defiRepo:    defiRepo,
		notifRepo:   notifRepo,
		cmdService:  cmdService,
		redisClient: redisClient,
		startTime:   time.Now(),
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

	// If command service is not available, return error
	if h.cmdService == nil {
		respondError(w, http.StatusServiceUnavailable, "Command service not available (Redis required)")
		return
	}

	// Send command to bot process
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	payload := map[string]interface{}{
		"scraper": scraperName,
	}

	resp, err := h.cmdService.SendCommand(ctx, command.CommandTriggerScraper, payload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to trigger scraper: "+err.Error())
		return
	}

	if !resp.Success {
		respondError(w, http.StatusInternalServerError, "Scraper triggering failed: "+resp.Error)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Scraper triggered successfully",
		"scraper":      scraperName,
		"triggered_at": time.Now(),
		"result":       resp.Data,
	})
}

// TriggerAllScrapers запускає всі scrapers
func (h *SystemHandler) TriggerAllScrapers(w http.ResponseWriter, r *http.Request) {
	if h.cmdService == nil {
		respondError(w, http.StatusServiceUnavailable, "Command service not available (Redis required)")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	resp, err := h.cmdService.SendCommand(ctx, command.CommandTriggerAllScrapers, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to trigger scrapers: "+err.Error())
		return
	}

	if !resp.Success {
		respondError(w, http.StatusInternalServerError, "Scrapers triggering failed: "+resp.Error)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "All scrapers triggered successfully",
		"triggered_at": time.Now(),
		"result":       resp.Data,
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
	if h.redisClient == nil {
		respondError(w, http.StatusServiceUnavailable, "Redis not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Clear all cache keys (be careful in production!)
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = "cache:*" // Default pattern
	}

	iter := h.redisClient.Scan(ctx, 0, pattern, 0).Iterator()
	keysDeleted := 0

	for iter.Next(ctx) {
		if err := h.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			continue
		}
		keysDeleted++
	}

	if err := iter.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to clear cache: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Cache cleared successfully",
		"pattern":      pattern,
		"keys_deleted": keysDeleted,
		"cleared_at":   time.Now(),
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
	if h.cmdService == nil {
		respondError(w, http.StatusServiceUnavailable, "Command service not available (Redis required)")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.cmdService.SendCommand(ctx, command.CommandRestartDispatcher, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to restart dispatcher: "+err.Error())
		return
	}

	if !resp.Success {
		respondError(w, http.StatusInternalServerError, "Dispatcher restart failed: "+resp.Error)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Notification dispatcher restarted successfully",
		"triggered_at": time.Now(),
		"result":       resp.Data,
	})
}
