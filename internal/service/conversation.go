package service

import (
    "context"
    "github.com/vasantbala/notebook-service/internal/model"
)

type ConversationService interface {
    ListConversations(ctx context.Context, notebookID, userID string) ([]model.Conversation, error)
    GetConversation(ctx context.Context, id, notebookID, userID string) (*model.Conversation, error)
    CreateConversation(ctx context.Context, notebookID, userID, title string) (model.Conversation, error)
    DeleteConversation(ctx context.Context, id, notebookID, userID string) error
    // ListMessages loads from cache first, then DB.
    ListMessages(ctx context.Context, conversationID, userID string) ([]model.Message, error)
}