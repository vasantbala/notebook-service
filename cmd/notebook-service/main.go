package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/vasantbala/notebook-service/internal/api"
	"github.com/vasantbala/notebook-service/internal/cache"
	"github.com/vasantbala/notebook-service/internal/config"
	"github.com/vasantbala/notebook-service/internal/service"
)

func main() {

	fmt.Println("notebook-service starting up...")

	_ = godotenv.Load()

	cfg := config.Load()
	log.Print("config loaded")

	runMigrations(cfg)
	log.Print("migrations ran")

	jwks, err := keyfunc.NewDefault([]string{cfg.JWKSEndpoint})
	if err != nil {
		log.Fatalf("failed to fetch JWKS: %v", err)
	}

	//redis caches
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	//convCache := cache.NewRedisConversationCache(rdb)
	jwtCache := cache.NewRedisJwtCache(rdb)
	// rateLimitCache := cache.NewRedisRateLimiter(rdb)

	svc := service.NewInMemNotebookService()
	h := &api.Handlers{Notebooks: svc}
	r := api.NewRouter(h, jwks, jwtCache)

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
