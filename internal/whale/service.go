package whale

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"fmt"
	"log"
	"strings"
	"time"
)

type Service struct {
	whaleRepo      repository.WhaleRepository
	clients        []BlockchainClient
	minUSD         float64 // Minimum transaction size in USD
	priceCache     map[string]float64 // Token -> USD price cache
	onWhaleDetected func(*models.WhaleTransaction) // Callback when new whale is detected
}

type Config struct {
	MinTransactionUSD float64
	EtherscanAPIKey   string
	BSCScanAPIKey     string
	Chains            []string
}

func NewService(whaleRepo repository.WhaleRepository, cfg *Config) *Service {
	service := &Service{
		whaleRepo:  whaleRepo,
		clients:    []BlockchainClient{},
		minUSD:     cfg.MinTransactionUSD,
		priceCache: make(map[string]float64),
	}

	// Initialize blockchain clients based on config
	for _, chain := range cfg.Chains {
		switch chain {
		case "ethereum":
			if cfg.EtherscanAPIKey != "" {
				service.clients = append(service.clients, NewEtherscanClient(cfg.EtherscanAPIKey))
				log.Printf("✅ Whale Watcher: Ethereum client initialized")
			}
		case "bsc":
			if cfg.BSCScanAPIKey != "" {
				service.clients = append(service.clients, NewBSCScanClient(cfg.BSCScanAPIKey))
				log.Printf("✅ Whale Watcher: BSC client initialized")
			}
		}
	}

	// Initialize price cache with approximate prices
	service.priceCache["ETH"] = 2400.0
	service.priceCache["BNB"] = 350.0
	service.priceCache["BTC"] = 45000.0
	service.priceCache["USDT"] = 1.0
	service.priceCache["USDC"] = 1.0

	return service
}

// ScanAll scans all configured blockchains for whale transactions
func (s *Service) ScanAll() ([]*models.WhaleTransaction, error) {
	var allWhales []*models.WhaleTransaction

	for _, client := range s.clients {
		whales, err := s.scanChain(client)
		if err != nil {
			log.Printf("⚠️ Error scanning %s: %v", client.GetChain(), err)
			continue
		}

		allWhales = append(allWhales, whales...)
		log.Printf("✅ Scanned %s: found %d whale transactions", client.GetChain(), len(whales))
	}

	return allWhales, nil
}

// scanChain scans a single blockchain for whale transactions
func (s *Service) scanChain(client BlockchainClient) ([]*models.WhaleTransaction, error) {
	transactions, err := client.GetRecentTransactions(s.minUSD)
	if err != nil {
		return nil, err
	}

	var whales []*models.WhaleTransaction

	for _, tx := range transactions {
		// Skip if already in database
		existing, _ := s.whaleRepo.GetByTxHash(tx.Hash)
		if existing != nil {
			continue
		}

		// Calculate USD value
		price, exists := s.priceCache[tx.Token]
		if !exists {
			price = 1.0 // Default for unknown tokens
		}
		amountUSD := tx.ValueDecimal * price

		// Skip if below threshold
		if amountUSD < s.minUSD {
			continue
		}

		// Determine direction
		direction := s.determineDirection(tx.From, tx.To)

		// Create whale transaction
		whale := &models.WhaleTransaction{
			Chain:          client.GetChain(),
			TxHash:         tx.Hash,
			Token:          tx.Token,
			TokenAddress:   tx.TokenAddress,
			Amount:         tx.ValueDecimal,
			AmountUSD:      amountUSD,
			FromAddress:    tx.From,
			ToAddress:      tx.To,
			Direction:      direction,
			BlockNumber:    tx.BlockNumber,
			BlockTimestamp: tx.BlockTimestamp,
			GasUsed:        tx.GasUsed,
			GasPrice:       tx.GasPrice,
			Status:         models.WhaleStatusNew,
			IsNotified:     false,
			ExplorerURL:    s.getExplorerURL(client.GetChain(), tx.Hash),
		}

		// Add labels for known addresses
		if label, exists := GetAddressLabel(strings.ToLower(tx.From)); exists {
			whale.FromLabel = label
		}
		if label, exists := GetAddressLabel(strings.ToLower(tx.To)); exists {
			whale.ToLabel = label
		}

		// Save to database
		if err := s.whaleRepo.Create(whale); err != nil {
			log.Printf("❌ Failed to save whale transaction: %v", err)
			continue
		}

		whales = append(whales, whale)
		log.Printf("🐋 New whale detected: %s transferred %.2f %s ($%.2f)",
			s.shortenAddress(tx.From), tx.ValueDecimal, tx.Token, amountUSD)

		// Trigger callback if set
		if s.onWhaleDetected != nil {
			s.onWhaleDetected(whale)
		}
	}

	return whales, nil
}

// determineDirection determines the transaction direction based on known addresses
func (s *Service) determineDirection(from, to string) string {
	fromLower := strings.ToLower(from)
	toLower := strings.ToLower(to)

	fromIsExchange := IsExchangeAddress(fromLower)
	toIsExchange := IsExchangeAddress(toLower)

	if fromIsExchange && !toIsExchange {
		return models.WhaleDirectionExchangeToWallet // Potential accumulation
	} else if !fromIsExchange && toIsExchange {
		return models.WhaleDirectionWalletToExchange // Potential sell
	} else if !fromIsExchange && !toIsExchange {
		return models.WhaleDirectionWalletToWallet // Whale transfer
	}

	return models.WhaleDirectionUnknown
}

