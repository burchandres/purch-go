package config

import (
	"log/slog"
	"os"
	"strconv"

	"sync"

	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/v40/plaid"
)

var (
	cfg  *Config
	once sync.Once
)

const (
	DEBUG = iota
	INFO
	WARN
	ERROR
)

type Config struct {
	// General settings
	LogLevel string
	// Database settings
	PostgresUrl string
	// Encryption settings
	SecretKey           string
	EncryptionAlgorithm string
	// Plaid settings
	PlaidClientID     string
	PlaidSecret       string
	PlaidEnv          string
	PlaidProducts     string
	PlaidCountryCodes string
	PlaidLanguage     string
	PlaidRedirectUri  string
}

func (c *Config) GetPlaidCountryCodes() []plaid.CountryCode {
	return []plaid.CountryCode{plaid.COUNTRYCODE_US}
}

func (c *Config) GetPlaidProducts() []plaid.Products {
	return []plaid.Products{plaid.PRODUCTS_AUTH, plaid.PRODUCTS_TRANSACTIONS}
}

func GetConfig() *Config {
	once.Do(func() {
		cfg = loadConfig()
	})
	return cfg
}

func loadConfig() *Config {
	// should be developing against docker deployment
	// but also pull from .env file if it exists
	// don't panic if nothing exists stuff will just break
	err := godotenv.Load(
		"/run/secrets/env", 
		".env",
		"../.env",
		"../../.env",
	)
	if err != nil {
		slog.Error("error loading config files", "error", err.Error())
	}

	config := &Config{
		LogLevel:            getEnvVar("LOG_LEVEL", "DEBUG"),
		PostgresUrl:         getEnvVar("POSTGRES_URL", "postgres://postgres:password@postgres:5432/purch?sslmode=disable"),
		SecretKey:           getEnvVar("SECRET_KEY", ""),
		EncryptionAlgorithm: getEnvVar("ENCRYPTION_ALGORITHM", "HS256"),
		PlaidClientID:       getEnvVar("PLAID_CLIENT_ID", ""),
		PlaidSecret:         getEnvVar("PLAID_SECRET", ""),
		PlaidEnv:            getEnvVar("PLAID_ENV", "saandbox"),
		PlaidProducts:       getEnvVar("PLAID_PRODUCTS", "auth,transactions"),
		PlaidCountryCodes:   getEnvVar("PLAID_COUNTRY_CODES", "US"),
		PlaidLanguage:       getEnvVar("PLAID_LANGUAGE", "en"),
		PlaidRedirectUri:    getEnvVar("PLAID_REDIRECT_URI", "http://localhost:5173/dashboard"),
	}

	return config
}

func getEnvVar[T string | int | bool | float64](key string, defaultValue T) T {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	var result any
	var err error

	switch any(defaultValue).(type) {
	case string:
		result = value
	case int:
		result, err = strconv.Atoi(value)
	case bool:
		result, err = strconv.ParseBool(value)
	case float64:
		result, err = strconv.ParseFloat(value, 64)
	}

	if err != nil {
		slog.Error("Error parsing %s: %v", key, err)
		return defaultValue
	}

	return result.(T)
}
