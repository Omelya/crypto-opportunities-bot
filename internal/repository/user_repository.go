package repository

import (
	"crypto-opportunities-bot/internal/models"
	"errors"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByTelegramID(telegramID int64) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	List(offset, limit int) ([]*models.User, error)
	Count() (int64, error)
}

type UserRepositoryImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{db: db}
}

func (u *UserRepositoryImpl) Create(user *models.User) error {
	return u.db.Create(user).Error
}

func (u *UserRepositoryImpl) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := u.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UserRepositoryImpl) GetByTelegramID(telegramID int64) (*models.User, error) {
	var user models.User
	err := u.db.Where("telegram_id = ?", telegramID).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (u *UserRepositoryImpl) Update(user *models.User) error {
	return u.db.Save(user).Error
}

func (u *UserRepositoryImpl) Delete(id uint) error {
	return u.db.Delete(&models.User{}, id).Error
}

func (u *UserRepositoryImpl) List(offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := u.db.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (u *UserRepositoryImpl) Count() (int64, error) {
	var count int64
	err := u.db.Model(&models.User{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
