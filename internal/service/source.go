package service

import (
	"context"
	"io"

	"github.com/vasantbala/notebook-service/internal/model"
)

type SourceService interface {
	ListSources(ctx context.Context, notebookID, userID string) ([]model.Source, error)
	GetSource(ctx context.Context, id, notebookID, userID string) (*model.Source, error)
	// UploadSource stores the raw file bytes, creates the DB record, then calls rag-anything
	// to ingest the document. Returns the source record with status=pending; ingestion is async.
	// storageKey is the object key under which the caller has already stored the file (e.g. S3 key).
	// bearerToken is the caller's JWT and is forwarded to rag-anything for auth.
	UploadSource(ctx context.Context, notebookID, userID, filename, storageKey, mimeType, bearerToken string, content io.Reader) (model.Source, error)
	DeleteSource(ctx context.Context, id, notebookID, userID string) error
	// ListRagDocIDs returns the rag_doc_id values for all ready sources in a notebook.
	// Used by the retrieval handler to build the doc_ids filter for rag-anything's /retrieve.
	ListRagDocIDs(ctx context.Context, notebookID, userID string) ([]string, error)
}
