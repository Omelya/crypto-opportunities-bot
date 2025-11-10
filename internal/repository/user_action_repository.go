package repository

import (
	"crypto-opportunities-bot/internal/models"

	"gorm.io/gorm"
)

type UserActionRepository interface {
	Create(action *models.UserAction) error
	CountByUserAndType(userID uint, actionType string, opportunityID uint) (int64, error)
}

type userActionRepository struct {
	db *gorm.DB
}

func NewUserActionRepository(db *gorm.DB) UserActionRepository {
	return &userActionRepository{db: db}
}

func (r *userActionRepository) Create(action *models.UserAction) error {
	return r.db.Create(action).Error
}

func (r *userActionRepository) CountByUserAndType(userID uint, actionType string, opportunityID uint) (int64, error) {
	var count int64
	query := r.db.Model(&models.UserAction{}).
		Where("user_id = ? AND action_type = ?", userID, actionType)

	if opportunityID > 0 {
		query = query.Where("opportunity_id = ?", opportunityID)
	}

	err := query.Count(&count).Error
	return count, err
}
