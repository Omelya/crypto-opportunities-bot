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

type GateIOScraper struct {
	httpClient *http.Client
}

func NewGateIOScraper() *GateIOScraper {
	return &GateIOScraper{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *GateIOScraper) GetExchange() string {
	return models.ExchangeGateIO
}

func (s *GateIOScraper) ScrapeAll() ([]*models.Opportunity, error) {
	var allOpps []*models.Opportunity
	var errors []error

	launchpoolOpps, err := s.ScrapeLaunchpool()
	if err != nil {
		log.Printf("Error scraping Gate.io launchpool: %v", err)
		errors = append(errors, fmt.Errorf("launchpool: %w", err))
	} else {
		allOpps = append(allOpps, launchpoolOpps...)
		log.Printf("✅ Scraped %d Gate.io launchpool opportunities", len(launchpoolOpps))
	}

	airdropOpps, err := s.ScrapeAirdrops()
	if err != nil {
		log.Printf("Error scraping Gate.io airdrops: %v", err)
		errors = append(errors, fmt.Errorf("airdrops: %w", err))
	} else {
		allOpps = append(allOpps, airdropOpps...)
		log.Printf("✅ Scraped %d Gate.io airdrop opportunities", len(airdropOpps))
	}

	learnEarnOpps, err := s.ScrapeLearnEarn()
	if err != nil {
		log.Printf("Error scraping Gate.io learn&earn: %v", err)
		errors = append(errors, fmt.Errorf("learn&earn: %w", err))
	} else {
		allOpps = append(allOpps, learnEarnOpps...)
		log.Printf("✅ Scraped %d Gate.io learn&earn opportunities", len(learnEarnOpps))
	}

	if len(errors) == 3 {
		return allOpps, fmt.Errorf("all Gate.io scrapers failed: %v", errors)
	}

	return allOpps, nil
}

func (s *GateIOScraper) ScrapeLaunchpool() ([]*models.Opportunity, error) {
	// Gate.io Startup API
	url := "https://www.gate.io/apiw/v1/startup/list"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
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

	var apiResp GateIOStartupResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	for _, project := range apiResp.Data {
		if project.Status != "ongoing" && project.Status != "upcoming" {
			continue
		}

		startDate := s.parseTimestamp(project.StartTime)
		endDate := s.parseTimestamp(project.EndTime)

		poolSize, _ := strconv.ParseFloat(project.TotalAmount, 64)
		minInvest, _ := strconv.ParseFloat(project.MinLock, 64)
		roi, _ := strconv.ParseFloat(project.EstimatedAPY, 64)

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("gateio", "launchpool", project.ProjectID),
			Exchange:      models.ExchangeGateIO,
			Type:          models.OpportunityTypeLaunchpool,
			Title:         fmt.Sprintf("Startup: %s", project.ProjectName),
			Description:   project.Introduction,
			Reward:        fmt.Sprintf("%s %s", project.TotalAmount, project.Token),
			EstimatedROI:  roi,
			PoolSize:      poolSize,
			MinInvestment: minInvest,
			Duration:      fmt.Sprintf("%d days", project.DurationDays),
			StartDate:     startDate,
			EndDate:       endDate,
			URL:           fmt.Sprintf("https://www.gate.io/startup/%s", project.ProjectID),
			IsActive:      project.Status == "ongoing",
			Metadata: map[string]interface{}{
				"lock_token":   project.LockToken,
				"reward_token": project.Token,
			},
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *GateIOScraper) ScrapeAirdrops() ([]*models.Opportunity, error) {
	// Gate.io Announcements API
	url := "https://www.gate.io/apiw/v1/announcements?category=activities&limit=20"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
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

	var apiResp GateIOAnnouncementResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	for _, announcement := range apiResp.Data {
		if !s.isAirdropAnnouncement(announcement.Title) {
			continue
		}

		releaseDate := s.parseTimestamp(announcement.PublishTime)
		endDate := releaseDate.AddDate(0, 0, 30)

		poolSize := s.extractPoolSize(announcement.Title)
		roi := 2.0

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("gateio", "airdrop", announcement.ID),
			Exchange:      models.ExchangeGateIO,
			Type:          models.OpportunityTypeAirdrop,
			Title:         s.cleanTitle(announcement.Title),
			Description:   "Gate.io Airdrop Campaign",
			Reward:        s.extractReward(announcement.Title),
			EstimatedROI:  roi,
			PoolSize:      poolSize,
			MinInvestment: 0,
			Duration:      "30 days",
			StartDate:     releaseDate,
			EndDate:       &endDate,
			URL:           fmt.Sprintf("https://www.gate.io/article/%s", announcement.ID),
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *GateIOScraper) ScrapeLearnEarn() ([]*models.Opportunity, error) {
	// Gate.io Learn & Earn campaigns (через announcements з фільтром)
	url := "https://www.gate.io/apiw/v1/announcements?category=learn&limit=20"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
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

	var apiResp GateIOAnnouncementResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	for _, announcement := range apiResp.Data {
		if !s.isLearnEarnAnnouncement(announcement.Title) {
			continue
		}

		releaseDate := s.parseTimestamp(announcement.PublishTime)
		endDate := releaseDate.AddDate(0, 0, 14)

		reward := s.extractLearnEarnReward(announcement.Title)
		roi := 0.5

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("gateio", "learn_earn", announcement.ID),
			Exchange:      models.ExchangeGateIO,
			Type:          models.OpportunityTypeLearnEarn,
			Title:         s.cleanTitle(announcement.Title),
			Description:   "Complete quizzes and earn rewards on Gate.io",
			Reward:        reward,
			EstimatedROI:  roi,
			MinInvestment: 0,
			Duration:      "5-10 minutes",
			StartDate:     releaseDate,
			EndDate:       &endDate,
			URL:           fmt.Sprintf("https://www.gate.io/learn/%s", announcement.ID),
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

// Helper methods
func (s *GateIOScraper) parseTimestamp(timestamp string) *time.Time {
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

func (s *GateIOScraper) isAirdropAnnouncement(title string) bool {
	keywords := []string{"airdrop", "giveaway", "distribution", "reward", "campaign", "promotion"}
	lowerTitle := strings.ToLower(title)

	for _, keyword := range keywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return false
}

func (s *GateIOScraper) isLearnEarnAnnouncement(title string) bool {
	keywords := []string{"learn", "quiz", "earn", "education"}
	lowerTitle := strings.ToLower(title)

	count := 0
	for _, keyword := range keywords {
		if strings.Contains(lowerTitle, keyword) {
			count++
		}
	}
	return count >= 2 // At least 2 keywords matched
}

func (s *GateIOScraper) extractPoolSize(title string) float64 {
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

func (s *GateIOScraper) extractReward(title string) string {
	re := regexp.MustCompile(`(\d+(?:,\d+)*(?:\.\d+)?)\s*(USDT|USD|\$|[A-Z]{2,10})`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 2 {
		return matches[1] + " " + matches[2]
	}

	return "Rewards Pool"
}

func (s *GateIOScraper) extractLearnEarnReward(title string) string {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(USDT|USD|\$)`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 2 {
		return matches[1] + " " + matches[2]
	}

	return "$5 USDT"
}

func (s *GateIOScraper) cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "\n", " ")
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	if len(title) > 150 {
		title = title[:147] + "..."
	}

	return title
}

// API Response Structures
type GateIOStartupResponse struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    []GateIOStartupProject `json:"data"`
}

type GateIOStartupProject struct {
	ProjectID    string `json:"project_id"`
	ProjectName  string `json:"project_name"`
	Introduction string `json:"introduction"`
	Token        string `json:"token"`
	LockToken    string `json:"lock_token"`
	TotalAmount  string `json:"total_amount"`
	EstimatedAPY string `json:"estimated_apy"`
	DurationDays int    `json:"duration_days"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Status       string `json:"status"`
	MinLock      string `json:"min_lock"`
}

type GateIOAnnouncementResponse struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    []GateIOAnnouncement  `json:"data"`
}

type GateIOAnnouncement struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	PublishTime string `json:"publish_time"`
}
