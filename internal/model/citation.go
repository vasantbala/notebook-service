package model

type citation struct {
	ID         string  `json:"id" db:"id"`
	Message    string  `json:"message_id" db:"message_id"`
	SourceID   string  `json:"source_id" db:"chunk_index"`
	ChunkIndex int     `json:"chunk_index" db:"chunk_index"`
	Score      float64 `json:"score" db:"score"` //cosine similarity
}
