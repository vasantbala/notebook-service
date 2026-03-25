package config

import (
	"os"
)

type Config struct {
	Port         string
	DatabaseURL  string
	RedisURL     string
	JWKSEndpoint string
}

func Load() Config {
	return Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  mustGetEnv("DATABASE_URL"),
		RedisURL:     getEnv("REDIS_URL", ""),
		JWKSEndpoint: mustGetEnv("JWKS_ENDPOINT"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}
