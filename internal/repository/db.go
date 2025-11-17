package repository

import (
	"crypto-opportunities-bot/internal/config"
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDatabase(cfg config.DatabaseConfig, app config.AppConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode,
	)

	var logLevel logger.LogLevel
	if app.Environment == "development" {
		logLevel = logger.Info
	} else {
		logLevel = logger.Error
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, err
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.MaxConns)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	pingErr := sqlDB.Ping()
	if pingErr != nil {
		return nil, pingErr
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.UserPreferences{},
		&models.Opportunity{},
		&models.Notification{},
		&models.UserAction{},
		&models.Subscription{},
		&models.Payment{},
		&models.ArbitrageOpportunity{},
		&models.DeFiOpportunity{},
		// Premium Client models
		&models.ClientSession{},
		&models.ClientTrade{},
		&models.ClientStatistics{},
	)
}

func CloseDatabase(db *gorm.DB) error {
	sqlDB, _ := db.DB()
	return sqlDB.Close()
}
