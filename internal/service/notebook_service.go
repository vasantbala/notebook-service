package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vasantbala/notebook-service/internal/db"
	"github.com/vasantbala/notebook-service/internal/model"
)

type notebookService struct {
	repository db.NotebookRepository
}

func NewNotebookService(repo db.NotebookRepository) NotebookService {
	return &notebookService{
		repository: repo,
	}
}

func (s *notebookService) ListNotebooks(ctx context.Context, userID string) ([]model.Notebook, error) {

	notebooks, err := s.repository.List(ctx, userID)

	if err != nil {
		return nil, err
	}

	return notebooks, nil
}

func (s *notebookService) GetNotebook(ctx context.Context, id, userID string) (*model.Notebook, error) {

	notebook, err := s.repository.Get(ctx, id, userID)

	if err != nil {
		return nil, err
	}

	return notebook, nil
}

func (s *notebookService) CreateNotebook(ctx context.Context, userID, title, description string) (model.Notebook, error) {
	nb := model.Notebook{
		ID:          uuid.NewString(),
		UserID:      userID,
		Title:       title,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := s.repository.Create(ctx, nb)

	if err != nil {
		return model.Notebook{}, fmt.Errorf("create notebook: %w", err)
	}

	return nb, nil
}

func (s *notebookService) DeleteNotebook(ctx context.Context, id, userID string) error {

	err := s.repository.Delete(ctx, id, userID)

	if err != nil {
		return err
	}

	return nil
}
func (s *notebookService) UpdateNotebook(ctx context.Context, id, userID, title, description string) (*model.Notebook, error) {
	nb, err := s.repository.Update(ctx, id, userID, title, description)
	if err != nil {
		return nil, fmt.Errorf("update notebook: %w", err)
	}
	return nb, nil
}
