package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ChunkResult mirrors the RetrievedChunkModel returned by rag-anything's /retrieve.
type ChunkResult struct {
	Text          string         `json:"text"`
	DocID         string         `json:"doc_id"`
	SourceType    string         `json:"source_type"`
	ChunkIndex    int            `json:"chunk_index"`
	PageNumber    *int           `json:"page_number"`
	RerankerScore float64        `json:"reranker_score"`
	Metadata      map[string]any `json:"metadata"`
}

type RetrievalService interface {
	// Search returns the top-K most relevant chunks for the given doc IDs.
	// docIDs should be the rag_doc_id values of the notebook's ready sources.
	Search(ctx context.Context, query, userID string, docIDs []string, topK int) ([]ChunkResult, error)
}

type ragAnythingClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewRAGAnythingClient(baseURL string) RetrievalService {
	return &ragAnythingClient{baseURL: baseURL, httpClient: &http.Client{}}
}

func (c *ragAnythingClient) Search(ctx context.Context, query, userID string, docIDs []string, topK int) ([]ChunkResult, error) {
	body, _ := json.Marshal(map[string]any{
		"question": query,
		"doc_ids":  docIDs,
		"top_k":    topK,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/retrieve", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build retrieve request: %w", err)
	}
	// Forward the caller's auth token so rag-anything can enforce its own authz.
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call rag-anything /retrieve: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rag-anything /retrieve returned %d", resp.StatusCode)
	}

	var result struct {
		Chunks     []ChunkResult `json:"chunks"`
		Sufficient bool          `json:"sufficient"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode retrieve response: %w", err)
	}
	return result.Chunks, nil
}
