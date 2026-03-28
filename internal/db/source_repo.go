package db

import (
	"context"

	"github.com/vasantbala/notebook-service/internal/model"
)

type SourceRepository interface {
	ListSources(ctx context.Context, notebookID, userID string) ([]model.Source, error)
	GetSource(ctx context.Context, id, notebookID, userID string) (*model.Source, error)
	// CreateSource inserts a new source record with status=pending.
	// File upload and rag-anything ingestion are handled by the service layer.
	CreateSource(ctx context.Context, notebookID, userID, filename, storageKey, mimeType string) (model.Source, error)
	// UpdateStatus sets the processing status and chunk count once ingestion completes.
	UpdateStatus(ctx context.Context, id string, status model.SourceStatus, chunkCount int) error
	// UpdateRagDocID stores the doc_id returned by rag-anything after successful ingestion.
	UpdateRagDocID(ctx context.Context, id, ragDocID string) error
	// ListRagDocIDs returns the rag_doc_id values for all ready sources in a notebook.
	// Used to build the doc_ids filter when calling rag-anything's /retrieve endpoint.
	ListRagDocIDs(ctx context.Context, notebookID, userID string) ([]string, error)
	DeleteSource(ctx context.Context, id, notebookID, userID string) error
}
