package scraper

import (
	"log"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	service *Service
}

func NewScheduler(service *Service) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		service: service,
	}
}

func (s *Scheduler) Start() error {
	// Scraping every 5 minutes
	_, err := s.cron.AddFunc("*/5 * * * *", func() {
		log.Println("Starting scheduled scraping...")
		if err := s.service.RunAll(); err != nil {
			log.Printf("Scheduled scraping error: %v", err)
		}
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	log.Println("✅ Scraper scheduler started (every 5 minutes)")

	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("Scraper scheduler stopped")
}

func (s *Scheduler) RunNow() error {
	return s.service.RunAll()
}

// RunScraper запускає конкретний scraper по імені
func (s *Scheduler) RunScraper(name string) error {
	return s.service.RunScraper(name)
}
