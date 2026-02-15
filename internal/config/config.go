package config

import (
	"log/slog"
	"os"
	"strings"

	"sync"

	"github.com/plaid/plaid-go/v40/plaid"
	"github.com/spf13/viper"
)

var (
	slogLevels = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	GetCachedConfig = sync.OnceValue(loadConfig)
)

type Config struct {
	// General settings
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	ApiPort     int    `mapstructure:"API_PORT"`
	WebhookPort int    `mapstructure:"WEBHOOK_PORT"`
	GinMode     string `mapstructure:"GIN_MODE"`
	// Database settings
	PostgresUrl string `mapstructure:"POSTGRES_URL"`
	// Encryption settings
	SecretKey           string `mapstructure:"SECRET_KEY"`
	EncryptionAlgorithm string `mapstructure:"ENCRYPTION_ALGORITHM"`
	BcryptCost          int    `mapstructure:"BCRYPT_COST"`
	// Plaid settings
	PlaidClientID     string `mapstructure:"PLAID_CLIENT_ID"`
	PlaidSecret       string `mapstructure:"PLAID_SECRET"`
	PlaidEnv          string `mapstructure:"PLAID_ENV"`
	PlaidProducts     string `mapstructure:"PLAID_PRODUCTS"`
	PlaidCountryCodes string `mapstructure:"PLAID_COUNTRY_CODES"`
	PlaidLanguage     string `mapstructure:"PLAID_LANGUAGE"`
	PlaidRedirectUri  string `mapstructure:"PLAID_REDIRECT_URI"`
	WebhookUrl        string `mapstructure:"WEBHOOK_URL"`
}

func (c *Config) GetPlaidCountryCodes() []plaid.CountryCode {
	return []plaid.CountryCode{plaid.COUNTRYCODE_US}
}

func (c *Config) GetPlaidProducts() []plaid.Products {
	return []plaid.Products{plaid.PRODUCTS_AUTH, plaid.PRODUCTS_TRANSACTIONS}
}

// Set sensible defaults except for plaid secret and client id
func setConfigDefaults() {
	viper.SetDefault("LOG_LEVEL", "DEBUG")
	viper.SetDefault("API_PORT", 8080)
	viper.SetDefault("WEBHOOK_PORT", 8081)
	viper.SetDefault("GIN_MODE", "debug")
	viper.SetDefault("POSTGRES_URL", "postgres://postgres:password@postgres:5432/purch?sslmode=disable")
	viper.SetDefault("BCRYPT_COST", 15)
	viper.SetDefault("PLAID_ENV", "Sandbox")
	viper.SetDefault("PLAID_PRODUCTS", "auth,transactions")
	viper.SetDefault("PLAID_COUNTRY_CODES", "US")
	viper.SetDefault("PLAID_LANGUAGE", "en")
	viper.SetDefault("PLAID_REDIRECT_URI", "http://localhost:5173/dashboard")
	viper.SetDefault("WEBHOOK_URL", "")
}

func loadConfig() Config {
	setConfigDefaults()
	// should be developing against docker deployment
	viper.AddConfigPath("/run/secrets")
	viper.SetConfigName("env")
	viper.SetConfigType("env")

	viper.MustBindEnv("PLAID_CLIENT_ID")
	viper.MustBindEnv("PLAID_SECRET")
	viper.MustBindEnv("SECRET_KEY")
	
	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("could not read config file", "error", err.Error())
	} else {
		slog.Info("successfully loaded config")
	}
	
	viper.AutomaticEnv()

	var config Config

	if err := viper.Unmarshal(&config); err != nil {
		panic(err)
	}

	configureLogging(config.LogLevel)
	slog.Info("log level set", "level", config.LogLevel)
	slog.Info("gin mode set", "mode", config.GinMode)
	slog.Info("webhook url set", "url", config.WebhookUrl)
	slog.Debug("viper settings", "settings", viper.AllSettings())
	slog.Debug("config values", "config", config)

	return config
}

func configureLogging(logLevel string) {
	logger := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slogLevels[strings.ToLower(logLevel)],
		},
	),
	)
	slog.SetDefault(logger)
}
