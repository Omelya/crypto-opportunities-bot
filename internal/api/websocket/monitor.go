package websocket

import (
	"crypto-opportunities-bot/internal/repository"
	"runtime"
	"time"
)

// MonitorService periodically broadcasts system metrics
type MonitorService struct {
	hub       *Hub
	userRepo  repository.UserRepository
	oppRepo   repository.OpportunityRepository
	arbRepo   repository.ArbitrageRepository
	defiRepo  repository.DeFiRepository
	notifRepo repository.NotificationRepository
	startTime time.Time
	ticker    *time.Ticker
	stopChan  chan bool
}

// NewMonitorService creates a new monitor service
func NewMonitorService(
	hub *Hub,
	userRepo repository.UserRepository,
	oppRepo repository.OpportunityRepository,
	arbRepo repository.ArbitrageRepository,
	defiRepo repository.DeFiRepository,
	notifRepo repository.NotificationRepository,
) *MonitorService {
	return &MonitorService{
		hub:       hub,
		userRepo:  userRepo,
		oppRepo:   oppRepo,
		arbRepo:   arbRepo,
		defiRepo:  defiRepo,
		notifRepo: notifRepo,
		startTime: time.Now(),
		stopChan:  make(chan bool),
	}
}

// Start begins broadcasting metrics
func (m *MonitorService) Start(interval time.Duration) {
	m.ticker = time.NewTicker(interval)
	go m.run()
}

// Stop halts the monitoring service
func (m *MonitorService) Stop() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
	m.stopChan <- true
}

// run is the main monitoring loop
func (m *MonitorService) run() {
	for {
		select {
		case <-m.ticker.C:
			m.broadcastMetrics()
		case <-m.stopChan:
			return
		}
	}
}

// broadcastMetrics collects and broadcasts system metrics
func (m *MonitorService) broadcastMetrics() {
	// Only broadcast if there are connected clients
	if m.hub.GetClientCount() == 0 {
		return
	}

	metrics := m.collectMetrics()
	m.hub.BroadcastSystemMetrics(metrics)
}

// collectMetrics gathers current system metrics
func (m *MonitorService) collectMetrics() map[string]interface{} {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// Collect database stats
	totalUsers, _ := m.userRepo.CountAll()
	activeOpps, _ := m.oppRepo.CountActive()
	activeArb, _ := m.arbRepo.CountActive()
	activeDeFi, _ := m.defiRepo.CountActive()

	pendingNotifs, _ := m.notifRepo.CountByStatus("pending")
	sentNotifs, _ := m.notifRepo.CountByStatus("sent")
	failedNotifs, _ := m.notifRepo.CountByStatus("failed")

	return map[string]interface{}{
		"timestamp": time.Now(),
		"uptime":    time.Since(m.startTime).String(),
		"system": map[string]interface{}{
			"goroutines":      runtime.NumGoroutine(),
			"memory_alloc_mb": mem.Alloc / 1024 / 1024,
			"memory_sys_mb":   mem.Sys / 1024 / 1024,
			"gc_runs":         mem.NumGC,
		},
		"database": map[string]interface{}{
			"users":         totalUsers,
			"opportunities": activeOpps,
			"arbitrage":     activeArb,
			"defi":          activeDeFi,
		},
		"notifications": map[string]interface{}{
			"pending": pendingNotifs,
			"sent":    sentNotifs,
			"failed":  failedNotifs,
		},
		"websocket": map[string]interface{}{
			"connected_clients": m.hub.GetClientCount(),
		},
	}
}

// BroadcastNotificationCreated broadcasts when a new notification is created
func (m *MonitorService) BroadcastNotificationCreated(notification interface{}) {
	m.hub.BroadcastNotification("created", notification)
}

// BroadcastNotificationSent broadcasts when a notification is sent
func (m *MonitorService) BroadcastNotificationSent(notification interface{}) {
	m.hub.BroadcastNotification("sent", notification)
}

// BroadcastNotificationFailed broadcasts when a notification fails
func (m *MonitorService) BroadcastNotificationFailed(notification interface{}) {
	m.hub.BroadcastNotification("failed", notification)
}

// BroadcastScraperStarted broadcasts when a scraper starts
func (m *MonitorService) BroadcastScraperStarted(scraperName string) {
	m.hub.BroadcastScraperEvent("started", map[string]interface{}{
		"scraper":    scraperName,
		"started_at": time.Now(),
	})
}

// BroadcastScraperCompleted broadcasts when a scraper completes
func (m *MonitorService) BroadcastScraperCompleted(scraperName string, results interface{}) {
	m.hub.BroadcastScraperEvent("completed", map[string]interface{}{
		"scraper":      scraperName,
		"completed_at": time.Now(),
		"results":      results,
	})
}

// BroadcastScraperFailed broadcasts when a scraper fails
func (m *MonitorService) BroadcastScraperFailed(scraperName string, err error) {
	m.hub.BroadcastScraperEvent("failed", map[string]interface{}{
		"scraper":  scraperName,
		"error":    err.Error(),
		"failed_at": time.Now(),
	})
}

// BroadcastOpportunityCreated broadcasts when a new opportunity is detected
func (m *MonitorService) BroadcastOpportunityCreated(opportunity interface{}) {
	m.hub.BroadcastOpportunity("created", opportunity)
}

// BroadcastOpportunityUpdated broadcasts when an opportunity is updated
func (m *MonitorService) BroadcastOpportunityUpdated(opportunity interface{}) {
	m.hub.BroadcastOpportunity("updated", opportunity)
}

// BroadcastOpportunityExpired broadcasts when an opportunity expires
func (m *MonitorService) BroadcastOpportunityExpired(opportunity interface{}) {
	m.hub.BroadcastOpportunity("expired", opportunity)
}

// BroadcastUserRegistered broadcasts when a new user registers
func (m *MonitorService) BroadcastUserRegistered(user interface{}) {
	m.hub.BroadcastUserAction("registered", user)
}

// BroadcastUserSubscribed broadcasts when a user subscribes to premium
func (m *MonitorService) BroadcastUserSubscribed(user interface{}) {
	m.hub.BroadcastUserAction("subscribed", user)
}
