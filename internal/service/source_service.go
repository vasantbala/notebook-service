package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

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
		if err := s.ingestToRAG(bgCtx, src.ID, filename, bearerToken, fileBytes); err != nil {
			// Log only — the handler has already returned. Callers poll status via GetSource.
			fmt.Printf("rag-anything ingest failed for source %s: %v\n", src.ID, err)
			_ = s.repo.UpdateStatus(bgCtx, src.ID, model.SourceStatusFailed, 0)
		}
	}()

	return src, nil
}

func (s *sourceService) ingestToRAG(ctx context.Context, sourceID, filename, bearerToken string, fileBytes []byte) error {
	_ = s.repo.UpdateStatus(ctx, sourceID, model.SourceStatusProcessing, 0)

	// Build a multipart/form-data body with the file — rag-anything's /upload
	// endpoint expects an UploadFile field named "file".
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err = fw.Write(fileBytes); err != nil {
		return fmt.Errorf("write form file: %w", err)
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ragAnythingURL+"/documents/upload", &buf)
	if err != nil {
		return fmt.Errorf("build ingest request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call rag-anything /documents/upload: %w", err)
	}
	defer resp.Body.Close()

	// /upload returns 202 Accepted with {doc_id, status}; ingestion runs in the background.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("rag-anything /documents/upload returned %d", resp.StatusCode)
	}

	// Store the doc_id rag-anything assigned so we can filter /retrieve calls later.
	var result struct {
		DocID  string `json:"doc_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode upload response: %w", err)
	}

	if err := s.repo.UpdateRagDocID(ctx, sourceID, result.DocID); err != nil {
		return fmt.Errorf("update rag_doc_id: %w", err)
	}
	// rag-anything processes async; poll until it reports ready or failed.
	if err := s.pollUntilDone(ctx, sourceID, result.DocID, bearerToken); err != nil {
		return fmt.Errorf("polling rag status: %w", err)
	}
	return nil
}

// pollUntilDone polls GET /documents on rag-anything every 5 seconds until the
// document's status is "ready" or "failed", then updates the notebook-service DB.
// It gives up after 10 minutes.
func (s *sourceService) pollUntilDone(ctx context.Context, sourceID, docID, bearerToken string) error {
	const (
		interval = 5 * time.Second
		timeout  = 10 * time.Minute
	)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		ragStatus, chunkCount, err := s.fetchDocStatus(ctx, docID, bearerToken)
		if err != nil {
			// transient error — keep polling
			fmt.Printf("poll rag status error (source=%s): %v\n", sourceID, err)
			continue
		}

		switch ragStatus {
		case "ready":
			return s.repo.UpdateStatus(ctx, sourceID, model.SourceStatusReady, chunkCount)
		case "failed":
			_ = s.repo.UpdateStatus(ctx, sourceID, model.SourceStatusFailed, 0)
			return fmt.Errorf("rag-anything reported failed for doc_id=%s", docID)
		}
		// still "processing" — keep waiting
	}
	return fmt.Errorf("rag-anything ingestion timed out after %s for doc_id=%s", timeout, docID)
}

// fetchDocStatus calls GET /documents on rag-anything and finds the status for docID.
func (s *sourceService) fetchDocStatus(ctx context.Context, docID, bearerToken string) (status string, chunkCount int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.ragAnythingURL+"/documents", nil)
	if err != nil {
		return "", 0, fmt.Errorf("build status request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("GET /documents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("GET /documents returned %d", resp.StatusCode)
	}

	var docs []struct {
		DocID      string `json:"doc_id"`
		Status     string `json:"status"`
		ChunkCount int    `json:"chunk_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&docs); err != nil {
		return "", 0, fmt.Errorf("decode /documents response: %w", err)
	}

	for _, d := range docs {
		if d.DocID == docID {
			return d.Status, d.ChunkCount, nil
		}
	}
	return "", 0, fmt.Errorf("doc_id=%s not found in /documents response", docID)
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
