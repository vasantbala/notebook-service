package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "userID"

type jwtCache interface {
	GetUserID(ctx context.Context, rawToken string) (string, bool, error)
	SetUserID(ctx context.Context, rawToken string, userID string, ttl time.Duration) error
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// AuthMiddleware validates a Bearer JWT on every request using the provided
// JWKS keyfunc. MicahParks/keyfunc refreshes the JWKS keys automatically in
// the background, so no additional Redis key caching is needed here.
// TODO: add Redis claim caching (Phase 11c) to skip re-parse for warm tokens.
func AuthMiddleware(jwks keyfunc.Keyfunc, jc jwtCache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")

			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			bearerToken := strings.TrimPrefix(token, "Bearer ")

			if userID, ok, _ := jc.GetUserID(r.Context(), bearerToken); ok {
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			userID, err := validateToken(bearerToken, jwks)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			_ = jc.SetUserID(r.Context(), bearerToken, userID, 15*time.Minute)

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func validateToken(bearerToken string, jwks keyfunc.Keyfunc) (string, error) {
	token, err := jwt.Parse(bearerToken, jwks.Keyfunc)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("missing sub claim")
	}

	return sub, nil
}
