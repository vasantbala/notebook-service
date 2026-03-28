package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vasantbala/notebook-service/internal/model"
)

type ConversationCache interface {
	GetMessages(ctx context.Context, conversationID string) ([]model.Message, error)
	SetMessages(ctx context.Context, conversationID string, msgs []model.Message, ttl time.Duration) error
	InvalidateMessages(ctx context.Context, conversationID string) error
}

type redisConversationCache struct{ client *redis.Client }

func NewRedisConversationCache(client *redis.Client) ConversationCache {
	return &redisConversationCache{client: client}
}

type redisJwtCache struct{ client *redis.Client }

func NewRedisJwtCache(client *redis.Client) *redisJwtCache {
	return &redisJwtCache{client: client}
}

func (c *redisConversationCache) GetMessages(ctx context.Context, convID string) ([]model.Message, error) {
	key := fmt.Sprintf("conv:%s:messages", convID)
	data, err := c.client.Get(ctx, key).Bytes()

	if err == redis.Nil {
		return nil, nil //cache miss - not an error
	}

	if err != nil {
		return nil, fmt.Errorf("cache get messages: %w", err)
	}

	var msgs []model.Message
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, fmt.Errorf("cache unmarshal messages: %w", err)
	}

	return msgs, nil
}

func (c *redisConversationCache) SetMessages(ctx context.Context, convID string, msgs []model.Message, ttl time.Duration) error {
	key := fmt.Sprintf("conv:%s:messages", convID)
	data, err := json.Marshal(msgs)

	if err != nil {
		return fmt.Errorf("cache marshal messages: %w", err)
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *redisConversationCache) InvalidateMessages(ctx context.Context, convID string) error {
	key := fmt.Sprintf("conv:%s:messages", convID)
	return c.client.Del(ctx, key).Err()
}

func (j *redisJwtCache) GetUserID(ctx context.Context, rawToken string) (string, bool, error) {
	key := fmt.Sprintf("jwt:%x", sha256.Sum256([]byte(rawToken)))
	userID, err := j.client.Get(ctx, key).Bytes()

	if err == redis.Nil {
		return "", false, nil //cache miss - not an error
	}

	if err != nil {
		return "", false, fmt.Errorf("cache get userId: %w", err)
	}

	return string(userID), true, nil
}

func (j *redisJwtCache) SetUserID(ctx context.Context, rawToken string, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("jwt:%x", sha256.Sum256([]byte(rawToken)))

	return j.client.Set(ctx, key, userID, ttl).Err()
}
