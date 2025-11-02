package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `yaml:"app" mapstructure:"app"`
	Telegram TelegramConfig `yaml:"telegram" mapstructure:"telegram"`
	Database DatabaseConfig `yaml:"database" mapstructure:"database"`
	Redis    RedisConfig    `yaml:"redis" mapstructure:"redis"`
	Payment  PaymentConfig  `yaml:"payment" mapstructure:"payment"`
}

type AppConfig struct {
	Environment string `yaml:"environment" mapstructure:"environment"`
	Port        string `yaml:"port" mapstructure:"port"`
	LogLevel    string `yaml:"log_level" mapstructure:"log_level"`
}

type TelegramConfig struct {
	BotToken   string `yaml:"bot_token" mapstructure:"bot_token"`
	WebhookURL string `yaml:"webhook_url" mapstructure:"webhook_url"`
	Debug      bool   `yaml:"debug" mapstructure:"debug"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     string `yaml:"port" mapstructure:"port"`
	User     string `yaml:"user" mapstructure:"user"`
	Password string `yaml:"password" mapstructure:"password"`
	DBName   string `yaml:"db_name" mapstructure:"db_name"`
	SSLMode  string `yaml:"ssl_mode" mapstructure:"ssl_mode"`
	MaxConns int    `yaml:"max_conns" mapstructure:"max_conns"`
}

type RedisConfig struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     string `yaml:"port" mapstructure:"port"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
}

type PaymentConfig struct {
	StripeSecretKey      string `yaml:"stripe_secret_key" mapstructure:"stripe_secret_key"`
	StripePublishableKey string `yaml:"stripe_publishable_key" mapstructure:"stripe_publishable_key"`
	StripeWebhookSecret  string `yaml:"stripe_webhook_secret" mapstructure:"stripe_webhook_secret"`
}

func LoadConfig(configPath string) (*Config, error) {
	_ = godotenv.Load()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	config.Telegram.BotToken = getEnv("TELEGRAM_BOT_TOKEN", config.Telegram.BotToken)
	config.Database.User = getEnv("DB_USER", config.Database.User)
	config.Database.Password = getEnv("DB_PASSWORD", config.Database.Password)
	config.Database.DBName = getEnv("DB_NAME", config.Database.DBName)
	config.Redis.Password = getEnv("REDIS_PASSWORD", config.Redis.Password)
	config.Payment.StripePublishableKey = getEnv("STRIPE_PUBLISHABLE_KEY", config.Payment.StripePublishableKey)
	config.Payment.StripeWebhookSecret = getEnv("STRIPE_WEBHOOK_SECRET", config.Payment.StripeWebhookSecret)
	config.Payment.StripeSecretKey = getEnv("STRIPE_SECRET_KEY", config.Payment.StripeSecretKey)

	err := config.Validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("telegram.bot_token is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}

	if c.Database.Port == "" {
		return fmt.Errorf("database.port is required")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("database.db_name is required")
	}

	if c.Database.Password == "" {
		return fmt.Errorf("database.password is required")
	}

	if c.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}

	if c.Redis.Port == "" {
		return fmt.Errorf("redis.port is required")
	}

	if c.App.Environment == "production" {
		if c.Telegram.WebhookURL == "" {
			return fmt.Errorf("telegram.webhook_url is required")
		}

		if c.Payment.StripeSecretKey == "" {
			return fmt.Errorf("payment.stripe_secret_key is required")
		}

		if c.Payment.StripePublishableKey == "" {
			return fmt.Errorf("payment.stripe_publishable_key is required")
		}

		if c.Payment.StripeWebhookSecret == "" {
			return fmt.Errorf("payment.stripe_webhook_secret is required")
		}
	}

	return nil
}

func (c *Config) SafeString() string {
	return fmt.Sprintf(`Config:
		Environment: %s
		Port: %s
		Log Level: %s

		Telegram:
			Bot Token: %s
			Webhook: %s
			Debug: %t
		  
		Database:
			Host: %s:%s
			User: %s
			Database: %s
			SSL Mode: %s
			Max Connections: %d
		  
		Redis:
			Host: %s:%s
			Database: %d
		  
		Payment (Stripe):
			Keys Configured: %t
		`,
		c.App.Environment,
		c.App.Port,
		c.App.LogLevel,
		maskSecret(c.Telegram.BotToken),
		c.Telegram.WebhookURL,
		c.Telegram.Debug,
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.DBName,
		c.Database.SSLMode,
		c.Database.MaxConns,
		c.Redis.Host,
		c.Redis.Port,
		c.Redis.DB,
		c.Payment.StripeSecretKey != "",
	)
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}

	length := len(s)
	if length <= 8 {
		return s + strings.Repeat("*", length-8)
	}

	return s[:4] + "..." + s[length-4:]
}
