package repository

import (
	"crypto-opportunities-bot/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type ClientSessionRepository interface {
	Create(session *models.ClientSession) error
	GetByID(id uint) (*models.ClientSession, error)
	GetBySessionID(sessionID string) (*models.ClientSession, error)
	GetActiveByUserID(userID uint) (*models.ClientSession, error)
	Update(session *models.ClientSession) error
	UpdateHeartbeat(sessionID string) error
	Disconnect(sessionID string) error
	ListActive() ([]*models.ClientSession, error)
	ListByUserID(userID uint) ([]*models.ClientSession, error)
	CountActive() (int64, error)
	CleanupStale(timeout time.Duration) error
}

type ClientSessionRepositoryImpl struct {
	db *gorm.DB
}

func NewClientSessionRepository(db *gorm.DB) ClientSessionRepository {
	return &ClientSessionRepositoryImpl{db: db}
}

func (r *ClientSessionRepositoryImpl) Create(session *models.ClientSession) error {
	return r.db.Create(session).Error
}

func (r *ClientSessionRepositoryImpl) GetByID(id uint) (*models.ClientSession, error) {
	var session models.ClientSession
	err := r.db.Preload("User").First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *ClientSessionRepositoryImpl) GetBySessionID(sessionID string) (*models.ClientSession, error) {
	var session models.ClientSession
	err := r.db.Preload("User").Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *ClientSessionRepositoryImpl) GetActiveByUserID(userID uint) (*models.ClientSession, error) {
	var session models.ClientSession
	err := r.db.Preload("User").
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("connected_at DESC").
		First(&session).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *ClientSessionRepositoryImpl) Update(session *models.ClientSession) error {
	return r.db.Save(session).Error
}

func (r *ClientSessionRepositoryImpl) UpdateHeartbeat(sessionID string) error {
	return r.db.Model(&models.ClientSession{}).
		Where("session_id = ?", sessionID).
		Update("last_heartbeat", time.Now()).Error
}

func (r *ClientSessionRepositoryImpl) Disconnect(sessionID string) error {
	now := time.Now()
	return r.db.Model(&models.ClientSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"is_active":       false,
			"disconnected_at": now,
		}).Error
}

func (r *ClientSessionRepositoryImpl) ListActive() ([]*models.ClientSession, error) {
	var sessions []*models.ClientSession
	err := r.db.Preload("User").
		Where("is_active = ?", true).
		Order("connected_at DESC").
		Find(&sessions).Error

	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *ClientSessionRepositoryImpl) ListByUserID(userID uint) ([]*models.ClientSession, error) {
	var sessions []*models.ClientSession
	err := r.db.Where("user_id = ?", userID).
		Order("connected_at DESC").
		Limit(10).
		Find(&sessions).Error

	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *ClientSessionRepositoryImpl) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&models.ClientSession{}).
		Where("is_active = ?", true).
		Count(&count).Error

	if err != nil {
		return 0, err
	}
	return count, nil
}

// CleanupStale деактивує сесії які не отримували heartbeat більше timeout
func (r *ClientSessionRepositoryImpl) CleanupStale(timeout time.Duration) error {
	cutoff := time.Now().Add(-timeout)
	now := time.Now()

	return r.db.Model(&models.ClientSession{}).
		Where("is_active = ? AND last_heartbeat < ?", true, cutoff).
		Updates(map[string]interface{}{
			"is_active":       false,
			"disconnected_at": now,
		}).Error
}
