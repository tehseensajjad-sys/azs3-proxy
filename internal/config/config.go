package config

import (
	"fmt"
	"os"
)

// Config holds all proxy configuration
type Config struct {
	// HTTP Server
	ListenAddr string

	// Azure Storage
	AzureStorageAccount string
	AzureStorageKey     string

	// S3 Auth
	S3AccessKeyID     string
	S3SecretAccessKey string

	// Logging
	LogLevel string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ListenAddr:          getEnv("LISTEN_ADDR", ":8080"),
		AzureStorageAccount: getEnvRequired("AZURE_STORAGE_ACCOUNT"),
		AzureStorageKey:     getEnvRequired("AZURE_STORAGE_KEY"),
		S3AccessKeyID:       getEnvRequired("S3_ACCESS_KEY"),
		S3SecretAccessKey:   getEnvRequired("S3_SECRET_KEY"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.AzureStorageAccount == "" {
		return fmt.Errorf("AZURE_STORAGE_ACCOUNT is required")
	}
	if c.AzureStorageKey == "" {
		return fmt.Errorf("AZURE_STORAGE_KEY is required")
	}
	if c.S3AccessKeyID == "" {
		return fmt.Errorf("S3_ACCESS_KEY is required")
	}
	if c.S3SecretAccessKey == "" {
		return fmt.Errorf("S3_SECRET_KEY is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return ""
}
