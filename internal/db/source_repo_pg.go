package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vasantbala/notebook-service/internal/model"
)

type pgSourceRepo struct {
	pool *pgxpool.Pool
}

func NewPGSourceRepo(pool *pgxpool.Pool) SourceRepository {
	return &pgSourceRepo{pool: pool}
}

func (r *pgSourceRepo) ListSources(ctx context.Context, notebookID, userID string) ([]model.Source, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, notebook_id, user_id, filename, storage_key, mime_type,
		        status, chunk_count, COALESCE(rag_doc_id, ''), created_at, updated_at
		 FROM   sources
		 WHERE  notebook_id = $1 AND user_id = $2
		 ORDER  BY created_at DESC`,
		notebookID, userID)
	if err != nil {
		return nil, fmt.Errorf("db list sources: %w", err)
	}
	defer rows.Close()

	var sources []model.Source
	for rows.Next() {
		var s model.Source
		if err := rows.Scan(
			&s.ID, &s.NotebookID, &s.UserID, &s.Filename, &s.StorageKey,
			&s.MimeType, &s.Status, &s.ChunkCount, &s.RagDocID,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("db list sources scan: %w", err)
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func (r *pgSourceRepo) GetSource(ctx context.Context, id, notebookID, userID string) (*model.Source, error) {
	var s model.Source
	err := r.pool.QueryRow(ctx,
		`SELECT id, notebook_id, user_id, filename, storage_key, mime_type,
		        status, chunk_count, COALESCE(rag_doc_id, ''), created_at, updated_at
		 FROM   sources
		 WHERE  id = $1 AND notebook_id = $2 AND user_id = $3`,
		id, notebookID, userID,
	).Scan(
		&s.ID, &s.NotebookID, &s.UserID, &s.Filename, &s.StorageKey,
		&s.MimeType, &s.Status, &s.ChunkCount, &s.RagDocID,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("db get source: %w", err)
	}
	return &s, nil
}

func (r *pgSourceRepo) CreateSource(ctx context.Context, notebookID, userID, filename, storageKey, mimeType string) (model.Source, error) {
	now := time.Now().UTC()
	var s model.Source
	err := r.pool.QueryRow(ctx,
		`INSERT INTO sources(notebook_id, user_id, filename, storage_key, mime_type, created_at, updated_at)
		 VALUES($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, notebook_id, user_id, filename, storage_key, mime_type,
		           status, chunk_count, COALESCE(rag_doc_id, ''), created_at, updated_at`,
		notebookID, userID, filename, storageKey, mimeType, now, now,
	).Scan(
		&s.ID, &s.NotebookID, &s.UserID, &s.Filename, &s.StorageKey,
		&s.MimeType, &s.Status, &s.ChunkCount, &s.RagDocID,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return model.Source{}, fmt.Errorf("db create source: %w", err)
	}
	return s, nil
}

func (r *pgSourceRepo) UpdateStatus(ctx context.Context, id string, status model.SourceStatus, chunkCount int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sources SET status = $1, chunk_count = $2, updated_at = NOW()
		 WHERE  id = $3`,
		status, chunkCount, id)
	if err != nil {
		return fmt.Errorf("db update source status: %w", err)
	}
	return nil
}

func (r *pgSourceRepo) UpdateRagDocID(ctx context.Context, id, ragDocID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sources SET rag_doc_id = $1, updated_at = NOW()
		 WHERE  id = $2`,
		ragDocID, id)
	if err != nil {
		return fmt.Errorf("db update source rag_doc_id: %w", err)
	}
	return nil
}

func (r *pgSourceRepo) ListRagDocIDs(ctx context.Context, notebookID, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT rag_doc_id FROM sources
		 WHERE  notebook_id = $1 AND user_id = $2
		   AND  status = 'ready' AND rag_doc_id IS NOT NULL`,
		notebookID, userID)
	if err != nil {
		return nil, fmt.Errorf("db list rag doc ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("db list rag doc ids scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *pgSourceRepo) DeleteSource(ctx context.Context, id, notebookID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM sources WHERE id = $1 AND notebook_id = $2 AND user_id = $3`,
		id, notebookID, userID)
	if err != nil {
		return fmt.Errorf("db delete source: %w", err)
	}
	return nil
}
