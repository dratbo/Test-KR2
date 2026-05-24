package config

import (
	"os"
)

type Config struct {
	Port           string
	UserServiceURL string
	TaskServiceURL string
}

func Load() *Config {
	return &Config{
		Port:           getEnv("GATEWAY_PORT", "8080"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "http://localhost:8081"),
		TaskServiceURL: getEnv("TASK_SERVICE_URL", "http://localhost:8082"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
