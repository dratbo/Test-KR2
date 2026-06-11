package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL   string
	RedisURL      string
	RedisCacheTTL int
	RabbitMQURL   string
}

func Load() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://dratbo:P@ssw0rd@localhost:5432/satisfactory?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", ""),
		RedisCacheTTL: getEnvAsInt("REDIS_CACHE_TTL", 60),
		RabbitMQURL:   getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
