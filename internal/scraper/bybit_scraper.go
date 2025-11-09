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

type BybitScraper struct {
	httpClient *http.Client
}

func NewBybitScraper() *BybitScraper {
	return &BybitScraper{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *BybitScraper) GetExchange() string {
	return models.ExchangeBybit
}

func (s *BybitScraper) ScrapeAll() ([]*models.Opportunity, error) {
	var allOpps []*models.Opportunity

	launchpoolOpps, err := s.ScrapeLaunchpool()
	if err != nil {
		log.Printf("Error scraping Bybit launchpool: %v", err)
	} else {
		allOpps = append(allOpps, launchpoolOpps...)
	}

	airdropOpps, err := s.ScrapeAirdrops()
	if err != nil {
		log.Printf("Error scraping Bybit airdrops: %v", err)
	} else {
		allOpps = append(allOpps, airdropOpps...)
	}

	learnEarnOpps, err := s.ScrapeLearnEarn()
	if err != nil {
		log.Printf("Error scraping Bybit learn&earn: %v", err)
	} else {
		allOpps = append(allOpps, learnEarnOpps...)
	}

	return allOpps, nil
}

func (s *BybitScraper) ScrapeLaunchpool() ([]*models.Opportunity, error) {
	url := "https://api.bybit.com/v5/announcements/index?locale=en-US&type=latest_activities&page=1&limit=20"

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch launchpool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp BybitAnnouncementResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	for _, item := range apiResp.Result.List {
		if !s.isLaunchpoolAnnouncement(item.Title) {
			continue
		}

		publishTime := time.Unix(item.DateTimestamp/1000, 0)
		endDate := publishTime.AddDate(0, 0, 14)

		poolSize := s.extractPoolSize(item.Title)
		roi := s.calculateROI(poolSize)
		duration := endDate.Sub(publishTime)
		days := int(duration / (24 * time.Hour))

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("bybit", "launchpool", strconv.FormatInt(item.ID, 10)),
			Exchange:      models.ExchangeBybit,
			Type:          models.OpportunityTypeLaunchpool,
			Title:         s.cleanTitle(item.Title),
			Description:   item.Description,
			Reward:        s.extractReward(item.Title),
			EstimatedROI:  roi,
			PoolSize:      poolSize,
			MinInvestment: 10,
			Duration:      fmt.Sprintf("%d days", days),
			StartDate:     &publishTime,
			EndDate:       &endDate,
			URL:           item.Url,
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *BybitScraper) ScrapeAirdrops() ([]*models.Opportunity, error) {
	url := "https://api.bybit.com/v5/announcements/index?locale=en-US&type=latest_activities&page=1&limit=20"

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch airdrops: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp BybitAnnouncementResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	for _, item := range apiResp.Result.List {
		if !s.isAirdropAnnouncement(item.Title) {
			continue
		}

		publishTime := time.Unix(item.DateTimestamp/1000, 0)
		endDate := publishTime.AddDate(0, 0, 30)

		poolSize := s.extractPoolSize(item.Title)
		duration := endDate.Sub(publishTime)
		days := int(duration / (24 * time.Hour))

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("bybit", "airdrop", strconv.FormatInt(item.ID, 10)),
			Exchange:      models.ExchangeBybit,
			Type:          models.OpportunityTypeAirdrop,
			Title:         s.cleanTitle(item.Title),
			Description:   item.Description,
			Reward:        s.extractReward(item.Title),
			EstimatedROI:  2.0,
			PoolSize:      poolSize,
			MinInvestment: 0,
			Duration:      fmt.Sprintf("%d days", days),
			StartDate:     &publishTime,
			EndDate:       &endDate,
			URL:           item.Url,
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *BybitScraper) ScrapeLearnEarn() ([]*models.Opportunity, error) {
	// Bybit Learn & Earn campaigns are less common
	// Scraping from announcements
	url := "https://api.bybit.com/v5/announcements/index?locale=en-US&type=latest_activities&page=1&limit=20"

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch learn&earn: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp BybitAnnouncementResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	for _, item := range apiResp.Result.List {
		if !s.isLearnEarnAnnouncement(item.Title) {
			continue
		}

		publishTime := time.Unix(item.DateTimestamp/1000, 0)
		endDate := publishTime.AddDate(0, 0, 14)

		reward := s.extractReward(item.Title)
		duration := endDate.Sub(publishTime)
		days := int(duration / (24 * time.Hour))

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("bybit", "learn_earn", strconv.FormatInt(item.ID, 10)),
			Exchange:      models.ExchangeBybit,
			Type:          models.OpportunityTypeLearnEarn,
			Title:         s.cleanTitle(item.Title),
			Description:   "Complete tasks and earn rewards",
			Reward:        reward,
			EstimatedROI:  0.5,
			MinInvestment: 0,
			Duration:      fmt.Sprintf("%d days", days),
			StartDate:     &publishTime,
			EndDate:       &endDate,
			URL:           item.Url,
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *BybitScraper) isLaunchpoolAnnouncement(title string) bool {
	keywords := []string{"launchpool", "launch pool", "staking rewards", "mining"}
	lowerTitle := strings.ToLower(title)

	for _, keyword := range keywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return false
}

func (s *BybitScraper) isAirdropAnnouncement(title string) bool {
	keywords := []string{"airdrop", "token distribution", "giveaway", "promotion"}
	lowerTitle := strings.ToLower(title)

	for _, keyword := range keywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return false
}

func (s *BybitScraper) isLearnEarnAnnouncement(title string) bool {
	keywords := []string{"learn", "quiz", "trading competition", "campaign"}
	lowerTitle := strings.ToLower(title)

	for _, keyword := range keywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return false
}

func (s *BybitScraper) extractPoolSize(title string) float64 {
	re := regexp.MustCompile(`(\d+(?:,\d+)*(?:\.\d+)?)\s*(?:USDT|USD|\$)`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 1 {
		numStr := strings.ReplaceAll(matches[1], ",", "")
		if val, err := strconv.ParseFloat(numStr, 64); err == nil {
			return val
		}
	}

	return 5000
}

func (s *BybitScraper) extractReward(title string) string {
	re := regexp.MustCompile(`(\d+(?:,\d+)*(?:\.\d+)?)\s*(USDT|USD|\$|[A-Z]{2,10})`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 2 {
		return matches[1] + " " + matches[2]
	}

	return "Reward Pool"
}

func (s *BybitScraper) calculateROI(poolSize float64) float64 {
	if poolSize >= 100000 {
		return 5.0
	} else if poolSize >= 50000 {
		return 3.0
	} else if poolSize >= 10000 {
		return 2.0
	}
	return 1.0
}

func (s *BybitScraper) cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "\n", " ")
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	if len(title) > 150 {
		title = title[:147] + "..."
	}

	return title
}

// API Response structures

type BybitAnnouncementResponse struct {
	RetCode int                   `json:"retCode"`
	RetMsg  string                `json:"retMsg"`
	Result  BybitAnnouncementData `json:"result"`
}

type BybitAnnouncementData struct {
	List []BybitAnnouncementItem `json:"list"`
}

type BybitAnnouncementItem struct {
	ID            int64                 `json:"id"`
	Title         string                `json:"title"`
	Description   string                `json:"description"`
	Type          BybitAnnouncementType `json:"type"`
	Url           string                `json:"url"`
	DateTimestamp int64                 `json:"dateTimestamp"`
	StartTime     int64                 `json:"startTime"`
	EndTime       int64                 `json:"endTime"`
}

type BybitAnnouncementType struct {
	Title string `json:"title"`
	Key   string `json:"key"`
}
