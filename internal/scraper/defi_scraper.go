package scraper

import (
	"crypto-opportunities-bot/internal/defi/defillama"
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"fmt"
	"log"
	"strings"
	"time"
)

// DeFiScraper scraper для DeFi opportunities
type DeFiScraper struct {
	client    *defillama.Client
	repo      repository.DeFiRepository
	config    DeFiScraperConfig
	callbacks []DeFiCallback
}

// DeFiScraperConfig конфігурація для DeFi scraper
type DeFiScraperConfig struct {
	Chains       []string
	Protocols    []string
	MinAPY       float64
	MinTVL       float64
	MaxIL        float64
	OnlyAudited  bool
	MinVolume24h float64
}

// DeFiCallback функція для обробки нових DeFi opportunities
type DeFiCallback func(opp *models.DeFiOpportunity)

// NewDeFiScraper створює новий DeFi scraper
func NewDeFiScraper(repo repository.DeFiRepository, config DeFiScraperConfig) *DeFiScraper {
	return &DeFiScraper{
		client:    defillama.NewClient(),
		repo:      repo,
		config:    config,
		callbacks: make([]DeFiCallback, 0),
	}
}

// GetExchange повертає назву "біржі" (для DeFi це "defi")
func (s *DeFiScraper) GetExchange() string {
	return "defi"
}

