package config

import "os"

type AppConfig struct {
	MongoURI     string
	RedisAddr    string
	KafkaBroker  string
	KafkaTopic   string
	KafkaGroupID string
	ServerPort   string
	ClientURL    string
	ClientAPIURL string
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
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
