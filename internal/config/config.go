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
	App       AppConfig       `yaml:"app" mapstructure:"app"`
	Telegram  TelegramConfig  `yaml:"telegram" mapstructure:"telegram"`
	Database  DatabaseConfig  `yaml:"database" mapstructure:"database"`
	Redis     RedisConfig     `yaml:"redis" mapstructure:"redis"`
	Payment   PaymentConfig   `yaml:"payment" mapstructure:"payment"`
	Arbitrage ArbitrageConfig `yaml:"arbitrage" mapstructure:"arbitrage"`
	DeFi      DeFiConfig      `yaml:"defi" mapstructure:"defi"`
	Admin     AdminConfig     `yaml:"admin" mapstructure:"admin"`
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
	// Stripe (deprecated, використовуємо Monobank)
	StripeSecretKey      string `yaml:"stripe_secret_key" mapstructure:"stripe_secret_key"`
	StripePublishableKey string `yaml:"stripe_publishable_key" mapstructure:"stripe_publishable_key"`
	StripeWebhookSecret  string `yaml:"stripe_webhook_secret" mapstructure:"stripe_webhook_secret"`

	// Monobank
	MonobankToken     string `yaml:"monobank_token" mapstructure:"monobank_token"`
	MonobankPublicKey string `yaml:"monobank_public_key" mapstructure:"monobank_public_key"`
	WebhookURL        string `yaml:"webhook_url" mapstructure:"webhook_url"`
	RedirectURL       string `yaml:"redirect_url" mapstructure:"redirect_url"`
	WebhookPort       string `yaml:"webhook_port" mapstructure:"webhook_port"`
}

type ArbitrageConfig struct {
	Enabled          bool     `yaml:"enabled" mapstructure:"enabled"`
	Pairs            []string `yaml:"pairs" mapstructure:"pairs"`
	Exchanges        []string `yaml:"exchanges" mapstructure:"exchanges"`
	MinProfitPercent float64  `yaml:"min_profit_percent" mapstructure:"min_profit_percent"`
	MinVolume24h     float64  `yaml:"min_volume_24h" mapstructure:"min_volume_24h"`
	MaxSpreadPercent float64  `yaml:"max_spread_percent" mapstructure:"max_spread_percent"`
	MaxSlippage      float64  `yaml:"max_slippage" mapstructure:"max_slippage"`
	Amount           float64  `yaml:"amount" mapstructure:"amount"`
	DeduplicateTTL   int      `yaml:"deduplicate_ttl" mapstructure:"deduplicate_ttl"` // minutes
}

type DeFiConfig struct {
	Enabled        bool     `yaml:"enabled" mapstructure:"enabled"`
	Chains         []string `yaml:"chains" mapstructure:"chains"`
	Protocols      []string `yaml:"protocols" mapstructure:"protocols"`
	MinAPY         float64  `yaml:"min_apy" mapstructure:"min_apy"`
	MinTVL         float64  `yaml:"min_tvl" mapstructure:"min_tvl"`
	MaxILRisk      float64  `yaml:"max_il_risk" mapstructure:"max_il_risk"`
	MinVolume24h   float64  `yaml:"min_volume_24h" mapstructure:"min_volume_24h"`
	ScrapeInterval int      `yaml:"scrape_interval" mapstructure:"scrape_interval"` // minutes
}

type AdminConfig struct {
	Enabled        bool     `yaml:"enabled" mapstructure:"enabled"`
	Host           string   `yaml:"host" mapstructure:"host"`
	Port           int      `yaml:"port" mapstructure:"port"`
	JWTSecret      string   `yaml:"jwt_secret" mapstructure:"jwt_secret"`
	AllowedOrigins []string `yaml:"allowed_origins" mapstructure:"allowed_origins"`
	RateLimit      int      `yaml:"rate_limit" mapstructure:"rate_limit"` // requests per minute
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
	config.Database.Port = getEnv("DB_PORT", config.Database.Port)
	config.Redis.Password = getEnv("REDIS_PASSWORD", config.Redis.Password)

	// Stripe (deprecated)
	config.Payment.StripePublishableKey = getEnv("STRIPE_PUBLISHABLE_KEY", config.Payment.StripePublishableKey)
	config.Payment.StripeWebhookSecret = getEnv("STRIPE_WEBHOOK_SECRET", config.Payment.StripeWebhookSecret)
	config.Payment.StripeSecretKey = getEnv("STRIPE_SECRET_KEY", config.Payment.StripeSecretKey)

	// Monobank
	config.Payment.MonobankToken = getEnv("MONOBANK_TOKEN", config.Payment.MonobankToken)
	config.Payment.MonobankPublicKey = getEnv("MONOBANK_PUBLIC_KEY", config.Payment.MonobankPublicKey)
	config.Payment.WebhookURL = getEnv("PAYMENT_WEBHOOK_URL", config.Payment.WebhookURL)
	config.Payment.RedirectURL = getEnv("PAYMENT_REDIRECT_URL", config.Payment.RedirectURL)
	config.Payment.WebhookPort = getEnv("PAYMENT_WEBHOOK_PORT", config.Payment.WebhookPort)

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

	// Redis обов'язковий лише для production
	// Для development він опціональний (деякі функції не працюватимуть без нього)
	if c.App.Environment == "production" {
		if c.Redis.Host == "" {
			return fmt.Errorf("redis.host is required for production")
		}

		if c.Redis.Port == "" {
			return fmt.Errorf("redis.port is required for production")
		}
	}

	if c.App.Environment == "production" {
		if c.Telegram.WebhookURL == "" {
			return fmt.Errorf("telegram.webhook_url is required")
		}

		// Monobank обов'язковий для production
		if c.Payment.MonobankToken == "" {
			return fmt.Errorf("payment.monobank_token is required for production")
		}

		if c.Payment.WebhookURL == "" {
			return fmt.Errorf("payment.webhook_url is required for production")
		}
	}

	// Для development теж потрібен Monobank token якщо хочемо тестувати
	if c.App.Environment == "development" && c.Payment.MonobankToken != "" {
		if c.Payment.WebhookPort == "" {
			c.Payment.WebhookPort = "8081" // Default webhook port
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

		Payment (Monobank):
			Token: %s
			Webhook URL: %s
			Webhook Port: %s
			Redirect URL: %s

		Arbitrage:
			Enabled: %t
			Pairs: %v
			Exchanges: %v
			Min Profit: %.2f%%
			Min Volume: $%.0f
			Max Spread: %.2f%%
			Max Slippage: %.2f%%
			Deduplicate TTL: %d min

		DeFi:
			Enabled: %t
			Chains: %v
			Protocols: %v
			Min APY: %.2f%%
			Min TVL: $%.0f
			Max IL Risk: %.2f%%
			Min Volume: $%.0f
			Scrape Interval: %d min
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
		maskSecret(c.Payment.MonobankToken),
		c.Payment.WebhookURL,
		c.Payment.WebhookPort,
		c.Payment.RedirectURL,
		c.Arbitrage.Enabled,
		c.Arbitrage.Pairs,
		c.Arbitrage.Exchanges,
		c.Arbitrage.MinProfitPercent,
		c.Arbitrage.MinVolume24h,
		c.Arbitrage.MaxSpreadPercent,
		c.Arbitrage.MaxSlippage,
		c.Arbitrage.DeduplicateTTL,
		c.DeFi.Enabled,
		c.DeFi.Chains,
		c.DeFi.Protocols,
		c.DeFi.MinAPY,
		c.DeFi.MinTVL,
		c.DeFi.MaxILRisk,
		c.DeFi.MinVolume24h,
		c.DeFi.ScrapeInterval,
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
