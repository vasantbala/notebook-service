package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vasantbala/notebook-service/internal/cache"
	"github.com/vasantbala/notebook-service/internal/db"
	"github.com/vasantbala/notebook-service/internal/model"
)

const messagesCacheTTL = time.Hour

type conversationService struct {
	repo  db.ConversationRepository
	cache cache.ConversationCache
}

func NewConversationService(repo db.ConversationRepository, cache cache.ConversationCache) ConversationService {
	return &conversationService{repo: repo, cache: cache}
}

func (s *conversationService) ListConversations(ctx context.Context, notebookID, userID string) ([]model.Conversation, error) {
	convs, err := s.repo.ListConversations(ctx, notebookID, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	return convs, nil
}

func (s *conversationService) GetConversation(ctx context.Context, id, notebookID, userID string) (*model.Conversation, error) {
	conv, err := s.repo.GetConversation(ctx, id, notebookID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get conversation: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return conv, nil
}

func (s *conversationService) CreateConversation(ctx context.Context, notebookID, userID, title string) (model.Conversation, error) {
	conv, err := s.repo.CreateConversation(ctx, notebookID, userID, title)
	if err != nil {
		return model.Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	return conv, nil
}

func (s *conversationService) DeleteConversation(ctx context.Context, id, notebookID, userID string) error {
	if err := s.repo.DeleteConversation(ctx, id, notebookID, userID); err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	// Evict cached messages so stale data is not served after deletion.
	_ = s.cache.InvalidateMessages(ctx, id)
	return nil
}

func (s *conversationService) UpdateConversation(ctx context.Context, id, notebookID, userID string, patch db.ConversationPatch) (*model.Conversation, error) {
	conv, err := s.repo.UpdateConversation(ctx, id, notebookID, userID, patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("update conversation: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("update conversation: %w", err)
	}
	return conv, nil
}

func (s *conversationService) ListMessages(ctx context.Context, conversationID, userID string) ([]model.Message, error) {
	// Cache-first: warm path avoids a DB round-trip on every chat turn.
	if msgs, err := s.cache.GetMessages(ctx, conversationID); err == nil && msgs != nil {
		return msgs, nil
	}

	msgs, err := s.repo.ListMessages(ctx, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	// Populate cache for subsequent requests; ignore cache write errors.
	_ = s.cache.SetMessages(ctx, conversationID, msgs, messagesCacheTTL)
	return msgs, nil
}
func (s *conversationService) AddMessage(
	ctx context.Context,
	conversationID string,
	role model.Role,
	content string,
	tokenCount int,
	citations []model.Citation,
) (model.Message, error) {
	msg, err := s.repo.AddMessage(ctx, conversationID, role, content, tokenCount, citations)
	if err != nil {
		return model.Message{}, fmt.Errorf("add message: %w", err)
	}
	// Invalidate cache so next ListMessages hits DB with the new message.
	_ = s.cache.InvalidateMessages(ctx, conversationID)
	return msg, nil
}
