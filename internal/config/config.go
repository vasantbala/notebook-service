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
	ReasoningEffort    string // "low", "medium", "high" — passed to o1/o3 models
	OpenAI             OpenAIConfig
	Langfuse           LangfuseConfig
}

type OpenAIConfig struct {
	BaseUrl        string
	APIKey         string
	Model          string
	ReasoningModel string // e.g. "o3" — used when conversation.use_reasoning = true
}

type LangfuseConfig struct {
	PublicKey string
	SecretKey string
	Host      string
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
		ReasoningEffort:    getEnv("LLM_REASONING_EFFORT", "medium"),
		OpenAI:             LoadOpenAIConfig(),
		Langfuse: LangfuseConfig{
			PublicKey: getEnv("LANGFUSE_PUBLIC_KEY", ""),
			SecretKey: getEnv("LANGFUSE_SECRET_KEY", ""),
			Host:      getEnv("LANGFUSE_HOST", "https://cloud.langfuse.com"),
		},
	}
}

func LoadOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		BaseUrl:        getEnv("OPENAI__BASEURL", "https://api.openai.com/v1"),
		APIKey:         mustGetEnv("OPENAI__APIKEY"),
		Model:          mustGetEnv("OPENAI__MODEL"),
		ReasoningModel: getEnv("OPENAI__REASONING_MODEL", ""),
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
