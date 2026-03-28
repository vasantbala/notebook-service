package config

import (
	"log"
	"os"
)

type Config struct {
	Port               string
	DatabaseURL        string
	RedisURL           string
	JWKSEndpoint       string
	RAGAnythingBaseUrl string
	LLMProvider        string
	OpenAI             OpenAIConfig
}

type OpenAIConfig struct {
	BaseUrl string
	APIKey  string
	Model   string
}

func Load() Config {

	llm_provider := getEnv("LLM_PROVIDER", "openai")

	if llm_provider != "openai" {
		log.Panicf("Unsupported llm_provider: %s", llm_provider)
	}

	return Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        mustGetEnv("DATABASE_URL"),
		RedisURL:           getEnv("REDIS_URL", ""),
		JWKSEndpoint:       mustGetEnv("JWKS_ENDPOINT"),
		RAGAnythingBaseUrl: mustGetEnv("RAG_ANYTHING_BASE_URL"),
		LLMProvider:        mustGetEnv("LLM_PROVIDER"),
		OpenAI:             LoadOpenAIConfig(),
	}
}

func LoadOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		BaseUrl: getEnv("OPENAI__BASEURL", "https://api.openai.com/v1"),
		APIKey:  mustGetEnv("OPENAI__APIKEY"),
		Model:   mustGetEnv("OPENAI__MODEL"),
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
