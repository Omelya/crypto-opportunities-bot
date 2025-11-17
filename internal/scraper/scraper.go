package scraper

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"crypto/md5"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Scraper interface {
	GetExchange() string
	ScrapeAll() ([]*models.Opportunity, error)
	ScrapeLaunchpool() ([]*models.Opportunity, error)
	ScrapeAirdrops() ([]*models.Opportunity, error)
	ScrapeLearnEarn() ([]*models.Opportunity, error)
}

type OpportunityCallback func(*models.Opportunity)

type Service struct {
	scrapers                []Scraper
	oppRepo                 repository.OpportunityRepository
	newOpportunityCallbacks []OpportunityCallback
}

func NewScraperService(oppRepo repository.OpportunityRepository) *Service {
	return &Service{
		scrapers:                []Scraper{},
		oppRepo:                 oppRepo,
		newOpportunityCallbacks: []OpportunityCallback{},
	}
}

func (s *Service) RegisterScraper(scraper Scraper) {
	s.scrapers = append(s.scrapers, scraper)
	log.Printf("Registered scraper: %s", scraper.GetExchange())
}

func (s *Service) OnNewOpportunity(callback OpportunityCallback) {
	s.newOpportunityCallbacks = append(s.newOpportunityCallbacks, callback)
}

func (s *Service) RunAll() error {
	totalNew := 0
	totalUpdated := 0

	for _, scraper := range s.scrapers {
		log.Printf("Scraping %s...", scraper.GetExchange())

		opportunities, err := scraper.ScrapeAll()
		if err != nil {
			log.Printf("Error scraping %s: %v", scraper.GetExchange(), err)
			continue
		}

		log.Printf("Found %d opportunities on %s", len(opportunities), scraper.GetExchange())

		for _, opp := range opportunities {
			existing, _ := s.oppRepo.GetByExternalID(opp.ExternalID)

			if existing == nil {
				if err := s.oppRepo.Create(opp); err != nil {
					log.Printf("Error creating opportunity: %v", err)
					continue
				}
				totalNew++
				log.Printf("✅ New opportunity: %s - %s", opp.Exchange, opp.Title)

				s.notifyNewOpportunity(opp)
			} else {
				existing.Title = opp.Title
				existing.Description = opp.Description
				existing.EstimatedROI = opp.EstimatedROI
				existing.EndDate = opp.EndDate
				existing.IsActive = opp.IsActive

				if err := s.oppRepo.Update(existing); err != nil {
					log.Printf("Error updating opportunity: %v", err)
					continue
				}
				totalUpdated++
			}
		}
	}

	log.Printf("Scraping completed: %d new, %d updated", totalNew, totalUpdated)

	if err := s.oppRepo.DeactivateExpired(); err != nil {
		log.Printf("Error deactivating expired: %v", err)
	}

	return nil
}

// RunScraper запускає конкретний scraper по імені
func (s *Service) RunScraper(name string) error {
	for _, scraper := range s.scrapers {
		if scraper.GetExchange() == name {
			log.Printf("Scraping %s...", scraper.GetExchange())

			opportunities, err := scraper.ScrapeAll()
			if err != nil {
				return fmt.Errorf("error scraping %s: %w", scraper.GetExchange(), err)
			}

			log.Printf("Found %d opportunities on %s", len(opportunities), scraper.GetExchange())

			totalNew := 0
			for _, opp := range opportunities {
				existing, _ := s.oppRepo.GetByExternalID(opp.ExternalID)

				if existing == nil {
					if err := s.oppRepo.Create(opp); err != nil {
						log.Printf("Error creating opportunity: %v", err)
						continue
					}
					totalNew++
					log.Printf("✅ New opportunity: %s - %s", opp.Exchange, opp.Title)
					s.notifyNewOpportunity(opp)
				}
			}

			log.Printf("Scraping %s completed: %d new opportunities", name, totalNew)
			return nil
		}
	}

	return fmt.Errorf("scraper not found: %s", name)
}

func (s *Service) notifyNewOpportunity(opp *models.Opportunity) {
	for _, callback := range s.newOpportunityCallbacks {
		go callback(opp)
	}
}

func GenerateExternalID(exchange, oppType, title string) string {
	data := fmt.Sprintf("%s:%s:%s", exchange, oppType, title)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func ParseDate(dateStr string) (*time.Time, error) {
	if matched, _ := regexp.MatchString(`^\d+$`, dateStr); matched {
		milliseconds, err := strconv.ParseInt(dateStr, 10, 64)
		if err == nil {
			t := time.Unix(milliseconds/1000, 0)
			return &t, nil
		}
	}

	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"02/01/2006",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unable to parse date: %s", dateStr)
}

func RemoveDecimal(numStr string) string {
	parts := strings.Split(numStr, ".")
	return parts[0]
}
