package notification

import (
	"crypto-opportunities-bot/internal/repository"
	"log"

	"github.com/robfig/cron/v3"
)

type DigestScheduler struct {
	cron    *cron.Cron
	service *Service
}

func NewDigestScheduler(service *Service) *DigestScheduler {
	return &DigestScheduler{
		cron:    cron.New(),
		service: service,
	}
}

func (s *DigestScheduler) Start() error {
	// Відправка щоденного дайджесту о 09:00 UTC
	_, err := s.cron.AddFunc("0 9 * * *", func() {
		log.Println("⏰ Starting daily digest job...")
		if err := s.service.SendDailyDigestToAll(); err != nil {
			log.Printf("❌ Daily digest error: %v", err)
		}
		log.Println("✅ Daily digest job completed")
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	log.Println("✅ Daily digest scheduler started (runs at 09:00 UTC)")

	return nil
}

func (s *DigestScheduler) Stop() {
	s.cron.Stop()
	log.Println("Daily digest scheduler stopped")
}

func (s *DigestScheduler) RunNow() error {
	log.Println("Running daily digest manually...")
	return s.service.SendDailyDigestToAll()
}

// UserDigestJob - для індивідуальної відправки з врахуванням timezone
type UserDigestJob struct {
	service  *Service
	userRepo repository.UserRepository
}

func NewUserDigestJob(service *Service, userRepo repository.UserRepository) *UserDigestJob {
	return &UserDigestJob{
		service:  service,
		userRepo: userRepo,
	}
}

// Run відправляє дайджест користувачам згідно з їх timezone
func (j *UserDigestJob) Run() error {
	// TODO: Імплементувати логіку з врахуванням timezone користувачів
	// Поки що просто відправляємо всім о 09:00 UTC

	users, err := j.userRepo.List(0, 10000)
	if err != nil {
		return err
	}

	sent := 0
	failed := 0

	for _, user := range users {
		if !user.IsActive || user.IsBlocked {
			continue
		}

		if err := j.service.SendDailyDigest(user.ID); err != nil {
			log.Printf("Failed to send digest to user %d: %v", user.ID, err)
			failed++
		} else {
			sent++
		}
	}

	log.Printf("Daily digest job: sent %d, failed %d", sent, failed)
	return nil
}
