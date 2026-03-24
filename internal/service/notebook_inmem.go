package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vasantbala/notebook-service/internal/model"
)

type inMemNotebookService struct {
	mu        sync.RWMutex
	notebooks map[string]*model.Notebook
}

func NewInMemNotebookService() NotebookService {
	return &inMemNotebookService{
		notebooks: make(map[string]*model.Notebook),
	}
}

func (s *inMemNotebookService) ListNotebooks(ctx context.Context, userID string) ([]model.Notebook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.Notebook

	for _, nb := range s.notebooks {
		if nb.UserID == userID {
			result = append(result, *nb)
		}
	}

	return result, nil
}

func (s *inMemNotebookService) CreateNotebook(ctx context.Context, userID, title, description string) (model.Notebook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newNotebook := model.Notebook{
		ID:          uuid.NewString(),
		UserID:      userID,
		Title:       title,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	s.notebooks[newNotebook.ID] = &newNotebook

	return newNotebook, nil
}

func (s *inMemNotebookService) GetNotebook(ctx context.Context, id, userID string) (*model.Notebook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notebook, ok := s.notebooks[id]
	if !ok || notebook.UserID != userID {
		return nil, errors.New("Either you do not have access or the id is not found.")
	}
	return notebook, nil
}

func (s *inMemNotebookService) DeleteNotebook(ctx context.Context, id, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	notebook, ok := s.notebooks[id]
	if !ok || notebook.UserID != userID {
		return errors.New("Either you do not have access or the id is not found.")
	}

	delete(s.notebooks, id)
	return nil
}