// ScrapeAll scrapes всі DeFi opportunities
func (s *DeFiScraper) ScrapeAll() ([]*models.Opportunity, error) {
	log.Printf("🌾 Starting DeFi scraping...")

	// Get pools з фільтрами
	filters := defillama.PoolFilters{
		Chains:       s.config.Chains,
		Protocols:    s.config.Protocols,
		MinAPY:       s.config.MinAPY,
		MinTVL:       s.config.MinTVL,
		MaxIL:        s.config.MaxIL,
		MinVolume24h: s.config.MinVolume24h,
	}

	pools, err := s.client.FilterPools(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	log.Printf("📊 Found %d DeFi pools matching criteria", len(pools))

	newCount := 0
	updatedCount := 0

	for _, pool := range pools {
		// Convert pool to DeFiOpportunity
		defiOpp := s.convertPoolToOpportunity(pool)

		// Check if exists
		existing, err := s.repo.GetByExternalID(defiOpp.ExternalID)
		if err == nil && existing != nil {
			// Update existing
			defiOpp.ID = existing.ID
			defiOpp.CreatedAt = existing.CreatedAt

			if err := s.repo.Update(defiOpp); err != nil {
				log.Printf("❌ Failed to update DeFi opportunity %s: %v", defiOpp.ExternalID, err)
				continue
			}
			updatedCount++
		} else {
			// Create new
			if err := s.repo.Create(defiOpp); err != nil {
				log.Printf("❌ Failed to create DeFi opportunity %s: %v", defiOpp.ExternalID, err)
				continue
			}
			newCount++

			// Trigger callbacks for new opportunities
			for _, callback := range s.callbacks {
				callback(defiOpp)
			}
		}
	}

	log.Printf("✅ DeFi scraping complete: %d new, %d updated", newCount, updatedCount)

	// Return empty array as we're not using Opportunity model for DeFi
	return []*models.Opportunity{}, nil
}

// convertPoolToOpportunity конвертує DeFiLlama Pool в DeFiOpportunity
func (s *DeFiScraper) convertPoolToOpportunity(pool defillama.Pool) *models.DeFiOpportunity {
	externalID := models.GenerateDeFiExternalID(pool.Project, pool.Chain, pool.PoolID)

	// Parse tokens from symbol
	token0, token1 := s.parseTokens(pool.Symbol)

	// Determine pool type
	poolType := s.determinePoolType(pool)

	// Calculate risk level
	riskLevel := s.calculateRiskLevel(pool)

	// Calculate daily return
	dailyReturn := pool.APY / 365.0

	// Determine audit status
	auditStatus := "unknown"
	if pool.PredictedClass == "stable" {
		auditStatus = "verified"
	}

	// Parse pool meta for URL
	poolURL := s.buildPoolURL(pool)

	defiOpp := &models.DeFiOpportunity{
		ExternalID:   externalID,
		Protocol:     pool.Project,
		Chain:        s.normalizeChain(pool.Chain),
		PoolID:       pool.PoolID,
		PoolName:     pool.Symbol,
		Token0:       token0,
		Token1:       token1,
		PoolType:     poolType,
		APY:          pool.APY,
		APR:          pool.APY, // Approximation
		APYBase:      pool.APYBase,
		APYReward:    pool.APYReward,
		DailyReturn:  dailyReturn,
		TVL:          pool.TVL,
		Volume24h:    pool.Volume1d,
		Volume7d:     pool.Volume7d,
		RiskLevel:    riskLevel,
		ILRisk:       pool.IL7d,
		ILRisk7d:     pool.IL7d,
		AuditStatus:  auditStatus,
		MinDeposit:   s.estimateMinDeposit(pool.TVL),
		LockPeriod:   0, // Default: no lock
		RewardTokens: pool.RewardTokens,
		PoolURL:      poolURL,
		ProtocolURL:  s.buildProtocolURL(pool.Project),
		PoolMeta: models.JSONMap{
			"stablecoin":     pool.Stablecoin,
			"il_risk_level":  pool.ILRisk,
			"predicted_class": pool.PredictedClass,
			"count":          pool.Count,
		},
		IsActive:    true,
		LastChecked: time.Now(),
	}

	return defiOpp
}

// parseTokens парсить токени з symbol
func (s *DeFiScraper) parseTokens(symbol string) (string, string) {
	// Common formats: "USDC-ETH", "USDC/ETH", "USDC-ETH-0.3%"
	symbol = strings.ReplaceAll(symbol, "/", "-")

	// Remove percentage info
	parts := strings.Split(symbol, " ")
	if len(parts) > 0 {
		symbol = parts[0]
	}

	// Split by dash
	tokens := strings.Split(symbol, "-")
	if len(tokens) >= 2 {
		return tokens[0], tokens[1]
	}

	// Single asset pool
	return symbol, ""
}

// determinePoolType визначає тип pool
func (s *DeFiScraper) determinePoolType(pool defillama.Pool) string {
	project := strings.ToLower(pool.Project)

	// Lending protocols
	if strings.Contains(project, "aave") || strings.Contains(project, "compound") {
		return "lending"
	}

	// Vaults
	if strings.Contains(project, "yearn") || strings.Contains(project, "beefy") {
		return "vault"
	}

	// Staking
	if strings.Contains(project, "staking") || pool.APYReward > pool.APYBase*2 {
		return "staking"
	}

	// Default: liquidity pool
	return "liquidity"
}

// calculateRiskLevel розраховує рівень ризику
func (s *DeFiScraper) calculateRiskLevel(pool defillama.Pool) string {
	score := 0

	// TVL weight (higher TVL = lower risk)
	if pool.TVL < 100000 {
		score += 3
	} else if pool.TVL < 1000000 {
		score += 2
	} else if pool.TVL < 10000000 {
		score += 1
	}

	// IL risk weight
	if pool.IL7d > 10 {
		score += 3
	} else if pool.IL7d > 5 {
		score += 2
	} else if pool.IL7d > 2 {
		score += 1
	}

	// APY weight (extremely high APY = higher risk)
	if pool.APY > 100 {
		score += 2
	} else if pool.APY > 50 {
		score += 1
	}

	// Stablecoin = lower risk
	if pool.Stablecoin {
		score -= 2
	}

	// Determine risk level
	if score <= 2 {
		return "low"
	} else if score <= 5 {
		return "medium"
	}
	return "high"
}

// estimateMinDeposit оцінює мінімальний депозит
func (s *DeFiScraper) estimateMinDeposit(tvl float64) float64 {
	// Estimate based on TVL
	if tvl > 100000000 {
		return 10 // $10 for large pools
	} else if tvl > 10000000 {
		return 50 // $50 for medium pools
	} else if tvl > 1000000 {
		return 100 // $100 for smaller pools
	}
	return 500 // $500 for very small pools
}

// normalizeChain нормалізує назву chain
func (s *DeFiScraper) normalizeChain(chain string) string {
	chain = strings.ToLower(chain)

	// Normalize common chains
	switch chain {
	case "eth", "ethereum":
		return "ethereum"
	case "bsc", "binance":
		return "bsc"
	case "matic", "polygon":
		return "polygon"
	case "arb", "arbitrum":
		return "arbitrum"
	case "op", "optimism":
		return "optimism"
	default:
		return chain
	}
}

// buildPoolURL будує URL для pool
func (s *DeFiScraper) buildPoolURL(pool defillama.Pool) string {
	project := strings.ToLower(pool.Project)
	chain := s.normalizeChain(pool.Chain)

	// Common URL patterns
	switch {
	case strings.Contains(project, "uniswap"):
		return fmt.Sprintf("https://app.uniswap.org/#/pool/%s", pool.PoolID)
	case strings.Contains(project, "pancakeswap"):
		return fmt.Sprintf("https://pancakeswap.finance/liquidity/%s", pool.PoolID)
	case strings.Contains(project, "aave"):
		return fmt.Sprintf("https://app.aave.com/?marketName=%s", chain)
	case strings.Contains(project, "curve"):
		return "https://curve.fi/#/ethereum/pools"
	default:
		return fmt.Sprintf("https://defillama.com/protocol/%s", pool.Project)
	}
}

// buildProtocolURL будує URL для protocol
func (s *DeFiScraper) buildProtocolURL(project string) string {
	return fmt.Sprintf("https://defillama.com/protocol/%s", project)
}

// OnNewDeFi реєструє callback для нових DeFi opportunities
func (s *DeFiScraper) OnNewDeFi(callback DeFiCallback) {
	s.callbacks = append(s.callbacks, callback)
}

// CleanupOld видаляє старі неактивні opportunities
func (s *DeFiScraper) CleanupOld(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	return s.repo.DeleteOld(cutoff)
}

// GetStats отримує статистику по DeFi opportunities
func (s *DeFiScraper) GetStats() (map[string]interface{}, error) {
	total, err := s.repo.CountActive()
	if err != nil {
		return nil, err
	}

	topAPY, err := s.repo.GetTopByAPY(1)
	if err != nil {
		return nil, err
	}

	topTVL, err := s.repo.GetTopByTVL(1)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_active": total,
		"last_updated": time.Now(),
	}

	if len(topAPY) > 0 {
		stats["max_apy"] = topAPY[0].APY
	}

	if len(topTVL) > 0 {
		stats["max_tvl"] = topTVL[0].TVL
	}

	return stats, nil
}
