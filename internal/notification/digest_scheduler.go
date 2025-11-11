package notification

import (
	"crypto-opportunities-bot/internal/repository"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type DigestScheduler struct {
	cron     *cron.Cron
	service  *Service
	userRepo repository.UserRepository
	timezone string
}

func NewDigestScheduler(service *Service) *DigestScheduler {
	return &DigestScheduler{
		cron:     cron.New(),
		service:  service,
		userRepo: service.userRepo,
		timezone: "UTC",
	}
}

func (s *DigestScheduler) Start() error {
	_, err := s.cron.AddFunc("0 * * * *", func() {
		log.Println("⏰ Running hourly digest check...")
		if err := s.sendDigestsByTimezone(); err != nil {
			log.Printf("❌ Digest error: %v", err)
		}
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	log.Println("✅ Daily digest scheduler started (checks every hour)")

	return nil
}

func (s *DigestScheduler) Stop() {
	s.cron.Stop()
	log.Println("Daily digest scheduler stopped")
}

func (s *DigestScheduler) RunNow() error {
	log.Println("Running daily digest manually...")
	return s.sendDigestsByTimezone()
}

func (s *DigestScheduler) sendDigestsByTimezone() error {
	users, err := s.userRepo.List(0, 10000)
	if err != nil {
		return err
	}

	sent := 0
	skipped := 0

	for _, user := range users {
		if !user.IsActive || user.IsBlocked {
			skipped++
			continue
		}

		prefs, err := s.service.prefsRepo.GetByUserID(user.ID)
		if err != nil || prefs == nil {
			skipped++
			continue
		}

		if !prefs.DailyDigestEnabled {
			skipped++
			continue
		}

		if !s.isDigestTimeForUser(user.Timezone, prefs.DailyDigestTime) {
			skipped++
			continue
		}

		if err := s.service.SendDailyDigest(user.ID); err != nil {
			log.Printf("Failed to send digest to user %d: %v", user.ID, err)
		} else {
			sent++
		}

		time.Sleep(100 * time.Millisecond)
	}

	if sent > 0 {
		log.Printf("Daily digest: sent %d, skipped %d", sent, skipped)
	}

	return nil
}

func (s *DigestScheduler) isDigestTimeForUser(userTimezone, digestTime string) bool {
	loc, err := time.LoadLocation(userTimezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	currentHour := now.Format("15:04")

	targetTime := digestTime
	if targetTime == "" {
		targetTime = "09:00"
	}

	targetHour := targetTime[:2]
	currentHourInt := currentHour[:2]

	return currentHourInt == targetHour
}
