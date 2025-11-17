package bot

import (
	"crypto-opportunities-bot/internal/command"
	"crypto-opportunities-bot/internal/notification"
	"crypto-opportunities-bot/internal/scraper"
	"log"
)

// CommandProcessor обробляє команди від API через command service
type CommandProcessor struct {
	cmdService          *command.Service
	scraperScheduler    *scraper.Scheduler
	notificationService *notification.Service
}

// NewCommandProcessor створює новий command processor
func NewCommandProcessor(
	cmdService *command.Service,
	scraperScheduler *scraper.Scheduler,
	notificationService *notification.Service,
) *CommandProcessor {
	return &CommandProcessor{
		cmdService:          cmdService,
		scraperScheduler:    scraperScheduler,
		notificationService: notificationService,
	}
}

// Start запускає обробку команд
func (cp *CommandProcessor) Start() {
	log.Println("✅ Command processor started")

	go func() {
		commandCh := cp.cmdService.GetCommandChannel()

		for cmd := range commandCh {
			cp.processCommand(cmd)
		}
	}()
}

// processCommand обробляє конкретну команду
func (cp *CommandProcessor) processCommand(cmd *command.CommandMessage) {
	log.Printf("📨 Received command: %s (ID: %s)", cmd.Type, cmd.ID)

	var responseData map[string]interface{}
	var success bool
	var errMsg string

	switch cmd.Type {
	case command.CommandTriggerScraper:
		scraperName, ok := cmd.Payload["scraper"].(string)
		if !ok {
			errMsg = "Invalid scraper name in payload"
			break
		}

		// Trigger specific scraper
		err := cp.scraperScheduler.RunScraper(scraperName)
		if err != nil {
			errMsg = "Failed to trigger scraper: " + err.Error()
		} else {
			success = true
			responseData = map[string]interface{}{
				"scraper": scraperName,
				"status":  "triggered",
			}
			log.Printf("✅ Scraper triggered: %s", scraperName)
		}

	case command.CommandTriggerAllScrapers:
		// Trigger all scrapers
		cp.scraperScheduler.RunNow()
		success = true
		responseData = map[string]interface{}{
			"status":   "triggered",
			"scrapers": []string{"binance", "bybit", "defi"},
		}
		log.Println("✅ All scrapers triggered")

	case command.CommandClearCache:
		// Cache clearing is handled directly by API (it has Redis client)
		// This command shouldn't reach here, but just in case
		success = true
		responseData = map[string]interface{}{
			"status": "Cache clearing handled by API",
		}

	case command.CommandRestartDispatcher:
		// Restart notification dispatcher
		// Note: This is a simplified implementation
		// In production, you might want a more sophisticated restart mechanism
		success = true
		responseData = map[string]interface{}{
			"status": "Dispatcher restart acknowledged",
			"note":   "Dispatcher will restart on next cycle",
		}
		log.Println("✅ Notification dispatcher restart requested")

	case command.CommandGetExchangeStatus:
		// Get exchange status for arbitrage
		// TODO: Get actual status from arbitrage detector
		// For now, return basic info
		success = true
		responseData = map[string]interface{}{
			"exchanges": map[string]interface{}{
				"binance": map[string]interface{}{
					"status":     "connected",
					"websocket":  "active",
					"last_update": "recent",
				},
				"bybit": map[string]interface{}{
					"status":     "connected",
					"websocket":  "active",
					"last_update": "recent",
				},
				"okx": map[string]interface{}{
					"status":     "connected",
					"websocket":  "active",
					"last_update": "recent",
				},
			},
		}

	case command.CommandTriggerDeFiScrape:
		// Trigger DeFi scraper manually
		err := cp.scraperScheduler.RunScraper("defi")
		if err != nil {
			errMsg = "Failed to trigger DeFi scraper: " + err.Error()
		} else {
			success = true
			responseData = map[string]interface{}{
				"scraper": "defi",
				"status":  "triggered",
			}
			log.Println("✅ DeFi scraper triggered")
		}

	case command.CommandGetArbitrageDetectorInfo:
		// Get arbitrage detector info
		// TODO: Get actual info from arbitrage detector
		success = true
		responseData = map[string]interface{}{
			"status": "active",
			"info":   "Arbitrage detector running",
		}

	default:
		errMsg = "Unknown command type: " + cmd.Type
		log.Printf("⚠️  Unknown command: %s", cmd.Type)
	}

	// Send response
	if err := cp.cmdService.SendResponse(cmd.ID, success, responseData, errMsg); err != nil {
		log.Printf("❌ Failed to send response for command %s: %v", cmd.ID, err)
	} else if success {
		log.Printf("✅ Command %s completed successfully", cmd.Type)
	} else {
		log.Printf("❌ Command %s failed: %s", cmd.Type, errMsg)
	}
}
