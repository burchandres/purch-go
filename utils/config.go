package utils

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	// "strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/v40/plaid"
)

var (
	cfg *Config
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
	LogLevel     		string
	// Database settings
	PostgresHost        string
	PostgresPort        int
	PostgresUser        string
	PostgresPassword    string
	PostgresDatabase    string
	// Encryption settings
	SecretKey           string
	EncryptionAlgorithm string
	// Plaid settings
	PlaidClientID       string
	PlaidSecret         string
	PlaidEnv            string
	PlaidProducts       string
	PlaidCountryCodes   string
	PlaidLanguage       string
	PlaidRedirectUri    string
}

func (c *Config) GetPlaidCountryCodes() []plaid.CountryCode {
	// countryCodes := strings.Split(c.PlaidCountryCodes, ",")
	// var codes []plaid.CountryCode
	// for _, code := range countryCodes {
	// 	codes = append(codes, plaid.CountryCode(code))
	// }
	// return codes
	return []plaid.CountryCode{plaid.COUNTRYCODE_US}
}

func (c *Config) GetPlaidProducts() []plaid.Products {
	// plaidProducts := strings.Split(c.PlaidProducts, ",")
	// var products []plaid.Products
	// for _, product := range plaidProducts {
	// 	products = append(products, plaid.Products(product))
	// }
	// return products
	return []plaid.Products{plaid.PRODUCTS_AUTH, plaid.PRODUCTS_TRANSACTIONS}
}

func (c *Config) GetPostgresURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s", 
		c.PostgresUser, 
		c.PostgresPassword, 
		c.PostgresHost, 
		c.PostgresPort, 
		c.PostgresDatabase,
	)
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
	_ = godotenv.Load("/run/secrets/env")
	_ = godotenv.Load(".env")

	config := &Config{
		LogLevel:            getEnvVar("LOG_LEVEL", "INFO"),
		PostgresHost:        getEnvVar("POSTGRES_HOST", "postgres"),
		PostgresPort:        getEnvVar("POSTGRES_PORT", 5432),
		PostgresUser:        getEnvVar("POSTGRES_USER", "postgres"),
		PostgresPassword:    getEnvVar("POSTGRES_PASSWORD", "password"),
		PostgresDatabase:    getEnvVar("POSTGRES_DATABASE", "purch"),
		SecretKey:           getEnvVar("SECRET_KEY", ""),
		EncryptionAlgorithm: getEnvVar("ENCRYPTION_ALGORITHM", "HS256"),
		PlaidClientID:       getEnvVar("PLAID_CLIENT_ID", "client_id"),
		PlaidSecret:         getEnvVar("PLAID_SECRET", "secret"),
		PlaidEnv:            getEnvVar("PLAID_ENV", "sandbox"),
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
