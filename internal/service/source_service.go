package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/vasantbala/notebook-service/internal/db"
	"github.com/vasantbala/notebook-service/internal/model"
)

type sourceService struct {
	repo           db.SourceRepository
	ragAnythingURL string // base URL of the rag-anything service, e.g. "http://rag-anything:8000"
	httpClient     *http.Client
}

func NewSourceService(repo db.SourceRepository, ragAnythingURL string) SourceService {
	return &sourceService{
		repo:           repo,
		ragAnythingURL: ragAnythingURL,
		httpClient:     &http.Client{},
	}
}

func (s *sourceService) ListSources(ctx context.Context, notebookID, userID string) ([]model.Source, error) {
	sources, err := s.repo.ListSources(ctx, notebookID, userID)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	return sources, nil
}

func (s *sourceService) GetSource(ctx context.Context, id, notebookID, userID string) (*model.Source, error) {
	src, err := s.repo.GetSource(ctx, id, notebookID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get source: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("get source: %w", err)
	}
	return src, nil
}

// UploadSource creates the DB record, then calls rag-anything's ingest endpoint
// to kick off async chunking and embedding. The source is returned with status=pending;
// the handler should poll GetSource (or use a webhook) to check when it becomes ready.
//
// content holds the raw file bytes that the caller read from the upload request.
// The caller is responsible for uploading the file to object storage (S3/local) first
// and passing the resulting storage key.
func (s *sourceService) UploadSource(ctx context.Context, notebookID, userID, filename, storageKey, mimeType, bearerToken string, content io.Reader) (model.Source, error) {
	// 1. Create the DB record (status defaults to 'pending' in the DB).
	src, err := s.repo.CreateSource(ctx, notebookID, userID, filename, storageKey, mimeType)
	if err != nil {
		return model.Source{}, fmt.Errorf("upload source create record: %w", err)
	}

	// 2. Read file bytes for the rag-anything ingest call.
	fileBytes, err := io.ReadAll(content)
	if err != nil {
		return model.Source{}, fmt.Errorf("upload source read content: %w", err)
	}

	// 3. Call rag-anything ingest — fire and forget (status updated via callback or polling).
	//    Run in a goroutine so the handler can return immediately with status=pending.
	go func() {
		bgCtx := context.Background() // request context will be cancelled after handler returns
		if err := s.ingestToRAG(bgCtx, src.ID, userID, filename, mimeType, bearerToken, fileBytes); err != nil {
			// Log only — the handler has already returned. Callers poll status via GetSource.
			fmt.Printf("rag-anything ingest failed for source %s: %v\n", src.ID, err)
			_ = s.repo.UpdateStatus(bgCtx, src.ID, model.SourceStatusFailed, 0)
		}
	}()

	return src, nil
}

func (s *sourceService) ingestToRAG(ctx context.Context, sourceID, userID, filename, mimeType, bearerToken string, fileBytes []byte) error {
	_ = s.repo.UpdateStatus(ctx, sourceID, model.SourceStatusProcessing, 0)

	body, err := json.Marshal(map[string]any{
		"doc_id":      sourceID,
		"user_id":     userID,
		"filename":    filename,
		"source_type": mimeType,
	})
	if err != nil {
		return fmt.Errorf("marshal ingest request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ragAnythingURL+"/ingest", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build ingest request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call rag-anything /ingest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("rag-anything /ingest returned %d", resp.StatusCode)
	}

	// Parse the doc_id and chunk_count from the response.
	var result struct {
		DocID      string `json:"doc_id"`
		ChunkCount int    `json:"chunk_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode ingest response: %w", err)
	}

	if err := s.repo.UpdateRagDocID(ctx, sourceID, result.DocID); err != nil {
		return fmt.Errorf("update rag_doc_id: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, sourceID, model.SourceStatusReady, result.ChunkCount); err != nil {
		return fmt.Errorf("update source status: %w", err)
	}
	return nil
}

func (s *sourceService) DeleteSource(ctx context.Context, id, notebookID, userID string) error {
	if err := s.repo.DeleteSource(ctx, id, notebookID, userID); err != nil {
		return fmt.Errorf("delete source: %w", err)
	}
	return nil
}

func (s *sourceService) ListRagDocIDs(ctx context.Context, notebookID, userID string) ([]string, error) {
	ids, err := s.repo.ListRagDocIDs(ctx, notebookID, userID)
	if err != nil {
		return nil, fmt.Errorf("list rag doc ids: %w", err)
	}
	return ids, nil
}
