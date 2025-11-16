package repository

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type SubscriptionRepository interface {
	Create(subscription *models.Subscription) error
	Update(subscription *models.Subscription) error
	GetByID(id uint) (*models.Subscription, error)
	GetByUserID(userID uint) (*models.Subscription, error)
	GetActiveByUserID(userID uint) (*models.Subscription, error)
	GetByMonobankInvoiceID(invoiceID string) (*models.Subscription, error)
	GetByReference(reference string) (*models.Subscription, error)
	ListActive() ([]*models.Subscription, error)
	ListExpiring(days int) ([]*models.Subscription, error)
	ListNeedRenewal() ([]*models.Subscription, error)
	Delete(id uint) error
}

type subscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(subscription *models.Subscription) error {
	result := r.db.Create(subscription)
	if result.Error != nil {
		return fmt.Errorf("failed to create subscription: %w", result.Error)
	}
	return nil
}

func (r *subscriptionRepository) Update(subscription *models.Subscription) error {
	result := r.db.Save(subscription)
	if result.Error != nil {
		return fmt.Errorf("failed to update subscription: %w", result.Error)
	}
	return nil
}

func (r *subscriptionRepository) GetByID(id uint) (*models.Subscription, error) {
	var subscription models.Subscription
	result := r.db.Preload("User").First(&subscription, id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subscription: %w", result.Error)
	}

	return &subscription, nil
}

func (r *subscriptionRepository) GetByUserID(userID uint) (*models.Subscription, error) {
	var subscription models.Subscription
	result := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Preload("User").
		First(&subscription)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subscription: %w", result.Error)
	}

	return &subscription, nil
}

func (r *subscriptionRepository) GetActiveByUserID(userID uint) (*models.Subscription, error) {
	var subscription models.Subscription
	result := r.db.Where("user_id = ? AND status = ?", userID, models.SubscriptionStatusActive).
		Where("current_period_end > ?", time.Now()).
		Preload("User").
		First(&subscription)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active subscription: %w", result.Error)
	}

	return &subscription, nil
}

func (r *subscriptionRepository) GetByMonobankInvoiceID(invoiceID string) (*models.Subscription, error) {
	var subscription models.Subscription
	result := r.db.Where("monobank_invoice_id = ?", invoiceID).
		Preload("User").
		First(&subscription)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subscription: %w", result.Error)
	}

	return &subscription, nil
}

func (r *subscriptionRepository) GetByReference(reference string) (*models.Subscription, error) {
	var subscription models.Subscription
	result := r.db.Where("monobank_reference = ?", reference).
		Preload("User").
		First(&subscription)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subscription: %w", result.Error)
	}

	return &subscription, nil
}

func (r *subscriptionRepository) ListActive() ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	result := r.db.Where("status = ?", models.SubscriptionStatusActive).
		Where("current_period_end > ?", time.Now()).
		Preload("User").
		Find(&subscriptions)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list active subscriptions: %w", result.Error)
	}

	return subscriptions, nil
}

func (r *subscriptionRepository) ListExpiring(days int) ([]*models.Subscription, error) {
	expiryDate := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	var subscriptions []*models.Subscription
	result := r.db.Where("status = ?", models.SubscriptionStatusActive).
		Where("current_period_end <= ? AND current_period_end > ?", expiryDate, time.Now()).
		Where("auto_renew = ?", true).
		Preload("User").
		Find(&subscriptions)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list expiring subscriptions: %w", result.Error)
	}

	return subscriptions, nil
}

func (r *subscriptionRepository) ListNeedRenewal() ([]*models.Subscription, error) {
	renewalDate := time.Now().Add(48 * time.Hour) // Renewal за 2 дні до закінчення

	var subscriptions []*models.Subscription
	result := r.db.Where("status = ?", models.SubscriptionStatusActive).
		Where("current_period_end <= ?", renewalDate).
		Where("current_period_end > ?", time.Now()).
		Where("auto_renew = ?", true).
		Where("cancel_at_period_end = ?", false).
		Preload("User").
		Find(&subscriptions)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list subscriptions need renewal: %w", result.Error)
	}

	return subscriptions, nil
}

func (r *subscriptionRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Subscription{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete subscription: %w", result.Error)
	}
	return nil
}
