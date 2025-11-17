package repository

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AdminRepository інтерфейс для роботи з адміністраторами
type AdminRepository interface {
	Create(admin *models.AdminUser) error
	Update(admin *models.AdminUser) error
	GetByID(id uint) (*models.AdminUser, error)
	GetByUsername(username string) (*models.AdminUser, error)
	GetByEmail(email string) (*models.AdminUser, error)
	List(limit, offset int) ([]*models.AdminUser, error)
	Delete(id uint) error
	CountAll() (int64, error)
	UpdateLastLogin(id uint) error
}

type adminRepository struct {
	db *gorm.DB
}

// NewAdminRepository створює новий AdminRepository
func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

// Create створює нового адміністратора
func (r *adminRepository) Create(admin *models.AdminUser) error {
	return r.db.Create(admin).Error
}

// Update оновлює адміністратора
func (r *adminRepository) Update(admin *models.AdminUser) error {
	return r.db.Save(admin).Error
}

// GetByID отримує адміністратора за ID
func (r *adminRepository) GetByID(id uint) (*models.AdminUser, error) {
	var admin models.AdminUser
	err := r.db.First(&admin, id).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByUsername отримує адміністратора за username
func (r *adminRepository) GetByUsername(username string) (*models.AdminUser, error) {
	var admin models.AdminUser
	err := r.db.Where("username = ?", username).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByEmail отримує адміністратора за email
func (r *adminRepository) GetByEmail(email string) (*models.AdminUser, error) {
	var admin models.AdminUser
	err := r.db.Where("email = ?", email).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// List повертає список адміністраторів з пагінацією
func (r *adminRepository) List(limit, offset int) ([]*models.AdminUser, error) {
	var admins []*models.AdminUser
	err := r.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&admins).Error
	if err != nil {
		return nil, err
	}
	return admins, nil
}

// Delete видаляє адміністратора (soft delete)
func (r *adminRepository) Delete(id uint) error {
	return r.db.Delete(&models.AdminUser{}, id).Error
}

// CountAll підраховує загальну кількість адміністраторів
func (r *adminRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&models.AdminUser{}).Count(&count).Error
	return count, err
}

// UpdateLastLogin оновлює час останнього входу
func (r *adminRepository) UpdateLastLogin(id uint) error {
	now := models.Time(time.Now())
	return r.db.Model(&models.AdminUser{}).Where("id = ?", id).
		Update("last_login_at", now).Error
}

// MigrateAdminTables виконує міграцію таблиць для admin панелі
func MigrateAdminTables(db *gorm.DB) error {
	return db.AutoMigrate(&models.AdminUser{})
}

// CreateDefaultAdmin створює admin за замовчуванням (для першого запуску)
func CreateDefaultAdmin(db *gorm.DB, username, password, email string) error {
	repo := NewAdminRepository(db)

	// Перевірити чи існує адміністратор
	existingAdmin, _ := repo.GetByUsername(username)
	if existingAdmin != nil {
		return fmt.Errorf("admin with username %s already exists", username)
	}

	// Створити нового адміністратора
	admin := &models.AdminUser{
		Username: username,
		Email:    email,
		Role:     models.AdminRoleSuperAdmin,
		IsActive: true,
	}

	if err := admin.SetPassword(password); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return repo.Create(admin)
}
