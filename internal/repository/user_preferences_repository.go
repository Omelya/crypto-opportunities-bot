package repository

import (
	"crypto-opportunities-bot/internal/models"
	"errors"

	"gorm.io/gorm"
)

type UserPreferencesRepository interface {
	Create(prefs *models.UserPreferences) error
	GetByUserID(userID uint) (*models.UserPreferences, error)
	Update(prefs *models.UserPreferences) error
}

type UserPreferencesRepositoryImpl struct {
	db *gorm.DB
}

func NewUserPreferencesRepository(db *gorm.DB) UserPreferencesRepository {
	return &UserPreferencesRepositoryImpl{db: db}
}

func (r *UserPreferencesRepositoryImpl) Create(prefs *models.UserPreferences) error {
	return r.db.Create(prefs).Error
}

func (r *UserPreferencesRepositoryImpl) GetByUserID(userID uint) (*models.UserPreferences, error) {
	var prefs models.UserPreferences
	err := r.db.Where("user_id = ?", userID).First(&prefs).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &prefs, nil
}

func (r *UserPreferencesRepositoryImpl) Update(prefs *models.UserPreferences) error {
	return r.db.Save(prefs).Error
}
