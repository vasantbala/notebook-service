package model

type Citation struct {
	ID         string  `json:"id"          db:"id"`
	MessageID  string  `json:"message_id"  db:"message_id"`
	SourceID   string  `json:"source_id"   db:"source_id"`
	ChunkIndex int     `json:"chunk_index" db:"chunk_index"`
	Score      float64 `json:"score"       db:"score"` // cosine similarity
}
