package models

import (
	"golang.org/x/crypto/bcrypt"
)

// AdminRole представляє роль адміністратора
type AdminRole string

const (
	AdminRoleSuperAdmin AdminRole = "super_admin" // Повний доступ
	AdminRoleAdmin      AdminRole = "admin"       // Управління користувачами та контентом
	AdminRoleViewer     AdminRole = "viewer"      // Тільки перегляд статистики
)

// AdminUser представляє адміністратора системи
type AdminUser struct {
	BaseModel
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"` // Не включаємо в JSON
	Email        string    `gorm:"uniqueIndex" json:"email"`
	Role         AdminRole `gorm:"type:varchar(50);not null;default:'viewer'" json:"role"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	LastLoginAt  *Time     `json:"last_login_at,omitempty"`
}

// TableName встановлює назву таблиці
func (AdminUser) TableName() string {
	return "admin_users"
}

// SetPassword хешує пароль і зберігає його
func (u *AdminUser) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword перевіряє чи пароль правильний
func (u *AdminUser) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsSuperAdmin перевіряє чи користувач super admin
func (u *AdminUser) IsSuperAdmin() bool {
	return u.Role == AdminRoleSuperAdmin
}

// IsAdmin перевіряє чи користувач admin або вище
func (u *AdminUser) IsAdmin() bool {
	return u.Role == AdminRoleSuperAdmin || u.Role == AdminRoleAdmin
}

// IsViewer перевіряє чи користувач має хоча б viewer права
func (u *AdminUser) IsViewer() bool {
	return u.Role == AdminRoleSuperAdmin || u.Role == AdminRoleAdmin || u.Role == AdminRoleViewer
}

// CanManageUsers перевіряє чи може користувач управляти іншими користувачами
func (u *AdminUser) CanManageUsers() bool {
	return u.IsSuperAdmin() || u.IsAdmin()
}

// CanModifyData перевіряє чи може користувач змінювати дані
func (u *AdminUser) CanModifyData() bool {
	return u.IsSuperAdmin() || u.IsAdmin()
}
