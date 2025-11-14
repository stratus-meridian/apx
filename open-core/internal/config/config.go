package config

import (
	"os"
	"strconv"
)

// Config holds router configuration
type Config struct {
	// Server
	Port        int
	Environment string // dev, staging, production
	PublicURL   string // Public-facing URL for status/stream endpoints

	// GCP Project
	ProjectID string
	Region    string

	// Policy Store
	PolicyStoreType string // firestore, gcs, local
	PolicyBucketName string
	FirestoreCollection string

	// Pub/Sub
	PubSubTopic string
	PubSubProjectID string

	// Redis (for rate limiting)
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Observability
	OTELEndpoint string
	OTELInsecure bool
	LogLevel     string
	SampleRate   float64

	// Feature Flags
	EnableCanary bool
	EnableCache  bool
}

// Load configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnvAsInt("PORT", 8081),
		Environment: getEnv("ENVIRONMENT", "dev"),
		PublicURL:   getEnv("PUBLIC_URL", ""),

		ProjectID: getEnv("GCP_PROJECT_ID", ""),
		Region:    getEnv("GCP_REGION", "us-central1"),

		PolicyStoreType:     getEnv("POLICY_STORE_TYPE", "firestore"),
		PolicyBucketName:    getEnv("POLICY_BUCKET", "apx-policy-artifacts"),
		FirestoreCollection: getEnv("FIRESTORE_COLLECTION", "policies"),

		PubSubTopic:     getEnv("PUBSUB_TOPIC", "apx-requests"),
		PubSubProjectID: getEnv("PUBSUB_PROJECT_ID", ""),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		OTELEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		OTELInsecure: getEnvAsBool("OTEL_INSECURE", true),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		SampleRate:   getEnvAsFloat("LOG_SAMPLE_RATE", 0.01),

		EnableCanary: getEnvAsBool("ENABLE_CANARY", false),
		EnableCache:  getEnvAsBool("ENABLE_CACHE", true),
	}

	// Set default PubSub project to main project if not specified
	if cfg.PubSubProjectID == "" {
		cfg.PubSubProjectID = cfg.ProjectID
	}

	// Note: GCP_PROJECT_ID is optional in open-core edition
	// If not set, Pub/Sub features will be disabled (which is fine for demo/testing)

	return cfg, nil
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
