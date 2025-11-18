package repository

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ReferralRepository interface {
	// Referral CRUD
	CreateReferral(referral *models.Referral) error
	GetReferralByID(id uint) (*models.Referral, error)
	GetReferralByUsers(referrerID, referredID uint) (*models.Referral, error)
	UpdateReferral(referral *models.Referral) error
	DeleteReferral(id uint) error

	// Referral queries
	GetReferralsByReferrer(referrerID uint) ([]*models.Referral, error)
	GetReferralsByReferred(referredID uint) ([]*models.Referral, error)
	GetReferralsByStatus(status string) ([]*models.Referral, error)
	GetPendingRewards(userID uint) ([]*models.Referral, error)
	CountReferralsByReferrer(referrerID uint) (int64, error)
	CountActiveReferrals(referrerID uint) (int64, error)

	// Referral actions
	ActivateReferral(referralID uint) error
	CompleteReferral(referralID uint) error
	ExpireOldReferrals() error

	// ReferralCode CRUD
	CreateReferralCode(code *models.ReferralCode) error
	GetReferralCodeByCode(code string) (*models.ReferralCode, error)
	GetReferralCodeByID(id uint) (*models.ReferralCode, error)
	GetReferralCodesByOwner(ownerID uint) ([]*models.ReferralCode, error)
	UpdateReferralCode(code *models.ReferralCode) error
	DeleteReferralCode(id uint) error
	IncrementCodeUses(code string) error

	// ReferralReward CRUD
	CreateReward(reward *models.ReferralReward) error
	GetRewardByID(id uint) (*models.ReferralReward, error)
	GetRewardsByUser(userID uint) ([]*models.ReferralReward, error)
	GetPendingRewardsByUser(userID uint) ([]*models.ReferralReward, error)
	UpdateReward(reward *models.ReferralReward) error
	ClaimReward(rewardID uint) error

	// Statistics
	GetReferralStats(userID uint) (*models.ReferralStats, error)
}

type referralRepository struct {
	db *gorm.DB
}

func NewReferralRepository(db *gorm.DB) ReferralRepository {
	return &referralRepository{db: db}
}

// Referral CRUD
func (r *referralRepository) CreateReferral(referral *models.Referral) error {
	return r.db.Create(referral).Error
}

func (r *referralRepository) GetReferralByID(id uint) (*models.Referral, error) {
	var referral models.Referral
	err := r.db.First(&referral, id).Error
	if err != nil {
		return nil, err
	}
	return &referral, nil
}

func (r *referralRepository) GetReferralByUsers(referrerID, referredID uint) (*models.Referral, error) {
	var referral models.Referral
	err := r.db.Where("referrer_id = ? AND referred_id = ?", referrerID, referredID).First(&referral).Error
	if err != nil {
		return nil, err
	}
	return &referral, nil
}

func (r *referralRepository) UpdateReferral(referral *models.Referral) error {
	return r.db.Save(referral).Error
}

func (r *referralRepository) DeleteReferral(id uint) error {
	return r.db.Delete(&models.Referral{}, id).Error
}

// Referral queries
func (r *referralRepository) GetReferralsByReferrer(referrerID uint) ([]*models.Referral, error) {
	var referrals []*models.Referral
	err := r.db.Where("referrer_id = ?", referrerID).Order("created_at DESC").Find(&referrals).Error
	return referrals, err
}

func (r *referralRepository) GetReferralsByReferred(referredID uint) ([]*models.Referral, error) {
	var referrals []*models.Referral
	err := r.db.Where("referred_id = ?", referredID).Order("created_at DESC").Find(&referrals).Error
	return referrals, err
}

func (r *referralRepository) GetReferralsByStatus(status string) ([]*models.Referral, error) {
	var referrals []*models.Referral
	err := r.db.Where("status = ?", status).Order("created_at DESC").Find(&referrals).Error
	return referrals, err
}

func (r *referralRepository) GetPendingRewards(userID uint) ([]*models.Referral, error) {
	var referrals []*models.Referral
	err := r.db.Where("referrer_id = ? AND status = ? AND reward_issued = ?",
		userID, models.ReferralStatusActive, false).Find(&referrals).Error
	return referrals, err
}

func (r *referralRepository) CountReferralsByReferrer(referrerID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Referral{}).Where("referrer_id = ?", referrerID).Count(&count).Error
	return count, err
}

func (r *referralRepository) CountActiveReferrals(referrerID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Referral{}).
		Where("referrer_id = ? AND status = ?", referrerID, models.ReferralStatusActive).
		Count(&count).Error
	return count, err
}

// Referral actions
func (r *referralRepository) ActivateReferral(referralID uint) error {
	now := time.Now()
	return r.db.Model(&models.Referral{}).Where("id = ?", referralID).Updates(map[string]interface{}{
		"status":       models.ReferralStatusActive,
		"activated_at": now,
	}).Error
}

func (r *referralRepository) CompleteReferral(referralID uint) error {
	now := time.Now()
	return r.db.Model(&models.Referral{}).Where("id = ?", referralID).Updates(map[string]interface{}{
		"status":           models.ReferralStatusCompleted,
		"reward_issued":    true,
		"reward_issued_at": now,
	}).Error
}

