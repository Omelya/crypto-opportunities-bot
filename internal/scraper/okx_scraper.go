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

type OKXScraper struct {
	httpClient *http.Client
}

func NewOKXScraper() *OKXScraper {
	return &OKXScraper{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *OKXScraper) GetExchange() string {
	return models.ExchangeOKX
}

func (s *OKXScraper) ScrapeAll() ([]*models.Opportunity, error) {
	var allOpps []*models.Opportunity
	var errors []error

	launchpoolOpps, err := s.ScrapeLaunchpool()
	if err != nil {
		log.Printf("Error scraping OKX launchpool: %v", err)
		errors = append(errors, fmt.Errorf("launchpool: %w", err))
	} else {
		allOpps = append(allOpps, launchpoolOpps...)
		log.Printf("✅ Scraped %d OKX launchpool opportunities", len(launchpoolOpps))
	}

	airdropOpps, err := s.ScrapeAirdrops()
	if err != nil {
		log.Printf("Error scraping OKX airdrops: %v", err)
		errors = append(errors, fmt.Errorf("airdrops: %w", err))
	} else {
		allOpps = append(allOpps, airdropOpps...)
		log.Printf("✅ Scraped %d OKX airdrop opportunities", len(airdropOpps))
	}

	learnEarnOpps, err := s.ScrapeLearnEarn()
	if err != nil {
		log.Printf("Error scraping OKX learn&earn: %v", err)
		errors = append(errors, fmt.Errorf("learn&earn: %w", err))
	} else {
		allOpps = append(allOpps, learnEarnOpps...)
		log.Printf("✅ Scraped %d OKX learn&earn opportunities", len(learnEarnOpps))
	}

	if len(errors) == 3 {
		return allOpps, fmt.Errorf("all OKX scrapers failed: %v", errors)
	}

	return allOpps, nil
}

func (s *OKXScraper) ScrapeLaunchpool() ([]*models.Opportunity, error) {
	// OKX Jumpstart API
	url := "https://www.okx.com/priapi/v1/activity/jumpstart/list"

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

	var apiResp OKXJumpstartResponse
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

		poolSize, _ := strconv.ParseFloat(project.TotalReward, 64)
		minInvest, _ := strconv.ParseFloat(project.MinStake, 64)
		roi := s.calculateROI(project.APY)

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("okx", "launchpool", project.ProjectID),
			Exchange:      models.ExchangeOKX,
			Type:          models.OpportunityTypeLaunchpool,
			Title:         fmt.Sprintf("Jumpstart: %s", project.ProjectName),
			Description:   project.Description,
			Reward:        fmt.Sprintf("%s %s", project.TotalReward, project.RewardToken),
			EstimatedROI:  roi,
			PoolSize:      poolSize,
			MinInvestment: minInvest,
			Duration:      fmt.Sprintf("%d days", project.Duration),
			StartDate:     startDate,
			EndDate:       endDate,
			URL:           fmt.Sprintf("https://www.okx.com/earn/jumpstart/%s", project.ProjectID),
			IsActive:      project.Status == "ongoing",
			Metadata: map[string]interface{}{
				"stake_token":  project.StakeToken,
				"reward_token": project.RewardToken,
			},
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *OKXScraper) ScrapeAirdrops() ([]*models.Opportunity, error) {
	// OKX Announcements API
	url := "https://www.okx.com/priapi/v1/invest/activity/list?t=announcement&limit=20"

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

	var apiResp OKXAnnouncementResponse
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
			ExternalID:    GenerateExternalID("okx", "airdrop", announcement.ID),
			Exchange:      models.ExchangeOKX,
			Type:          models.OpportunityTypeAirdrop,
			Title:         s.cleanTitle(announcement.Title),
			Description:   "OKX Airdrop Campaign",
			Reward:        s.extractReward(announcement.Title),
			EstimatedROI:  roi,
			PoolSize:      poolSize,
			MinInvestment: 0,
			Duration:      "30 days",
			StartDate:     releaseDate,
			EndDate:       &endDate,
			URL:           fmt.Sprintf("https://www.okx.com/support/hc/en-us/articles/%s", announcement.ID),
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *OKXScraper) ScrapeLearnEarn() ([]*models.Opportunity, error) {
	// OKX Learn API
	url := "https://www.okx.com/priapi/v1/invest/activity/list?t=learn&limit=20"

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

	var apiResp OKXAnnouncementResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	for _, announcement := range apiResp.Data {
		releaseDate := s.parseTimestamp(announcement.PublishTime)
		endDate := releaseDate.AddDate(0, 0, 14)

		reward := s.extractLearnEarnReward(announcement.Title)
		roi := 0.5

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("okx", "learn_earn", announcement.ID),
			Exchange:      models.ExchangeOKX,
			Type:          models.OpportunityTypeLearnEarn,
			Title:         s.cleanTitle(announcement.Title),
			Description:   "Complete quizzes and earn rewards on OKX",
			Reward:        reward,
			EstimatedROI:  roi,
			MinInvestment: 0,
			Duration:      "5-10 minutes",
			StartDate:     releaseDate,
			EndDate:       &endDate,
			URL:           fmt.Sprintf("https://www.okx.com/learn/%s", announcement.ID),
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

// Helper methods
func (s *OKXScraper) parseTimestamp(timestamp string) *time.Time {
	if timestamp == "" {
		return nil
	}

	// Try parsing as Unix timestamp (milliseconds)
	if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		t := time.Unix(ts/1000, 0)
		return &t
	}

	// Try parsing as ISO 8601
	if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
		return &t
	}

	return nil
}

func (s *OKXScraper) calculateROI(apyStr string) float64 {
	// Remove % sign and parse
	apyStr = strings.TrimSuffix(apyStr, "%")
	apy, err := strconv.ParseFloat(apyStr, 64)
	if err != nil {
		return 5.0 // Default ROI
	}
	return apy
}

func (s *OKXScraper) isAirdropAnnouncement(title string) bool {
	keywords := []string{"airdrop", "giveaway", "distribution", "reward campaign", "promotion"}
	lowerTitle := strings.ToLower(title)

	for _, keyword := range keywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return false
}

func (s *OKXScraper) extractPoolSize(title string) float64 {
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

func (s *OKXScraper) extractReward(title string) string {
	re := regexp.MustCompile(`(\d+(?:,\d+)*(?:\.\d+)?)\s*(USDT|USD|\$|[A-Z]{2,10})`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 2 {
		return matches[1] + " " + matches[2]
	}

	return "Rewards Pool"
}

func (s *OKXScraper) extractLearnEarnReward(title string) string {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(USDT|USD|\$)`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 2 {
		return matches[1] + " " + matches[2]
	}

	return "$5 USDT"
}

func (s *OKXScraper) cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "\n", " ")
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	if len(title) > 150 {
		title = title[:147] + "..."
	}

	return title
}

// API Response Structures
type OKXJumpstartResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"msg"`
	Data    []OKXJumpstartProject `json:"data"`
}

type OKXJumpstartProject struct {
	ProjectID    string `json:"projectId"`
	ProjectName  string `json:"projectName"`
	Description  string `json:"description"`
	RewardToken  string `json:"rewardToken"`
	StakeToken   string `json:"stakeToken"`
	TotalReward  string `json:"totalReward"`
	APY          string `json:"apy"`
	Duration     int    `json:"duration"`
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
	Status       string `json:"status"`
	MinStake     string `json:"minStake"`
}

type OKXAnnouncementResponse struct {
	Code    string                `json:"code"`
	Message string                `json:"msg"`
	Data    []OKXAnnouncement     `json:"data"`
}

type OKXAnnouncement struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	PublishTime string `json:"publishTime"`
}
