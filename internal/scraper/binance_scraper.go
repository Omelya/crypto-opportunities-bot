package scraper

import (
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type BinanceScraper struct {
	httpClient *http.Client
}

func NewBinanceScraper() *BinanceScraper {
	return &BinanceScraper{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *BinanceScraper) GetExchange() string {
	return models.ExchangeBinance
}

func (s *BinanceScraper) ScrapeAll() ([]*models.Opportunity, error) {
	var allOpps []*models.Opportunity
	var errors []error

	launchpoolOpps, err := s.ScrapeLaunchpool()
	if err != nil {
		log.Printf("Error scraping launchpool: %v", err)
		errors = append(errors, fmt.Errorf("launchpool: %w", err))
	} else {
		allOpps = append(allOpps, launchpoolOpps...)
		log.Printf("✅ Scraped %d launchpool opportunities", len(launchpoolOpps))
	}

	airdropOpps, err := s.ScrapeAirdrops()
	if err != nil {
		log.Printf("Error scraping airdrops: %v", err)
		errors = append(errors, fmt.Errorf("airdrops: %w", err))
	} else {
		allOpps = append(allOpps, airdropOpps...)
		log.Printf("✅ Scraped %d airdrop opportunities", len(airdropOpps))
	}

	learnEarnOpps, err := s.ScrapeLearnEarn()
	if err != nil {
		log.Printf("Error scraping learn&earn: %v", err)
		errors = append(errors, fmt.Errorf("learn&earn: %w", err))
	} else {
		allOpps = append(allOpps, learnEarnOpps...)
		log.Printf("✅ Scraped %d learn&earn opportunities", len(learnEarnOpps))
	}

	if len(errors) == 3 {
		return allOpps, fmt.Errorf("all scrapers failed: %v", errors)
	}

	return allOpps, nil
}

func (s *BinanceScraper) ScrapeLaunchpool() ([]*models.Opportunity, error) {
	url := "https://www.binance.com/bapi/earn/v1/public/launchpool/project/list"

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

	var apiResp BinanceLaunchpoolResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var opportunities []*models.Opportunity

	for _, project := range apiResp.Data.Tracking.List {
		if project.Status != "RUNNING" {
			continue
		}

		endDate, _ := ParseDate(project.MineEndTime)
		startDate, _ := ParseDate(project.InvestStartTime)
		totalAmount := RemoveDecimal(project.RebateTotalAmount)
		roi, _ := strconv.ParseFloat(project.AnnualRate, 64)
		poolSize, _ := strconv.ParseFloat(project.RebateTotalAmount, 64)
		minInvest, _ := strconv.ParseFloat(project.MinInvestAmount, 64)

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("binance", "launchpool", project.ProjectID),
			Exchange:      models.ExchangeBinance,
			Type:          models.OpportunityTypeLaunchpool,
			Title:         fmt.Sprintf("Launchpool: %s", project.ProjectName),
			Description:   project.Description,
			Reward:        fmt.Sprintf("%s %s", totalAmount, project.RebateCoin),
			EstimatedROI:  roi,
			PoolSize:      poolSize,
			MinInvestment: minInvest,
			Duration:      fmt.Sprintf("%s days", project.Duration),
			StartDate:     startDate,
			EndDate:       endDate,
			URL:           fmt.Sprintf("https://www.binance.com/en/earn/launchpool/%s", project.ProjectID),
			IsActive:      true,
			Metadata: map[string]interface{}{
				"invest_asset": project.Asset,
				"reward_coin":  project.RebateCoin,
			},
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *BinanceScraper) ScrapeAirdrops() ([]*models.Opportunity, error) {
	url := "https://www.binance.com/bapi/composite/v1/public/cms/article/list/query"

	params := urlpkg.Values{}
	params.Add("type", "1")
	params.Add("catalogId", "128")
	params.Add("pageNo", "1")
	params.Add("pageSize", "20")

	fullURL := url + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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

	var apiResp BinanceAnnouncementResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(apiResp.Data.Catalogs) == 0 {
		return []*models.Opportunity{}, nil
	}

	var opportunities []*models.Opportunity

	for _, article := range apiResp.Data.Catalogs[0].Articles {
		if !s.isAirdropArticle(article.Title) {
			continue
		}

		releaseDate := time.Unix(article.ReleaseDate/1000, 0)
		endDate := releaseDate.AddDate(0, 0, 30)

		poolSize := s.extractPoolSize(article.Title)
		roi := 2.0

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("binance", "airdrop", article.Code),
			Exchange:      models.ExchangeBinance,
			Type:          models.OpportunityTypeAirdrop,
			Title:         s.cleanTitle(article.Title),
			Description:   "Binance Airdrop Campaign",
			Reward:        s.extractReward(article.Title),
			EstimatedROI:  roi,
			PoolSize:      poolSize,
			MinInvestment: 0,
			Duration:      "30 days",
			StartDate:     &releaseDate,
			EndDate:       &endDate,
			URL:           fmt.Sprintf("https://www.binance.com/en/support/announcement/%s", article.Code),
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *BinanceScraper) ScrapeLearnEarn() ([]*models.Opportunity, error) {
	url := "https://www.binance.com/bapi/composite/v1/public/cms/article/list/query"

	payload := map[string]interface{}{
		"type":      1,
		"catalogId": 220,
		"pageNo":    1,
		"pageSize":  20,
	}

	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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

	var apiResp BinanceAnnouncementResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(apiResp.Data.Catalogs) == 0 {
		return []*models.Opportunity{}, nil
	}

	var opportunities []*models.Opportunity

	for _, article := range apiResp.Data.Catalogs[0].Articles {
		if !s.isLearnEarnArticle(article.Title) {
			continue
		}

		releaseDate := time.Unix(article.ReleaseDate/1000, 0)
		endDate := releaseDate.AddDate(0, 0, 14)

		reward := s.extractLearnEarnReward(article.Title)
		roi := 0.5

		opp := &models.Opportunity{
			ExternalID:    GenerateExternalID("binance", "learn_earn", article.Code),
			Exchange:      models.ExchangeBinance,
			Type:          models.OpportunityTypeLearnEarn,
			Title:         s.cleanTitle(article.Title),
			Description:   "Complete quizzes and earn rewards",
			Reward:        reward,
			EstimatedROI:  roi,
			MinInvestment: 0,
			Duration:      "5-10 minutes",
			StartDate:     &releaseDate,
			EndDate:       &endDate,
			URL:           fmt.Sprintf("https://www.binance.com/en/support/announcement/%s", article.Code),
			IsActive:      true,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

func (s *BinanceScraper) isAirdropArticle(title string) bool {
	keywords := []string{"airdrop", "token distribution", "promotion", "giveaway", "campaign"}
	lowerTitle := strings.ToLower(title)

	for _, keyword := range keywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return false
}

func (s *BinanceScraper) isLearnEarnArticle(title string) bool {
	keywords := []string{"learn and earn", "learn & earn", "quiz"}
	lowerTitle := strings.ToLower(title)

	for _, keyword := range keywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return false
}

func (s *BinanceScraper) extractPoolSize(title string) float64 {
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

func (s *BinanceScraper) extractReward(title string) string {
	re := regexp.MustCompile(`(\d+(?:,\d+)*(?:\.\d+)?)\s*(USDT|USD|\$|[A-Z]{2,10})`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 2 {
		return matches[1] + " " + matches[2]
	}

	return "Rewards Pool"
}

func (s *BinanceScraper) extractLearnEarnReward(title string) string {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(USDT|USD|\$)`)
	matches := re.FindStringSubmatch(title)

	if len(matches) > 2 {
		return matches[1] + " " + matches[2]
	}

	return "$5 USDT"
}

func (s *BinanceScraper) cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "\n", " ")
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	if len(title) > 150 {
		title = title[:147] + "..."
	}

	return title
}

type BinanceLaunchpoolResponse struct {
	Code    string                `json:"code"`
	Message string                `json:"message"`
	Data    BinanceLaunchpoolData `json:"data"`
}

type BinanceLaunchpoolData struct {
	Tracking BinanceLaunchpoolTracking `json:"tracking"`
}

type BinanceLaunchpoolTracking struct {
	Total string                     `json:"total"`
	List  []BinanceLaunchpoolProject `json:"list"`
}

type BinanceLaunchpoolProject struct {
	ProjectID         string `json:"projectId"`
	ProjectName       string `json:"projectName"`
	Description       string `json:"description"`
	RebateCoin        string `json:"rebateCoin"`
	RebateTotalAmount string `json:"rebateTotalAmount"`
	AnnualRate        string `json:"annualRate"`
	Duration          string `json:"duration"`
	InvestStartTime   string `json:"investStartTime"`
	MineEndTime       string `json:"mineEndTime"`
	Status            string `json:"status"`
	MinInvestAmount   string `json:"minInvestAmount"`
	Asset             string `json:"asset"`
}

type BinanceAnnouncementResponse struct {
	Code    string                  `json:"code"`
	Message string                  `json:"message"`
	Data    BinanceAnnouncementData `json:"data"`
}

type BinanceAnnouncementData struct {
	Catalogs []BinanceCatalog `json:"catalogs"`
	Total    int              `json:"total"`
}

type BinanceCatalog struct {
	CatalogId int                       `json:"catalogId"`
	Articles  []BinanceAnnouncementItem `json:"articles"`
}

type BinanceAnnouncementItem struct {
	Id          int64  `json:"id"`
	Code        string `json:"code"`
	Title       string `json:"title"`
	Type        int    `json:"type"`
	ReleaseDate int64  `json:"releaseDate"`
}
