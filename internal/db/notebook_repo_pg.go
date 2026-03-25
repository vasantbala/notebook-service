package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vasantbala/notebook-service/internal/model"
)

type pgNotebookRepo struct {
	pool *pgxpool.Pool
}

func NewPgNotebookRepo(pool *pgxpool.Pool) NotebookRepository {
	return &pgNotebookRepo{pool: pool}
}

func (r *pgNotebookRepo) List(ctx context.Context, userID string) ([]model.Notebook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, title, description, created_at, updated_at
		FROM notebooks WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("db list notebooks: %w", err)
	}

	defer rows.Close()

	var notebooks []model.Notebook
	for rows.Next() {
		var nb model.Notebook
		if err := rows.Scan(&nb.ID, &nb.UserID, &nb.Title, &nb.Description, &nb.CreatedAt, &nb.UpdatedAt); err != nil {
			return nil, fmt.Errorf("db scan notebook: %w", err)
		}
		notebooks = append(notebooks, nb)
	}

	return notebooks, rows.Err()
}

func (r *pgNotebookRepo) Get(ctx context.Context, id, userID string) (*model.Notebook, error) {
	var nb model.Notebook
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, created_at, updated_at
		FROM notebooks WHERE id = $1 AND user_id = $2`, id, userID).Scan(&nb.ID, &nb.UserID, &nb.Title, &nb.Description, &nb.CreatedAt, &nb.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%s not found", id)
	}

	if err != nil {
		return nil, fmt.Errorf("db get notebook: %w", err)
	}

	return &nb, nil
}

func (r *pgNotebookRepo) Create(ctx context.Context, nb model.Notebook) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notebooks (user_id, title, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`, nb.UserID, nb.Title, nb.Description, nb.CreatedAt, nb.UpdatedAt)

	if err != nil {
		return fmt.Errorf("db create notebook: %w", err)
	}

	return nil
}

func (r *pgNotebookRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM notebooks WHERE user_id = $1 AND id = $2`,
		id, userID)

	if err != nil {
		return fmt.Errorf("db delete notebook: %w", err)
	}

	return nil
}
