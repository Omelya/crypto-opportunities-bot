package scraper

import (
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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

	// Launchpool
	launchpoolOpps, err := s.ScrapeLaunchpool()
	if err != nil {
		return nil, err
	}
	allOpps = append(allOpps, launchpoolOpps...)

	// Airdrops
	airdropOpps, err := s.ScrapeAirdrops()
	if err != nil {
		return nil, err
	}
	allOpps = append(allOpps, airdropOpps...)

	// Learn & Earn
	learnEarnOpps, err := s.ScrapeLearnEarn()
	if err != nil {
		return nil, err
	}
	allOpps = append(allOpps, learnEarnOpps...)

	return allOpps, nil
}

func (s *BinanceScraper) ScrapeLaunchpool() ([]*models.Opportunity, error) {
	// Binance має API для launchpool
	// Документація: https://www.binance.com/bapi/composite/v1/public/cms/article/list/query

	url := "https://www.binance.com/bapi/earn/v1/public/launchpool/project/list"

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch launchpool: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("failed to close body: %w", err)
		}
	}(resp.Body)

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
	// Binance airdrops частіше оголошуються через announcements
	// Тут можна парсити сторінку або використати API

	// TODO: Імплементація парсингу airdrops
	// Поки що повертаємо порожній список

	return []*models.Opportunity{}, nil
}

func (s *BinanceScraper) ScrapeLearnEarn() ([]*models.Opportunity, error) {
	// Binance Learn & Earn campaigns
	// TODO: Імплементація

	return []*models.Opportunity{}, nil
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
	Status            string `json:"status"` // RUNNING, ENDED
	MinInvestAmount   string `json:"minInvestAmount"`
	Asset             string `json:"asset"`
}
