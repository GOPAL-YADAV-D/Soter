package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// Server
	Port string
	Host string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Azure Blob Storage
	AzureStorageAccount   string
	AzureStorageKey       string
	AzureStorageContainer string
	AzureStorageEndpoint  string

	// JWT
	JWTSecret string

	// Rate Limiting
	RateLimitRPS      int
	RateLimitBurst    int
	StorageQuotaMB    int

	// Logging
	LogLevel string
}

func LoadConfig() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logrus.Debug("No .env file found, using environment variables")
	}

	config := &Config{
		// Server defaults
		Port: getEnv("PORT", "8080"),
		Host: getEnv("HOST", "0.0.0.0"),

		// Database defaults
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "soter"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		// Azure Storage defaults (Azurite for local development)
		AzureStorageAccount:   getEnv("AZURE_STORAGE_ACCOUNT", "devstoreaccount1"),
		AzureStorageKey:       getEnv("AZURE_STORAGE_KEY", "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="),
		AzureStorageContainer: getEnv("AZURE_STORAGE_CONTAINER", "files"),
		AzureStorageEndpoint:  getEnv("AZURE_STORAGE_ENDPOINT", "http://localhost:10000/devstoreaccount1"),

		// Security defaults
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),

		// Rate limiting defaults
		RateLimitRPS:   getEnvAsInt("RATE_LIMIT_RPS", 2),
		RateLimitBurst: getEnvAsInt("RATE_LIMIT_BURST", 5),
		StorageQuotaMB: getEnvAsInt("STORAGE_QUOTA_MB", 10),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}