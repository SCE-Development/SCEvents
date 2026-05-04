package config

import (
	"os"
	"strconv"
	"time"
)

type AppConfig struct {
	MongoURI     string
	RedisAddr    string
	KafkaBroker  string
	KafkaTopic   string
	KafkaGroupID string
	ServerPort   string
	ClientURL    string
	ClientAPIURL string
	AutoMigrateEnabled  bool
	AutoMigrateInterval time.Duration
}

func Load() AppConfig {
	return AppConfig{
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBroker:  getEnv("KAFKA_BROKER", "localhost:9092"),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "registrations"),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "registration-consumers"),
		ServerPort:   getEnv("SERVER_PORT", "8002"),
		ClientURL:    getEnv("CLIENT_URL", "http://localhost:3000"),
		ClientAPIURL: getEnv("CLIENT_API_URL", "http://localhost:8080"),
		AutoMigrateEnabled:  getEnvBool("AUTO_MIGRATE_ENABLED", true),
		AutoMigrateInterval: getEnvDurationHours("AUTO_MIGRATE_INTERVAL_HOURS", 24),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDurationHours(key string, fallbackHours int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(fallbackHours) * time.Hour
	}

	hours, err := strconv.Atoi(value)
	if err != nil || hours <= 0 {
		return time.Duration(fallbackHours) * time.Hour
	}
	return time.Duration(hours) * time.Hour
}
