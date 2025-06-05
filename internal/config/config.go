package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI      string
	DatabaseName  string
	JWTSecret     string
	JWTExpiration time.Duration
	ServerPort    string
	Environment   string
	StripeKey     string
	StripeWebhook string
}

var AppConfig Config

// Load loads the configuration from environment variables
func Load() error {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Set default values
	AppConfig = Config{
		MongoURI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DatabaseName:  getEnv("DB_NAME", "course-api"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpiration: time.Duration(getEnvAsInt("JWT_EXPIRATION_HOURS", 24)) * time.Hour,
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		StripeKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhook: getEnv("STRIPE_WEBHOOK_SECRET", ""),
	}

	return nil
}

// Helper function to get environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Helper function to get environment variable as integer with a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
