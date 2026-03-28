package db

import (
	"context"

	"github.com/vasantbala/notebook-service/internal/model"
)

type NotebookRepository interface {
	List(ctx context.Context, userID string) ([]model.Notebook, error)
	Get(ctx context.Context, id, userID string) (*model.Notebook, error)
	Create(ctx context.Context, nb model.Notebook) error
	Delete(ctx context.Context, id, userID string) error
	Update(ctx context.Context, id, userID, title, description string) (*model.Notebook, error)
}
