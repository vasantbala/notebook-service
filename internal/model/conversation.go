package model

import "time"

type Conversation struct {
	ID         string    `json:"id" db:"id"`
	NotebookID string    `json:"notebook_id" db:"notebook_id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Title      string    `json:"title" db:"title"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
