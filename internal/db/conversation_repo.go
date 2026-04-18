package db

import (
	"context"

	"github.com/vasantbala/notebook-service/internal/model"
)

type ConversationRepository interface {
	ListConversations(ctx context.Context, notebookID, userID string) ([]model.Conversation, error)
	GetConversation(ctx context.Context, id, notebookID, userID string) (*model.Conversation, error)
	CreateConversation(ctx context.Context, notebookID, userID, title string) (model.Conversation, error)
	UpdateConversation(ctx context.Context, id, notebookID, userID string, patch ConversationPatch) (*model.Conversation, error)
	DeleteConversation(ctx context.Context, id, notebookID, userID string) error
	ListMessages(ctx context.Context, conversationID, userID string) ([]model.Message, error)
	AddMessage(ctx context.Context, conversationID string, role model.Role, content string, tokenCount int, citations []model.Citation) (model.Message, error)
}

// ConversationPatch carries optional fields for a partial update.
// Nil pointer = leave the field unchanged.
type ConversationPatch struct {
	Title        *string
	RAGEnabled   *bool
	UseReasoning *bool
	Model        *string // set to pointer-to-empty-string to clear the override
}
