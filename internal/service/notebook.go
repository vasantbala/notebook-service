package service

import (
	"context"

	"github.com/vasantbala/notebook-service/internal/model"
)

type NotebookService interface {
	ListNotebooks(ctx context.Context, userID string) ([]model.Notebook, error)
	GetNotebook(ctx context.Context, id, userID string) (*model.Notebook, error)
	CreateNotebook(ctx context.Context, userID, title, description string) (model.Notebook, error)
	DeleteNotebook(ctx context.Context, id, userID string) error
        UpdateNotebook(ctx context.Context, id, userID, title, description string) (*model.Notebook, error)
}
