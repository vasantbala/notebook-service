package db

import (
	"context"

	"github.com/vasantbala/notebook-service/internal/model"
)

type ConversationRepository interface {
	ListConversations(ctx context.Context, notebookID, userID string) ([]model.Conversation, error)
	GetConversation(ctx context.Context, id, notebookID, userID string) (*model.Conversation, error)
	CreateConversation(ctx context.Context, notebookID, userID, title string) (model.Conversation, error)
	DeleteConversation(ctx context.Context, id, notebookID, userID string) error
	ListMessages(ctx context.Context, conversationID, userID string) ([]model.Message, error)
	AddMessage(ctx context.Context, conversationID string, role model.Role, content string, tokenCount int, citations []model.Citation) (model.Message, error)
}
