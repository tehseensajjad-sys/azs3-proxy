package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all proxy configuration loaded from environment variables.
// It includes settings for HTTP server, TLS, Azure authentication, S3 authentication, logging, and caching.
type Config struct {
	// HTTP Server configuration
	ListenAddr string // Address and port to listen on (default: :8080)

	// HTTPS/TLS Configuration
	EnableTLS bool   // Whether to enable HTTPS
	CertFile  string // Path to TLS certificate file
	KeyFile   string // Path to TLS private key file

	// Azure Storage Backend Type (blob or file)
	AzureBackendType string // Backend type: "blob" or "file" (default: "blob")

	// Azure Storage Authentication (supports multiple auth modes: account key, SAS, MSI, SPN, federated, Azure CLI)
	AzureAuth *AzureAuthConfig

	// S3 Authentication (SigV4 signature verification)
	S3AccessKeyID     string // S3 access key for authentication
	S3SecretAccessKey string // S3 secret key for authentication

	// Logging configuration
	LogLevel string // Log level: debug, info, warn, error, crit
	LogFile  string // Path to log file (empty string means console only)
	LogMode  string // Logging mode: console, file, or both

	// Local caching configuration
	CacheEnabled bool   // Whether to enable local caching of objects
	CachePath    string // Directory path for cached objects
	CacheMaxSize int64  // Maximum cache size in bytes (e.g., 1GB = 1073741824)
	CacheTTL     int    // Time-to-live for cache entries in seconds (default: 3600 = 1 hour)
}

// LoadConfig loads all proxy configuration from environment variables.
// It loads Azure auth config, S3 credentials, logging, and cache settings,
// then validates all required fields are present.
func LoadConfig() (*Config, error) {
	// Determine backend type early so we can build the correct endpoint default
	azureBackendType := getEnv("AZURE_BACKEND_TYPE", "blob")

	// Load and parse Azure authentication configuration from environment
	azureAuth, err := LoadAzureAuthConfigWithBackend(azureBackendType)
	if err != nil {
		return nil, fmt.Errorf("failed to load Azure authentication config: %w", err)
	}

	// Parse cache max size from environment (default: 1GB)
	cacheMaxSize := int64(1073741824) // 1 GB default
	if cacheSizeStr := os.Getenv("CACHE_MAX_SIZE"); cacheSizeStr != "" {
		if size, err := strconv.ParseInt(cacheSizeStr, 10, 64); err == nil {
			cacheMaxSize = size
		}
	}

	// Parse cache TTL from environment (default: 3600 seconds = 1 hour)
	cacheTTL := 3600
	if cacheTTLStr := os.Getenv("CACHE_TTL"); cacheTTLStr != "" {
		if ttl, err := strconv.Atoi(cacheTTLStr); err == nil && ttl > 0 {
			cacheTTL = ttl
		}
	}

	// Build configuration from environment variables with defaults
	cfg := &Config{
		ListenAddr:        getEnv("LISTEN_ADDR", ":8080"),
		EnableTLS:         getEnv("ENABLE_TLS", "false") == "true",
		CertFile:          getEnv("TLS_CERT_FILE", ""),
		KeyFile:           getEnv("TLS_KEY_FILE", ""),
		AzureBackendType:  azureBackendType,
		AzureAuth:         azureAuth,
		S3AccessKeyID:     getEnvRequired("S3_ACCESS_KEY"),
		S3SecretAccessKey: getEnvRequired("S3_SECRET_KEY"),
		LogLevel:          getEnv("LOG_LEVEL", "warn"),
		LogFile:           getEnv("LOG_FILE", ""),
		LogMode:           getEnv("LOG_MODE", "console"),
		CacheEnabled:      getEnv("CACHE_ENABLED", "false") == "true",
		CachePath:         getEnv("CACHE_PATH", "/tmp/azs3-proxy-cache"),
		CacheMaxSize:      cacheMaxSize,
		CacheTTL:          cacheTTL,
	}

	// Validate all required configuration is present and valid
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the entire configuration to ensure all required fields are set
// and configured properly. Returns an error if validation fails.
func (c *Config) Validate() error {
	// Check Azure authentication is configured
	if c.AzureAuth == nil {
		return fmt.Errorf("azure authentication config is required")
	}

	// Check S3 authentication credentials are present
	if c.S3AccessKeyID == "" {
		return fmt.Errorf("S3_ACCESS_KEY is required")
	}
	if c.S3SecretAccessKey == "" {
		return fmt.Errorf("S3_SECRET_KEY is required")
	}

	// Validate backend type
	if c.AzureBackendType != "blob" && c.AzureBackendType != "file" {
		return fmt.Errorf("AZURE_BACKEND_TYPE must be either 'blob' or 'file', got: %s", c.AzureBackendType)
	}

	// Validate TLS configuration if enabled
	if c.EnableTLS {
		if c.CertFile == "" || c.KeyFile == "" {
			return fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE are required when ENABLE_TLS is true")
		}
	}

	// Validate cache configuration if enabled
	if c.CacheEnabled {
		if c.CachePath == "" {
			return fmt.Errorf("CACHE_PATH is required when CACHE_ENABLED is true")
		}
		if c.CacheMaxSize <= 0 {
			return fmt.Errorf("CACHE_MAX_SIZE must be positive when caching is enabled")
		}
		if c.CacheTTL <= 0 {
			return fmt.Errorf("CACHE_TTL must be positive when caching is enabled")
		}
	}

	return nil
}

// getEnv retrieves an environment variable value with a default fallback.
// If the variable is not set, returns defaultValue.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvRequired retrieves a required environment variable.
// Returns empty string if not set (validation should catch this).
func getEnvRequired(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value

	}
	return ""
}
