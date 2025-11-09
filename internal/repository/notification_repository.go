package repository

import (
	"crypto-opportunities-bot/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(notification *models.Notification) error
	GetByID(id uint) (*models.Notification, error)
	Update(notification *models.Notification) error
	Delete(id uint) error
	GetPending(limit int) ([]*models.Notification, error)
	GetPendingForUser(userID uint, limit int) ([]*models.Notification, error)
	GetScheduled(before time.Time, limit int) ([]*models.Notification, error)
	GetFailed(limit int) ([]*models.Notification, error)
	CountByStatus(status string) (int64, error)
	CountByUserAndStatus(userID uint, status string) (int64, error)
	DeleteOld(days int) error
	DeleteByUserID(userID uint) error
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(notification *models.Notification) error {
	return r.db.Create(notification).Error
}

func (r *notificationRepository) GetByID(id uint) (*models.Notification, error) {
	var notification models.Notification
	err := r.db.Preload("User").Preload("Opportunity").First(&notification, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &notification, nil
}

func (r *notificationRepository) Update(notification *models.Notification) error {
	return r.db.Save(notification).Error
}

func (r *notificationRepository) Delete(id uint) error {
	return r.db.Delete(&models.Notification{}, id).Error
}

func (r *notificationRepository) GetPending(limit int) ([]*models.Notification, error) {
	var notifications []*models.Notification

	err := r.db.
		Preload("User").
		Preload("Opportunity").
		Where("status = ?", models.NotificationStatusPending).
		Where("scheduled_for IS NULL OR scheduled_for <= ?", time.Now()).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&notifications).Error

	return notifications, err
}

func (r *notificationRepository) GetPendingForUser(userID uint, limit int) ([]*models.Notification, error) {
	var notifications []*models.Notification

	err := r.db.
		Preload("Opportunity").
		Where("user_id = ?", userID).
		Where("status = ?", models.NotificationStatusPending).
		Where("scheduled_for IS NULL OR scheduled_for <= ?", time.Now()).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&notifications).Error

	return notifications, err
}

func (r *notificationRepository) GetScheduled(before time.Time, limit int) ([]*models.Notification, error) {
	var notifications []*models.Notification

	err := r.db.
		Preload("User").
		Preload("Opportunity").
		Where("status = ?", models.NotificationStatusPending).
		Where("scheduled_for IS NOT NULL").
		Where("scheduled_for <= ?", before).
		Order("scheduled_for ASC").
		Limit(limit).
		Find(&notifications).Error

	return notifications, err
}

func (r *notificationRepository) GetFailed(limit int) ([]*models.Notification, error) {
	var notifications []*models.Notification

	err := r.db.
		Preload("User").
		Preload("Opportunity").
		Where("status = ?", models.NotificationStatusFailed).
		Where("retry_count < max_retries").
		Order("created_at ASC").
		Limit(limit).
		Find(&notifications).Error

	return notifications, err
}

func (r *notificationRepository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Notification{}).
		Where("status = ?", status).
		Count(&count).Error

	return count, err
}

func (r *notificationRepository) CountByUserAndStatus(userID uint, status string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Notification{}).
		Where("user_id = ? AND status = ?", userID, status).
		Count(&count).Error

	return count, err
}

func (r *notificationRepository) DeleteOld(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)

	return r.db.
		Where("created_at < ?", cutoff).
		Where("status = ?", models.NotificationStatusSent).
		Delete(&models.Notification{}).Error
}

func (r *notificationRepository) DeleteByUserID(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.Notification{}).Error
}