// getExplorerURL returns the blockchain explorer URL for a transaction
func (s *Service) getExplorerURL(chain, txHash string) string {
	switch chain {
	case "ethereum":
		return fmt.Sprintf("https://etherscan.io/tx/%s", txHash)
	case "bsc":
		return fmt.Sprintf("https://bscscan.com/tx/%s", txHash)
	case "polygon":
		return fmt.Sprintf("https://polygonscan.com/tx/%s", txHash)
	case "arbitrum":
		return fmt.Sprintf("https://arbiscan.io/tx/%s", txHash)
	case "optimism":
		return fmt.Sprintf("https://optimistic.etherscan.io/tx/%s", txHash)
	default:
		return ""
	}
}

// shortenAddress shortens an address for display
func (s *Service) shortenAddress(address string) string {
	if len(address) <= 10 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}

// GetPendingNotifications returns whale transactions that need to be notified
func (s *Service) GetPendingNotifications() ([]*models.WhaleTransaction, error) {
	return s.whaleRepo.GetPendingNotifications()
}

// MarkAsNotified marks a whale transaction as notified
func (s *Service) MarkAsNotified(id uint) error {
	return s.whaleRepo.MarkAsNotified(id)
}

// GetStats24h returns whale statistics for the last 24 hours
func (s *Service) GetStats24h(chain, token string) (*models.WhaleStats, error) {
	return s.whaleRepo.GetStats24h(chain, token)
}

// GetRecent returns recent whale transactions
func (s *Service) GetRecent(limit int) ([]*models.WhaleTransaction, error) {
	return s.whaleRepo.GetRecent(limit)
}

// GetRecentByChain returns recent whale transactions for a specific chain
func (s *Service) GetRecentByChain(chain string, limit int) ([]*models.WhaleTransaction, error) {
	return s.whaleRepo.GetRecentByChain(chain, limit)
}

// GetRecentByToken returns recent whale transactions for a specific token
func (s *Service) GetRecentByToken(token string, limit int) ([]*models.WhaleTransaction, error) {
	return s.whaleRepo.GetRecentByToken(token, limit)
}

// CleanupOld removes old whale transactions
func (s *Service) CleanupOld(daysOld int) error {
	return s.whaleRepo.CleanupOld(daysOld)
}

// UpdatePriceCache updates the price cache with current prices
func (s *Service) UpdatePriceCache(token string, priceUSD float64) {
	s.priceCache[token] = priceUSD
}

// GetTopTokens24h returns the most active tokens in the last 24 hours
func (s *Service) GetTopTokens24h(limit int) ([]string, error) {
	return s.whaleRepo.GetTopTokens24h(limit)
}

// FormatWhaleMessage formats a whale transaction for notification
func (s *Service) FormatWhaleMessage(whale *models.WhaleTransaction) string {
	var msg strings.Builder

	// Header with emoji based on size
	if whale.IsMegaWhale() {
		msg.WriteString("🐋🐋 MEGA WHALE ALERT 🐋🐋\n\n")
	} else if whale.IsLargeWhale() {
		msg.WriteString("🐋 LARGE WHALE ALERT 🐋\n\n")
	} else {
		msg.WriteString("🐋 Whale Transaction Detected\n\n")
	}

	// Transaction details
	msg.WriteString(fmt.Sprintf("💰 *Amount:* %.2f %s ($%.2f)\n", whale.Amount, whale.Token, whale.AmountUSD))
	msg.WriteString(fmt.Sprintf("⛓️ *Chain:* %s\n", strings.Title(whale.Chain)))
	msg.WriteString(fmt.Sprintf("%s *Direction:* %s\n\n", whale.GetDirectionEmoji(), whale.Direction))

	// Signal interpretation
	msg.WriteString(fmt.Sprintf("📊 *Signal:* %s\n\n", whale.GetSignalInterpretation()))

	// Addresses
	msg.WriteString("📍 *From:* ")
	if whale.FromLabel != "" {
		msg.WriteString(fmt.Sprintf("%s\n", whale.FromLabel))
	} else {
		msg.WriteString(fmt.Sprintf("`%s`\n", s.shortenAddress(whale.FromAddress)))
	}

	msg.WriteString("📍 *To:* ")
	if whale.ToLabel != "" {
		msg.WriteString(fmt.Sprintf("%s\n\n", whale.ToLabel))
	} else {
		msg.WriteString(fmt.Sprintf("`%s`\n\n", s.shortenAddress(whale.ToAddress)))
	}

	// Time
	msg.WriteString(fmt.Sprintf("⏰ *Time:* %s\n", whale.GetTimeAgo()))

	// Explorer link
	if whale.ExplorerURL != "" {
		msg.WriteString(fmt.Sprintf("\n[View on Explorer](%s)", whale.ExplorerURL))
	}

	return msg.String()
}

// OnWhaleDetected sets the callback function for when a new whale is detected
func (s *Service) OnWhaleDetected(callback func(*models.WhaleTransaction)) {
	s.onWhaleDetected = callback
}
