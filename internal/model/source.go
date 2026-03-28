package model

import "time"

type SourceStatus string

const (
	SourceStatusPending    SourceStatus = "pending"
	SourceStatusProcessing SourceStatus = "processing"
	SourceStatusReady      SourceStatus = "ready"
	SourceStatusFailed     SourceStatus = "failed"
)

type Source struct {
	ID         string       `json:"id"          db:"id"`
	NotebookID string       `json:"notebook_id" db:"notebook_id"`
	UserID     string       `json:"user_id"     db:"user_id"`
	Filename   string       `json:"filename"    db:"filename"`
	StorageKey string       `json:"storage_key" db:"storage_key"` // e.g. S3 object key
	MimeType   string       `json:"mime_type"   db:"mime_type"`
	Status     SourceStatus `json:"status"      db:"status"`
	ChunkCount int          `json:"chunk_count" db:"chunk_count"`
	RagDocID   string       `json:"rag_doc_id"  db:"rag_doc_id"` // doc_id from rag-anything
	CreatedAt  time.Time    `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"  db:"updated_at"`
}
