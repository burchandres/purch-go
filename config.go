package main

import (
	"os"
	"strconv"
	"strings"
	"log/slog"
	
	"github.com/joho/godotenv"
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
}

func (c *Config) GetPlaidProducts() []string {
	return strings.Split(c.PlaidProducts, ",")
}

func loadConfig() (*Config, error) {
	// should be developing against docker deployment
	if err := godotenv.Load("/run/secrets/env"); err != nil {
		// if not check for a .env file in the current directory
		if err := godotenv.Load(".env"); err != nil {
			return nil, err
		}
	}

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
	}

	return config, nil
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
