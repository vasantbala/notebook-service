package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/vasantbala/notebook-service/internal/api"
	"github.com/vasantbala/notebook-service/internal/cache"
	"github.com/vasantbala/notebook-service/internal/config"
	"github.com/vasantbala/notebook-service/internal/db"
	_ "github.com/vasantbala/notebook-service/internal/docs" // registers OpenAPI spec
	"github.com/vasantbala/notebook-service/internal/llm"
	"github.com/vasantbala/notebook-service/internal/service"
)

// @title          Notebook Service API
// @version        1.0
// @description    RAG-backed notebook and conversation management service.
//
// @host       localhost:8080
// @BasePath   /
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT issued by Authentik. Prefix with "Bearer ".
func main() {

	fmt.Println("notebook-service starting up...")

	_ = godotenv.Load()

	cfg := config.Load()

	log.Print("config loaded")

	runMigrations(cfg)
	log.Print("migrations ran")

	//postgres db
	pool, _ := pgxpool.New(context.Background(), cfg.DatabaseURL)
	//redis cache
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})

	jwks, err := keyfunc.NewDefault([]string{cfg.JWKSEndpoint})
	if err != nil {
		log.Fatalf("failed to fetch JWKS: %v", err)
	}

	//redis caches
	convCache := cache.NewRedisConversationCache(rdb)
	jwtCache := cache.NewRedisJwtCache(rdb)
	rateLimitCache := cache.NewRedisRateLimiter(rdb)

	//repos
	notebookRepo := db.NewPgNotebookRepo(pool)
	conversationRepo := db.NewPGConversationRepo(pool)
	sourceRepo := db.NewPGSourceRepo(pool)

	//clients
	ragClient := service.NewRAGAnythingClient(cfg.RAGAnythingBaseUrl)
	llmClient := llm.NewOpenAIClient(cfg.OpenAI.APIKey, cfg.OpenAI.Model, cfg.OpenAI.BaseUrl)

	h := &api.Handlers{
		Notebooks:     service.NewNotebookService(notebookRepo),
		Conversations: service.NewConversationService(conversationRepo, convCache),
		Sources:       service.NewSourceService(sourceRepo, cfg.RAGAnythingBaseUrl),
		Retrieval:     ragClient,
		LLM:           llmClient,
	}
	r := api.NewRouter(h, jwks, jwtCache, rateLimitCache)

	log.Printf("Starting server on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

func runMigrations(cfg config.Config) {
	m, err := migrate.New("file://migrations", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create migrator: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migration failed: %v", err)
	}
}
