package model

import "time"

type Role string

const (
	RoleUser      Role = "User"
	RoleAssistant Role = "Assistant"
	RoleMessage   Role = "Message"
)

type Message struct {
	ID             string     `json:"id" db:"id"`
	ConversationID string     `json:"conversation_id" db:"conversation_id"`
	Role           Role       `json:"role" db:"role"`
	Content        string     `json:"content" db:"content"`
	TokenCount     int        `json:"token_count" db:"token_count"`
	Citations      []Citation `json:"citations,omitempty" db:"-"` // loaded separately
	CreatedAt      time.Time  `json:"created_at"      db:"created_at"`
}
