package payment

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/payment/monobank"
	"crypto-opportunities-bot/internal/repository"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	monoClient *monobank.Client
	subsRepo   repository.SubscriptionRepository
	paymentRepo repository.PaymentRepository
	userRepo   repository.UserRepository

	webhookURL  string
	redirectURL string
}

type Config struct {
	MonobankToken string
	WebhookURL    string
	RedirectURL   string
}

func NewService(
	cfg *Config,
	subsRepo repository.SubscriptionRepository,
	paymentRepo repository.PaymentRepository,
	userRepo repository.UserRepository,
) *Service {
	return &Service{
		monoClient:  monobank.NewClient(cfg.MonobankToken),
		subsRepo:    subsRepo,
		paymentRepo: paymentRepo,
		userRepo:    userRepo,
		webhookURL:  cfg.WebhookURL,
		redirectURL: cfg.RedirectURL,
	}
}

// CreateSubscription створює нову підписку та invoice для оплати
func (s *Service) CreateSubscription(userID uint, plan string, trialDays int) (*models.Subscription, string, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, "", fmt.Errorf("user not found")
	}

	// Перевірити чи вже є активна підписка
	existingSub, err := s.subsRepo.GetActiveByUserID(userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check existing subscription: %w", err)
	}

	if existingSub != nil {
		return nil, "", fmt.Errorf("user already has active subscription")
	}

	// Отримати ціну плану
	amount, ok := monobank.PlanPrices[plan]
	if !ok {
		return nil, "", fmt.Errorf("invalid plan: %s", plan)
	}

	duration, ok := monobank.PlanDurations[plan]
	if !ok {
		return nil, "", fmt.Errorf("invalid plan duration: %s", plan)
	}

	now := time.Now()
	periodStart := now
	periodEnd := now.Add(duration)

	// Trial період
	var trialEnd *time.Time
	if trialDays > 0 {
		trial := now.Add(time.Duration(trialDays) * 24 * time.Hour)
		trialEnd = &trial
		periodEnd = trial
	}

	// Генерувати унікальний reference
	reference := fmt.Sprintf("sub_%s_%d", uuid.New().String()[:8], userID)

	// Створити підписку в БД
	subscription := &models.Subscription{
		UserID:             userID,
		Provider:           models.PaymentProviderMonobank,
		Plan:               plan,
		Status:             models.SubscriptionStatusPending,
		Amount:             amount,
		Currency:           "UAH",
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		TrialEnd:           trialEnd,
		AutoRenew:          true,
		MonobankReference:  reference,
	}

	if err := s.subsRepo.Create(subscription); err != nil {
		return nil, "", fmt.Errorf("failed to create subscription: %w", err)
	}

	// Якщо trial - не створювати invoice
	if trialDays > 0 {
		log.Printf("✅ Trial subscription created for user %d: %d days", userID, trialDays)
		subscription.Status = models.SubscriptionStatusActive
		if err := s.subsRepo.Update(subscription); err != nil {
			log.Printf("Failed to update trial subscription: %v", err)
		}

		// Оновити user
		user.SubscriptionTier = "premium"
		user.SubscriptionExpiresAt = &periodEnd
		if err := s.userRepo.Update(user); err != nil {
			log.Printf("Failed to update user tier: %v", err)
		}

		return subscription, "", nil
	}

	// Створити Monobank invoice
	invoiceReq := &monobank.InvoiceRequest{
		Amount: amount,
		Ccy:    monobank.CurrencyUAH,
		MerchantPaymInfo: monobank.MerchantPaymInfo{
			Reference:   reference,
			Destination: fmt.Sprintf("Підписка %s - Crypto Opportunities Bot", s.getPlanNameUA(plan)),
			Comment:     fmt.Sprintf("Оплата підписки для користувача %s", user.FirstName),
		},
		RedirectUrl: s.redirectURL,
		WebHookUrl:  s.webhookURL,
		Validity:    3600, // 1 година
		SaveCardData: &monobank.SaveCardData{
			SaveCard: true, // Зберегти картку для auto-renewal
		},
	}

	invoiceResp, err := s.monoClient.CreateInvoice(invoiceReq)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create monobank invoice: %w", err)
	}

	// Оновити підписку з invoice ID
	subscription.MonobankInvoiceID = invoiceResp.InvoiceId
	if err := s.subsRepo.Update(subscription); err != nil {
		return nil, "", fmt.Errorf("failed to update subscription with invoice: %w", err)
	}

	// Створити payment record
	payment := &models.Payment{
		UserID:         userID,
		SubscriptionID: &subscription.ID,
		Provider:       models.PaymentProviderMonobank,
		TransactionID:  invoiceResp.InvoiceId,
		Reference:      reference,
		Amount:         amount,
		Currency:       "UAH",
		Status:         models.PaymentStatusPending,
	}

	if err := s.paymentRepo.Create(payment); err != nil {
		log.Printf("Failed to create payment record: %v", err)
	}

	log.Printf("✅ Subscription created for user %d: %s (invoice: %s)", userID, plan, invoiceResp.InvoiceId)

	return subscription, invoiceResp.PageUrl, nil
}