func (r *referralRepository) ExpireOldReferrals() error {
	now := time.Now()
	return r.db.Model(&models.Referral{}).
		Where("status = ? AND expires_at < ?", models.ReferralStatusPending, now).
		Update("status", models.ReferralStatusExpired).Error
}

// ReferralCode CRUD
func (r *referralRepository) CreateReferralCode(code *models.ReferralCode) error {
	return r.db.Create(code).Error
}

func (r *referralRepository) GetReferralCodeByCode(code string) (*models.ReferralCode, error) {
	var refCode models.ReferralCode
	err := r.db.Where("code = ?", code).First(&refCode).Error
	if err != nil {
		return nil, err
	}
	return &refCode, nil
}

func (r *referralRepository) GetReferralCodeByID(id uint) (*models.ReferralCode, error) {
	var code models.ReferralCode
	err := r.db.First(&code, id).Error
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (r *referralRepository) GetReferralCodesByOwner(ownerID uint) ([]*models.ReferralCode, error) {
	var codes []*models.ReferralCode
	err := r.db.Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&codes).Error
	return codes, err
}

func (r *referralRepository) UpdateReferralCode(code *models.ReferralCode) error {
	return r.db.Save(code).Error
}

func (r *referralRepository) DeleteReferralCode(id uint) error {
	return r.db.Delete(&models.ReferralCode{}, id).Error
}

func (r *referralRepository) IncrementCodeUses(code string) error {
	return r.db.Model(&models.ReferralCode{}).
		Where("code = ?", code).
		Update("current_uses", gorm.Expr("current_uses + ?", 1)).Error
}

// ReferralReward CRUD
func (r *referralRepository) CreateReward(reward *models.ReferralReward) error {
	return r.db.Create(reward).Error
}

func (r *referralRepository) GetRewardByID(id uint) (*models.ReferralReward, error) {
	var reward models.ReferralReward
	err := r.db.First(&reward, id).Error
	if err != nil {
		return nil, err
	}
	return &reward, nil
}

func (r *referralRepository) GetRewardsByUser(userID uint) ([]*models.ReferralReward, error) {
	var rewards []*models.ReferralReward
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&rewards).Error
	return rewards, err
}

func (r *referralRepository) GetPendingRewardsByUser(userID uint) ([]*models.ReferralReward, error) {
	var rewards []*models.ReferralReward
	err := r.db.Where("user_id = ? AND status = ?", userID, "pending").
		Order("created_at DESC").Find(&rewards).Error
	return rewards, err
}

func (r *referralRepository) UpdateReward(reward *models.ReferralReward) error {
	return r.db.Save(reward).Error
}

func (r *referralRepository) ClaimReward(rewardID uint) error {
	now := time.Now()
	return r.db.Model(&models.ReferralReward{}).Where("id = ?", rewardID).Updates(map[string]interface{}{
		"status":     "claimed",
		"claimed_at": now,
	}).Error
}

// Statistics
func (r *referralRepository) GetReferralStats(userID uint) (*models.ReferralStats, error) {
	stats := &models.ReferralStats{
		UserID: userID,
	}

	// Total referrals
	var totalCount int64
	if err := r.db.Model(&models.Referral{}).Where("referrer_id = ?", userID).Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count total referrals: %w", err)
	}
	stats.TotalReferrals = int(totalCount)

	// Active referrals
	var activeCount int64
	if err := r.db.Model(&models.Referral{}).
		Where("referrer_id = ? AND status = ?", userID, models.ReferralStatusActive).
		Count(&activeCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count active referrals: %w", err)
	}
	stats.ActiveReferrals = int(activeCount)

	// Completed referrals
	var completedCount int64
	if err := r.db.Model(&models.Referral{}).
		Where("referrer_id = ? AND status = ?", userID, models.ReferralStatusCompleted).
		Count(&completedCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count completed referrals: %w", err)
	}
	stats.CompletedReferrals = int(completedCount)

	// Total rewards earned
	var totalRewards int64
	if err := r.db.Model(&models.ReferralReward{}).
		Where("user_id = ? AND status = ?", userID, "claimed").
		Count(&totalRewards).Error; err != nil {
		return nil, fmt.Errorf("failed to count total rewards: %w", err)
	}
	stats.TotalRewardsEarned = int(totalRewards)

	// Pending rewards
	var pendingRewards int64
	if err := r.db.Model(&models.ReferralReward{}).
		Where("user_id = ? AND status = ?", userID, "pending").
		Count(&pendingRewards).Error; err != nil {
		return nil, fmt.Errorf("failed to count pending rewards: %w", err)
	}
	stats.PendingRewards = int(pendingRewards)

	// Custom code info
	var code models.ReferralCode
	if err := r.db.Where("owner_id = ? AND is_active = ?", userID, true).First(&code).Error; err == nil {
		stats.CustomCode = code.Code
		stats.CustomCodeUses = code.CurrentUses
	}

	return stats, nil
}
