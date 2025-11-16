package repository

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"

	"gorm.io/gorm"
)

type PaymentRepository interface {
	Create(payment *models.Payment) error
	Update(payment *models.Payment) error
	GetByID(id uint) (*models.Payment, error)
	GetByTransactionID(transactionID string) (*models.Payment, error)
	GetByReference(reference string) (*models.Payment, error)
	ListByUserID(userID uint, limit, offset int) ([]*models.Payment, error)
	ListBySubscriptionID(subscriptionID uint) ([]*models.Payment, error)
	Delete(id uint) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(payment *models.Payment) error {
	result := r.db.Create(payment)
	if result.Error != nil {
		return fmt.Errorf("failed to create payment: %w", result.Error)
	}
	return nil
}

func (r *paymentRepository) Update(payment *models.Payment) error {
	result := r.db.Save(payment)
	if result.Error != nil {
		return fmt.Errorf("failed to update payment: %w", result.Error)
	}
	return nil
}

func (r *paymentRepository) GetByID(id uint) (*models.Payment, error) {
	var payment models.Payment
	result := r.db.Preload("User").Preload("Subscription").First(&payment, id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment: %w", result.Error)
	}

	return &payment, nil
}

func (r *paymentRepository) GetByTransactionID(transactionID string) (*models.Payment, error) {
	var payment models.Payment
	result := r.db.Where("transaction_id = ?", transactionID).
		Preload("User").
		Preload("Subscription").
		First(&payment)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment: %w", result.Error)
	}

	return &payment, nil
}

func (r *paymentRepository) GetByReference(reference string) (*models.Payment, error) {
	var payment models.Payment
	result := r.db.Where("reference = ?", reference).
		Preload("User").
		Preload("Subscription").
		First(&payment)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment: %w", result.Error)
	}

	return &payment, nil
}

func (r *paymentRepository) ListByUserID(userID uint, limit, offset int) ([]*models.Payment, error) {
	var payments []*models.Payment
	result := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Preload("Subscription").
		Find(&payments)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list payments: %w", result.Error)
	}

	return payments, nil
}

func (r *paymentRepository) ListBySubscriptionID(subscriptionID uint) ([]*models.Payment, error) {
	var payments []*models.Payment
	result := r.db.Where("subscription_id = ?", subscriptionID).
		Order("created_at DESC").
		Find(&payments)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list payments: %w", result.Error)
	}

	return payments, nil
}

func (r *paymentRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Payment{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete payment: %w", result.Error)
	}
	return nil
}