// HandleWebhook обробляє webhook від Monobank
func (s *Service) HandleWebhook(payload *monobank.WebhookPayload) error {
	log.Printf("📥 Webhook received: invoice=%s, status=%s, reference=%s",
		payload.InvoiceId, payload.Status, payload.Reference)

	// Знайти підписку за reference
	subscription, err := s.subsRepo.GetByReference(payload.Reference)
	if err != nil {
		return fmt.Errorf("failed to find subscription: %w", err)
	}

	if subscription == nil {
		return fmt.Errorf("subscription not found for reference: %s", payload.Reference)
	}

	// Знайти payment
	payment, err := s.paymentRepo.GetByTransactionID(payload.InvoiceId)
	if err != nil {
		return fmt.Errorf("failed to find payment: %w", err)
	}

	// Обробити статус
	switch payload.Status {
	case monobank.StatusSuccess:
		return s.handleSuccessfulPayment(subscription, payment, payload)

	case monobank.StatusFailure:
		return s.handleFailedPayment(subscription, payment, payload)

	case monobank.StatusExpired:
		return s.handleExpiredPayment(subscription, payment)

	default:
		log.Printf("ℹ️ Webhook status: %s (no action needed)", payload.Status)
	}

	return nil
}

func (s *Service) handleSuccessfulPayment(
	subscription *models.Subscription,
	payment *models.Payment,
	payload *monobank.WebhookPayload,
) error {
	log.Printf("✅ Payment successful: subscription=%d, invoice=%s", subscription.ID, payload.InvoiceId)

	// Зберегти wallet ID для auto-renewal
	walletID := ""
	if payload.PaymentInfo != nil {
		walletID = payload.PaymentInfo.WalletId
	}

	// Активувати підписку
	subscription.MarkAsActive(walletID)
	if err := s.subsRepo.Update(subscription); err != nil {
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	// Оновити payment
	if payment != nil {
		payment.MarkAsSuccess()
		if err := s.paymentRepo.Update(payment); err != nil {
			log.Printf("Failed to update payment: %v", err)
		}
	}

	// Оновити user
	user, err := s.userRepo.GetByID(subscription.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.SubscriptionTier = "premium"
	user.SubscriptionExpiresAt = &subscription.CurrentPeriodEnd
	user.SubscriptionStripeID = subscription.MonobankInvoiceID // Зберігаємо для сумісності

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	log.Printf("🎉 User %d upgraded to Premium until %s", user.ID, subscription.CurrentPeriodEnd.Format("2006-01-02"))

	return nil
}

func (s *Service) handleFailedPayment(
	subscription *models.Subscription,
	payment *models.Payment,
	payload *monobank.WebhookPayload,
) error {
	log.Printf("❌ Payment failed: subscription=%d, reason=%s", subscription.ID, payload.FailureReason)

	subscription.MarkAsFailed()
	if err := s.subsRepo.Update(subscription); err != nil {
		return fmt.Errorf("failed to mark subscription as failed: %w", err)
	}

	if payment != nil {
		payment.MarkAsFailed(payload.FailureReason)
		if err := s.paymentRepo.Update(payment); err != nil {
			log.Printf("Failed to update payment: %v", err)
		}
	}

	return nil
}

func (s *Service) handleExpiredPayment(subscription *models.Subscription, payment *models.Payment) error {
	log.Printf("⏱️ Payment expired: subscription=%d", subscription.ID)

	subscription.Status = models.SubscriptionStatusExpired
	if err := s.subsRepo.Update(subscription); err != nil {
		return fmt.Errorf("failed to mark subscription as expired: %w", err)
	}

	if payment != nil {
		payment.MarkAsFailed("Payment link expired")
		if err := s.paymentRepo.Update(payment); err != nil {
			log.Printf("Failed to update payment: %v", err)
		}
	}

	return nil
}

// CancelSubscription скасовує підписку
func (s *Service) CancelSubscription(userID uint, immediately bool, reason string) error {
	subscription, err := s.subsRepo.GetActiveByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if subscription == nil {
		return fmt.Errorf("no active subscription found")
	}

	subscription.MarkAsCanceled(reason, immediately)
	if err := s.subsRepo.Update(subscription); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	// Якщо скасування негайне - оновити user
	if immediately {
		user, err := s.userRepo.GetByID(userID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		user.SubscriptionTier = "free"
		now := time.Now()
		user.SubscriptionExpiresAt = &now

		if err := s.userRepo.Update(user); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		log.Printf("❌ Subscription canceled immediately for user %d", userID)
	} else {
		log.Printf("⏸️ Subscription will cancel at period end for user %d", userID)
	}

	return nil
}

// RenewSubscription продовжує підписку (auto-renewal)
func (s *Service) RenewSubscription(subscriptionID uint) error {
	subscription, err := s.subsRepo.GetByID(subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if subscription == nil {
		return fmt.Errorf("subscription not found")
	}

	if subscription.MonobankWalletID == "" {
		return fmt.Errorf("no saved card for renewal")
	}

	// Генерувати новий reference
	reference := fmt.Sprintf("renewal_%s_%d", uuid.New().String()[:8], subscription.UserID)

	amount := monobank.PlanPrices[subscription.Plan]

	// Створити invoice з збереженою карткою
	invoiceReq := &monobank.InvoiceRequest{
		Amount: amount,
		Ccy:    monobank.CurrencyUAH,
		MerchantPaymInfo: monobank.MerchantPaymInfo{
			Reference:   reference,
			Destination: fmt.Sprintf("Продовження підписки %s", s.getPlanNameUA(subscription.Plan)),
		},
		WebHookUrl: s.webhookURL,
		SaveCardData: &monobank.SaveCardData{
			WalletId: subscription.MonobankWalletID,
		},
	}

	invoiceResp, err := s.monoClient.CreateRecurringPayment(subscription.MonobankWalletID, invoiceReq)
	if err != nil {
		subscription.MarkAsFailed()
		s.subsRepo.Update(subscription)
		return fmt.Errorf("failed to create renewal invoice: %w", err)
	}

	// Створити payment record
	payment := &models.Payment{
		UserID:         subscription.UserID,
		SubscriptionID: &subscription.ID,
		Provider:       models.PaymentProviderMonobank,
		TransactionID:  invoiceResp.InvoiceId,
		Reference:      reference,
		Amount:         amount,
		Currency:       "UAH",
		Status:         models.PaymentStatusPending,
	}

	if err := s.paymentRepo.Create(payment); err != nil {
		log.Printf("Failed to create payment record: %v", err)
	}

	log.Printf("🔄 Renewal initiated for subscription %d (invoice: %s)", subscriptionID, invoiceResp.InvoiceId)

	return nil
}

// CheckExpiredSubscriptions перевіряє та деактивує закінчені підписки
func (s *Service) CheckExpiredSubscriptions() error {
	subscriptions, err := s.subsRepo.ListActive()
	if err != nil {
		return fmt.Errorf("failed to list active subscriptions: %w", err)
	}

	expired := 0
	for _, sub := range subscriptions {
		if sub.IsExpired() && !sub.CancelAtPeriodEnd {
			// Спробувати автопродовження
			if sub.AutoRenew && sub.MonobankWalletID != "" {
				if err := s.RenewSubscription(sub.ID); err != nil {
					log.Printf("Failed to renew subscription %d: %v", sub.ID, err)
					sub.MarkAsExpired()
				} else {
					continue
				}
			} else {
				sub.MarkAsExpired()
			}

			if err := s.subsRepo.Update(sub); err != nil {
				log.Printf("Failed to update expired subscription %d: %v", sub.ID, err)
				continue
			}

			// Downgrade user
			user, err := s.userRepo.GetByID(sub.UserID)
			if err != nil {
				log.Printf("Failed to get user %d: %v", sub.UserID, err)
				continue
			}

			user.SubscriptionTier = "free"
			now := time.Now()
			user.SubscriptionExpiresAt = &now

			if err := s.userRepo.Update(user); err != nil {
				log.Printf("Failed to downgrade user %d: %v", user.ID, err)
			}

			expired++
			log.Printf("⏰ Subscription %d expired for user %d", sub.ID, sub.UserID)
		}
	}

	if expired > 0 {
		log.Printf("📊 Expired %d subscriptions", expired)
	}

	return nil
}

func (s *Service) getPlanNameUA(plan string) string {
	names := map[string]string{
		monobank.PlanPremiumMonthly: "Місячна",
		monobank.PlanPremiumWeekly:  "Тижнева",
		monobank.PlanPremiumYearly:  "Річна",
	}

	if name, ok := names[plan]; ok {
		return name
	}

	return "Premium"
}
