package configs

import (
	"os"
	"strconv"
)

type Config struct {
	AppName        string
	AppEnv         string
	HTTPPort       int
	JWTSecret      string
	PasswordPepper string
}

func Load() Config {
	return Config{
		AppName:        getEnv("APP_NAME", "gobank"),
		AppEnv:         getEnv("APP_ENV", "development"),
		HTTPPort:       getEnvAsInt("HTTP_PORT", 8080),
		JWTSecret:      getEnv("JWT_SECRET", "super-secret-key"),
		PasswordPepper: getEnv("PASSWORD_PEPPER", "change-me"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return number
}
