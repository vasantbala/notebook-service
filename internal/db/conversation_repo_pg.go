package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vasantbala/notebook-service/internal/model"
)

type pgConversationRepo struct {
	pool *pgxpool.Pool
}

func NewPGConversationRepo(pool *pgxpool.Pool) ConversationRepository {
	return &pgConversationRepo{pool: pool}
}

func (r *pgConversationRepo) ListConversations(ctx context.Context, notebookID, userID string) ([]model.Conversation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, notebook_id, user_id, title, created_at, updated_at 
				FROM conversations WHERE user_id = $1 AND notebook_id = $2`, userID, notebookID)
	if err != nil {
		return nil, fmt.Errorf("db list conversations %w", err)
	}

	defer rows.Close()

	var conversations []model.Conversation
	for rows.Next() {
		var cv model.Conversation
		err := rows.Scan(&cv.ID, &cv.NotebookID, &cv.UserID, &cv.Title, &cv.CreatedAt, &cv.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("db list conversations scan %w", err)
		}

		conversations = append(conversations, cv)
	}

	return conversations, rows.Err()

}

func (r *pgConversationRepo) GetConversation(ctx context.Context, id, notebookID, userID string) (*model.Conversation, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, notebook_id, user_id, title, created_at, updated_at 
				FROM conversations WHERE user_id = $1 AND notebook_id = $2 AND id = $3`, userID, notebookID, id)
	var cv model.Conversation
	err := row.Scan(&cv.ID, &cv.NotebookID, &cv.UserID, &cv.Title, &cv.CreatedAt, &cv.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("db list conversations scan %w", err)
	}

	return &cv, nil
}

func (r *pgConversationRepo) CreateConversation(ctx context.Context, notebookID, userID, title string) (model.Conversation, error) {
	now := time.Now().UTC()
	var cv model.Conversation
	err := r.pool.QueryRow(ctx,
		`INSERT INTO conversations(notebook_id, user_id, title, created_at, updated_at)
		 VALUES($1, $2, $3, $4, $5)
		 RETURNING id, notebook_id, user_id, title, created_at, updated_at`,
		notebookID, userID, title, now, now,
	).Scan(&cv.ID, &cv.NotebookID, &cv.UserID, &cv.Title, &cv.CreatedAt, &cv.UpdatedAt)
	if err != nil {
		return model.Conversation{}, fmt.Errorf("db create conversation: %w", err)
	}
	return cv, nil
}

func (r *pgConversationRepo) DeleteConversation(ctx context.Context, id, notebookID, userID string) error {

	_, err := r.pool.Exec(ctx, `DELETE FROM conversations WHERE user_id = $1 AND notebook_id = $2 AND id = $3`, userID, notebookID, id)

	if err != nil {
		return fmt.Errorf("db delete conversations %w", err)
	}

	return nil
}

func (r *pgConversationRepo) ListMessages(ctx context.Context, conversationID, userID string) ([]model.Message, error) {
	// JOIN to conversations so userID is validated — messages have no user_id column.
	// LEFT JOIN citations because most messages will have none.
	// One row is returned per citation; messages with no citations produce one row
	// with NULL citation columns.
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.conversation_id, m.role, m.content, m.token_count, m.created_at,
		        c.id, c.message_id, c.source_id, c.chunk_index, c.score
		 FROM   messages m
		 JOIN   conversations cv ON cv.id = m.conversation_id
		 LEFT JOIN citations c   ON c.message_id = m.id
		 WHERE  m.conversation_id = $1
		   AND  cv.user_id = $2
		 ORDER  BY m.created_at, c.chunk_index`,
		conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("db list messages: %w", err)
	}
	defer rows.Close()

	// Use a map to group citation rows back onto their parent message.
	var ordered []string // preserves insertion order
	index := map[string]*model.Message{}

	for rows.Next() {
		var msg model.Message
		// Citation columns are nullable because of the LEFT JOIN.
		var (
			citID        *string
			citMessageID *string
			citSourceID  *string
			citChunkIdx  *int
			citScore     *float64
		)
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.TokenCount, &msg.CreatedAt,
			&citID, &citMessageID, &citSourceID, &citChunkIdx, &citScore,
		); err != nil {
			return nil, fmt.Errorf("db list messages scan: %w", err)
		}

		// First time we see this message ID — add it to the map.
		if _, seen := index[msg.ID]; !seen {
			index[msg.ID] = &msg
			ordered = append(ordered, msg.ID)
		}

		// Only build a Citation when the LEFT JOIN actually matched a row.
		if citID != nil {
			index[msg.ID].Citations = append(index[msg.ID].Citations, model.Citation{
				ID:         *citID,
				MessageID:  *citMessageID,
				SourceID:   *citSourceID,
				ChunkIndex: *citChunkIdx,
				Score:      *citScore,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db list messages rows: %w", err)
	}

	result := make([]model.Message, 0, len(ordered))
	for _, id := range ordered {
		result = append(result, *index[id])
	}
	return result, nil
}
func (r *pgConversationRepo) AddMessage(
	ctx context.Context,
	conversationID string,
	role model.Role,
	content string,
	tokenCount int,
	citations []model.Citation,
) (model.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Message{}, fmt.Errorf("db add message begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var msg model.Message
	err = tx.QueryRow(ctx,
		`INSERT INTO messages(conversation_id, role, content, token_count)
                 VALUES($1, $2, $3, $4)
                 RETURNING id, conversation_id, role, content, token_count, created_at`,
		conversationID, role, content, tokenCount,
	).Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.TokenCount, &msg.CreatedAt)
	if err != nil {
		return model.Message{}, fmt.Errorf("db add message insert: %w", err)
	}

	for i := range citations {
		var citID string
		err = tx.QueryRow(ctx,
			`INSERT INTO citations(message_id, source_id, chunk_index, score)
                         VALUES($1, $2, $3, $4)
                         RETURNING id`,
			msg.ID, citations[i].SourceID, citations[i].ChunkIndex, citations[i].Score,
		).Scan(&citID)
		if err != nil {
			return model.Message{}, fmt.Errorf("db add message citation %d: %w", i, err)
		}
		citations[i].ID = citID
		citations[i].MessageID = msg.ID
	}

	if err = tx.Commit(ctx); err != nil {
		return model.Message{}, fmt.Errorf("db add message commit: %w", err)
	}

	msg.Citations = citations
	return msg, nil
}
