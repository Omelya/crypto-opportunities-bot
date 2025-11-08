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
	_, err := s.cron.AddFunc("*/5 * * * *", func() {
		log.Println("Starting scheduled scraping...")
		if err := s.service.RunAll(); err != nil {
			log.Printf("Scheduled scraping error: %v", err)
		}
	})

	if err != nil {
		return err
	}

	_, err = s.cron.AddFunc("0 2 * * *", func() {
		log.Println("Cleaning up old opportunities...")

		if err := s.service.oppRepo.DeleteOld(30); err != nil {
			log.Printf("Cleanup error: %v", err)
		}
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	log.Println("✅ Scraper scheduler started")

	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("Scraper scheduler stopped")
}

func (s *Scheduler) RunNow() error {
	return s.service.RunAll()
}
