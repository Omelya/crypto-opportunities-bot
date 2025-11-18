package analytics

import (
	"log"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron            *cron.Cron
	analyticsService *Service
}

func NewScheduler(analyticsService *Service) *Scheduler {
	return &Scheduler{
		cron:            cron.New(),
		analyticsService: analyticsService,
	}
}

// Start starts the analytics scheduler
func (s *Scheduler) Start() error {
	// Run daily metrics calculation at 3:00 AM UTC
	_, err := s.cron.AddFunc("0 3 * * *", func() {
		log.Println("📊 Running daily analytics metrics calculation...")
		if err := s.analyticsService.CalculateDailyMetrics(); err != nil {
			log.Printf("❌ Failed to calculate daily metrics: %v", err)
		} else {
			log.Println("✅ Daily analytics metrics calculated")
		}
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	log.Println("✅ Analytics scheduler started (daily at 3:00 AM UTC)")
	return nil
}

// Stop stops the analytics scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("⏹ Analytics scheduler stopped")
}

// RunNow triggers metrics calculation immediately (for testing)
func (s *Scheduler) RunNow() error {
	log.Println("📊 Running analytics metrics calculation now...")
	if err := s.analyticsService.CalculateDailyMetrics(); err != nil {
		log.Printf("❌ Failed to calculate metrics: %v", err)
		return err
	}
	log.Println("✅ Analytics metrics calculated")
	return nil
}
