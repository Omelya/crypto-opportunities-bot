package referral

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/repository"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"
)

type Service struct {
	referralRepo repository.ReferralRepository
	userRepo     repository.UserRepository
	subsRepo     repository.SubscriptionRepository
}

func NewService(
	referralRepo repository.ReferralRepository,
	userRepo repository.UserRepository,
	subsRepo repository.SubscriptionRepository,
) *Service {
	return &Service{
		referralRepo: referralRepo,
		userRepo:     userRepo,
		subsRepo:     subsRepo,
	}
}

// GenerateReferralCode generates a unique referral code for a user
func (s *Service) GenerateReferralCode(userID uint) (string, error) {
	// Generate 6-character alphanumeric code
	for attempts := 0; attempts < 10; attempts++ {
		code := generateRandomCode(6)

		// Check if code already exists
		_, err := s.referralRepo.GetReferralCodeByCode(code)
		if err != nil {
			// Code doesn't exist, we can use it
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after 10 attempts")
}

// CreateUserReferralCode creates a default referral code for a user
func (s *Service) CreateUserReferralCode(userID uint) (*models.ReferralCode, error) {
	// Check if user already has a code
	existingCodes, err := s.referralRepo.GetReferralCodesByOwner(userID)
	if err == nil && len(existingCodes) > 0 {
		return existingCodes[0], nil
	}

	code, err := s.GenerateReferralCode(userID)
	if err != nil {
		return nil, err
	}

	refCode := &models.ReferralCode{
		Code:                code,
		OwnerID:             userID,
		Description:         "Personal referral code",
		MaxUses:             0, // Unlimited
		CurrentUses:         0,
		IsActive:            true,
		IsPublic:            true,
		ReferrerRewardType:  models.ReferralRewardTypePremium,
		ReferrerRewardValue: 30, // 30 days premium
		FriendBenefitType:   "discount_percent",
		FriendBenefitValue:  20, // 20% discount
	}

	if err := s.referralRepo.CreateReferralCode(refCode); err != nil {
		return nil, err
	}

	// Update user's referral code
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	user.ReferralCode = code
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return refCode, nil
}

// ProcessReferral processes a referral when a new user registers with a code
func (s *Service) ProcessReferral(referredUserID uint, referralCode string) error {
	// Get referral code
	refCode, err := s.referralRepo.GetReferralCodeByCode(referralCode)
	if err != nil {
		return fmt.Errorf("referral code not found: %w", err)
	}

	// Check if code can be used
	if !refCode.CanUse() {
		return fmt.Errorf("referral code cannot be used (expired or limit reached)")
	}

	// Check if user is not referring themselves
	if refCode.OwnerID == referredUserID {
		return fmt.Errorf("cannot use own referral code")
	}

	// Check if user was already referred
	referredUser, err := s.userRepo.GetByID(referredUserID)
	if err != nil {
		return err
	}

	if referredUser.ReferredByID != nil {
		return fmt.Errorf("user already has a referrer")
	}

	// Create referral record
	expiresAt := time.Now().AddDate(0, 3, 0) // 3 months expiry
	referral := &models.Referral{
		ReferrerID:          refCode.OwnerID,
		ReferredID:          referredUserID,
		Code:                referralCode,
		Status:              models.ReferralStatusPending,
		RewardType:          refCode.ReferrerRewardType,
		FriendBenefitType:   refCode.FriendBenefitType,
		ExpiresAt:           &expiresAt,
	}

	if err := s.referralRepo.CreateReferral(referral); err != nil {
		return err
	}

	// Update referred user
	ownerID := refCode.OwnerID
	referredUser.ReferredByID = &ownerID
	if err := s.userRepo.Update(referredUser); err != nil {
		return err
	}

	// Increment code uses
	if err := s.referralRepo.IncrementCodeUses(referralCode); err != nil {
		log.Printf("Error incrementing code uses: %v", err)
	}

	// Apply friend benefit (20% discount on first month)
	if refCode.FriendBenefitType == "discount_percent" {
		log.Printf("✅ Friend benefit applied: %d%% discount for user %d",
			refCode.FriendBenefitValue, referredUserID)
		// The discount will be applied during payment processing
	}

	log.Printf("✅ Referral created: User %d referred by User %d", referredUserID, refCode.OwnerID)

	return nil
}

// ActivateReferral activates a referral when the referred user subscribes to Premium
func (s *Service) ActivateReferral(referredUserID uint) error {
	// Find pending referral for this user
	referrals, err := s.referralRepo.GetReferralsByReferred(referredUserID)
	if err != nil || len(referrals) == 0 {
		return fmt.Errorf("no referral found for user %d", referredUserID)
	}

	var pendingReferral *models.Referral
	for _, ref := range referrals {
		if ref.Status == models.ReferralStatusPending {
			pendingReferral = ref
			break
		}
	}

	if pendingReferral == nil {
		return fmt.Errorf("no pending referral for user %d", referredUserID)
	}

	// Activate referral
	if err := s.referralRepo.ActivateReferral(pendingReferral.ID); err != nil {
		return err
	}

	// Issue reward to referrer
	if err := s.IssueReferrerReward(pendingReferral.ReferrerID, pendingReferral.ID); err != nil {
		log.Printf("Error issuing referrer reward: %v", err)
		// Don't fail activation if reward issuance fails
	}

	log.Printf("✅ Referral activated: User %d → User %d",
		pendingReferral.ReferrerID, referredUserID)

	return nil
}

// IssueReferrerReward issues reward to the referrer
func (s *Service) IssueReferrerReward(referrerID uint, referralID uint) error {
	// Get referral to know reward type
	referral, err := s.referralRepo.GetReferralByID(referralID)
	if err != nil {
		return err
	}

	// Create reward
	expiresAt := time.Now().AddDate(0, 6, 0) // 6 months to claim
	reward := &models.ReferralReward{
		UserID:     referrerID,
		ReferralID: referralID,
		Type:       referral.RewardType,
		Value:      30, // 30 days premium
		Status:     "issued",
		IssuedAt:   ptrTime(time.Now()),
		ExpiresAt:  &expiresAt,
	}

	if err := s.referralRepo.CreateReward(reward); err != nil {
		return err
	}

	// Auto-claim and apply reward (extend premium subscription)
	if err := s.ClaimReward(reward.ID); err != nil {
		log.Printf("Error auto-claiming reward: %v", err)
	}

	log.Printf("✅ Reward issued: User %d gets %d days premium", referrerID, reward.Value)

	return nil
}

// ClaimReward claims a reward and applies it to user's subscription
func (s *Service) ClaimReward(rewardID uint) error {
	reward, err := s.referralRepo.GetRewardByID(rewardID)
	if err != nil {
		return err
	}

	if reward.IsClaimed() {
		return fmt.Errorf("reward already claimed")
	}

	if reward.IsExpired() {
		return fmt.Errorf("reward expired")
	}

	// Apply reward based on type
	if reward.Type == models.ReferralRewardTypePremium {
		if err := s.applyPremiumReward(reward.UserID, reward.Value); err != nil {
			return err
		}
	}

	// Mark as claimed
	if err := s.referralRepo.ClaimReward(rewardID); err != nil {
		return err
	}

	log.Printf("✅ Reward claimed: User %d claimed %d days premium", reward.UserID, reward.Value)

	return nil
}

// applyPremiumReward extends user's premium subscription
func (s *Service) applyPremiumReward(userID uint, days int) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	// Calculate new expiration date
	var newExpiresAt time.Time
	if user.SubscriptionExpiresAt != nil && user.SubscriptionExpiresAt.After(time.Now()) {
		// Extend existing subscription
		newExpiresAt = user.SubscriptionExpiresAt.Add(time.Duration(days) * 24 * time.Hour)
	} else {
		// Start new subscription
		newExpiresAt = time.Now().Add(time.Duration(days) * 24 * time.Hour)
		user.SubscriptionTier = "premium"
	}

	user.SubscriptionExpiresAt = &newExpiresAt

	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	log.Printf("✅ Premium extended: User %d until %s", userID, newExpiresAt.Format("2006-01-02"))

	return nil
}

// GetUserReferralStats gets referral statistics for a user
func (s *Service) GetUserReferralStats(userID uint) (*models.ReferralStats, error) {
	return s.referralRepo.GetReferralStats(userID)
}

// GetUserReferralCode gets or creates a referral code for a user
func (s *Service) GetUserReferralCode(userID uint) (string, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return "", err
	}

	// If user already has a code, return it
	if user.ReferralCode != "" {
		return user.ReferralCode, nil
	}

	// Create new code
	refCode, err := s.CreateUserReferralCode(userID)
	if err != nil {
		return "", err
	}

	return refCode.Code, nil
}

// ExpireOldReferrals expires referrals older than 3 months with no activation
func (s *Service) ExpireOldReferrals() error {
	return s.referralRepo.ExpireOldReferrals()
}

// Helper functions
func generateRandomCode(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().Unix())
	}

	code := base64.URLEncoding.EncodeToString(b)
	code = strings.ToUpper(code)
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, "_", "")

	if len(code) > length {
		code = code[:length]
	}

	return code
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
