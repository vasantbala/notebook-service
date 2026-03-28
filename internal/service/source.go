package service

import (
    "context"
    "io"
    "github.com/vasantbala/notebook-service/internal/model"
)

type SourceService interface {
    ListSources(ctx context.Context, notebookID, userID string) ([]model.Source, error)
    GetSource(ctx context.Context, id, notebookID, userID string) (*model.Source, error)
    // UploadSource stores the file, creates the DB record, enqueues ingestion.
    UploadSource(ctx context.Context, notebookID, userID, filename, mimeType string, r io.Reader) (model.Source, error)
    DeleteSource(ctx context.Context, id, notebookID, userID string) error
}