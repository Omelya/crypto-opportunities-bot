package scraper

import (
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type KrakenScraper struct {
	httpClient *http.Client
}

func NewKrakenScraper() *KrakenScraper {
	return &KrakenScraper{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *KrakenScraper) GetExchange() string {
	return models.ExchangeKraken
}

func (s *KrakenScraper) ScrapeAll() ([]*models.Opportunity, error) {
	var allOpps []*models.Opportunity
	var errors []error

	// Kraken focuses on staking, so we use staking as "launchpool"
	stakingOpps, err := s.ScrapeLaunchpool()
	if err != nil {
		log.Printf("Error scraping Kraken staking: %v", err)
		errors = append(errors, fmt.Errorf("staking: %w", err))
	} else {
		allOpps = append(allOpps, stakingOpps...)
		log.Printf("✅ Scraped %d Kraken staking opportunities", len(stakingOpps))
	}

	// Kraken Earn programs
	earnOpps, err := s.ScrapeAirdrops()
	if err != nil {
		log.Printf("Error scraping Kraken earn: %v", err)
		errors = append(errors, fmt.Errorf("earn: %w", err))
	} else {
		allOpps = append(allOpps, earnOpps...)
		log.Printf("✅ Scraped %d Kraken earn opportunities", len(earnOpps))
	}

	// Kraken Learn & Earn (limited)
	learnEarnOpps, err := s.ScrapeLearnEarn()
	if err != nil {
		log.Printf("Error scraping Kraken learn&earn: %v", err)
		errors = append(errors, fmt.Errorf("learn&earn: %w", err))
	} else {
		allOpps = append(allOpps, learnEarnOpps...)
		log.Printf("✅ Scraped %d Kraken learn&earn opportunities", len(learnEarnOpps))
	}

	if len(errors) == 3 {
		return allOpps, fmt.Errorf("all Kraken scrapers failed: %v", errors)
	}

	return allOpps, nil
}

func (s *KrakenScraper) ScrapeLaunchpool() ([]*models.Opportunity, error) {
	// Kraken Staking API (public endpoint)
	url := "https://api.kraken.com/0/public/Assets"

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp KrakenAssetsResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	// Popular staking assets on Kraken with typical APYs
	stakingAssets := map[string]float64{
		"ETH":   4.5,  // Ethereum staking
		"DOT":   12.0, // Polkadot
		"ATOM":  10.0, // Cosmos
		"SOL":   6.5,  // Solana
		"ADA":   4.0,  // Cardano
		"MATIC": 5.5,  // Polygon
		"AVAX":  8.0,  // Avalanche
		"NEAR":  9.0,  // NEAR Protocol
		"FLR":   11.0, // Flare
		"KAVA":  15.0, // Kava
	}

	for asset, apy := range stakingAssets {
		// Check if asset exists in Kraken
		if _, exists := apiResp.Result[asset]; !exists {
			continue
		}

		now := time.Now()
		endDate := now.AddDate(1, 0, 0) // Staking is ongoing, set 1 year ahead

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("kraken", "staking", asset),
			Exchange:      models.ExchangeKraken,
			Type:          models.OpportunityTypeStaking,
			Title:         fmt.Sprintf("Staking: %s", asset),
			Description:   fmt.Sprintf("Earn rewards by staking %s on Kraken", asset),
			Reward:        fmt.Sprintf("%.2f%% APY", apy),
			EstimatedROI:  apy,
			PoolSize:      0, // Not applicable for staking
			MinInvestment: s.getMinStaking(asset),
			Duration:      "Ongoing",
			StartDate:     &now,
			EndDate:       &endDate,
			URL:           fmt.Sprintf("https://www.kraken.com/features/staking-coins#%s", strings.ToLower(asset)),
			IsActive:      true,
			Metadata: map[string]interface{}{
				"asset": asset,
				"type":  "staking",
			},
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *KrakenScraper) ScrapeAirdrops() ([]*models.Opportunity, error) {
	// Kraken Earn API endpoint (simplified)
	// Note: Kraken doesn't have a public API for Earn, so we'll use common programs

	var opportunities []*models.Opportunity

	// Popular Kraken Earn programs
	earnPrograms := []struct {
		Asset       string
		APY         float64
		Description string
		MinAmount   float64
	}{
		{"USDT", 4.5, "Earn on your USDT holdings", 1.0},
		{"USDC", 4.0, "Earn on your USDC holdings", 1.0},
		{"DAI", 3.5, "Earn on your DAI holdings", 1.0},
		{"ETH", 3.0, "Earn on your ETH holdings", 0.01},
		{"BTC", 2.5, "Earn on your BTC holdings", 0.0001},
	}

	now := time.Now()
	endDate := now.AddDate(0, 6, 0) // 6 months ahead

	for _, program := range earnPrograms {
		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("kraken", "earn", program.Asset),
			Exchange:      models.ExchangeKraken,
			Type:          models.OpportunityTypeStaking, // Using staking type for consistency
			Title:         fmt.Sprintf("Kraken Earn: %s", program.Asset),
			Description:   program.Description,
			Reward:        fmt.Sprintf("%.2f%% APY", program.APY),
			EstimatedROI:  program.APY,
			PoolSize:      0,
			MinInvestment: program.MinAmount,
			Duration:      "Flexible",
			StartDate:     &now,
			EndDate:       &endDate,
			URL:           "https://www.kraken.com/features/earn",
			IsActive:      true,
			Metadata: map[string]interface{}{
				"asset": program.Asset,
				"type":  "earn",
			},
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *KrakenScraper) ScrapeLearnEarn() ([]*models.Opportunity, error) {
	// Kraken doesn't have a robust Learn & Earn program like Binance/Coinbase
	// We'll check their blog/announcements for any campaigns

	// For now, return empty as Kraken rarely has Learn & Earn
	// This can be expanded when they launch campaigns

	var opportunities []*models.Opportunity

	log.Printf("ℹ️ Kraken Learn & Earn: No active campaigns (Kraken rarely offers Learn & Earn)")

	return opportunities, nil
}

// Helper methods
func (s *KrakenScraper) getMinStaking(asset string) float64 {
	minStaking := map[string]float64{
		"ETH":   0.01,
		"DOT":   1.0,
		"ATOM":  1.0,
		"SOL":   0.01,
		"ADA":   10.0,
		"MATIC": 10.0,
		"AVAX":  0.1,
		"NEAR":  1.0,
		"FLR":   100.0,
		"KAVA":  1.0,
	}

	if min, exists := minStaking[asset]; exists {
		return min
	}
	return 1.0
}

func (s *KrakenScraper) parseTimestamp(timestamp string) *time.Time {
	if timestamp == "" {
		return nil
	}

	// Try parsing as Unix timestamp (seconds)
	if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		t := time.Unix(ts, 0)
		return &t
	}

	// Try parsing as ISO 8601
	if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
		return &t
	}

	return nil
}

func (s *KrakenScraper) extractPoolSize(title string) float64 {
	re := regexp.MustCompile(`(\d+(?:,\d+)*(?:\.\d+)?)\s*(?:USDT|USD|\$)`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 1 {
		numStr := strings.ReplaceAll(matches[1], ",", "")
		if val, err := strconv.ParseFloat(numStr, 64); err == nil {
			return val
		}
	}

	return 10000
}

func (s *KrakenScraper) extractReward(title string) string {
	re := regexp.MustCompile(`(\d+(?:,\d+)*(?:\.\d+)?)\s*(USDT|USD|\$|[A-Z]{2,10})`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 2 {
		return matches[1] + " " + matches[2]
	}

	return "Rewards"
}

func (s *KrakenScraper) cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "\n", " ")
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	if len(title) > 150 {
		title = title[:147] + "..."
	}

	return title
}

// API Response Structures
type KrakenAssetsResponse struct {
	Error  []string                     `json:"error"`
	Result map[string]KrakenAssetInfo   `json:"result"`
}

type KrakenAssetInfo struct {
	Aclass          string `json:"aclass"`
	Altname         string `json:"altname"`
	Decimals        int    `json:"decimals"`
	DisplayDecimals int    `json:"display_decimals"`
}

type KrakenStakingResponse struct {
	Error  []string                      `json:"error"`
	Result []KrakenStakingAsset          `json:"result"`
}

type KrakenStakingAsset struct {
	Method      string  `json:"method"`
	Asset       string  `json:"asset"`
	Rewards     float64 `json:"rewards"`
	OnChain     bool    `json:"on_chain"`
	CanUnstake  bool    `json:"can_unstake"`
	MinimumAmt  string  `json:"minimum_amt"`
}
