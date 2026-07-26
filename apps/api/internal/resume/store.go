package resume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	List(ctx context.Context, limit, offset int) ([]Resume, error)
	Create(ctx context.Context, input Input) (Resume, error)
	Get(ctx context.Context, id uuid.UUID) (Resume, error)
	Update(ctx context.Context, id uuid.UUID, input Input) (Resume, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Duplicate(ctx context.Context, id uuid.UUID) (Resume, error)
}

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) List(ctx context.Context, limit, offset int) ([]Resume, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, document, created_at, updated_at
		FROM resumes
		ORDER BY updated_at DESC, id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list resumes: %w", err)
	}
	defer rows.Close()

	items := make([]Resume, 0)
	for rows.Next() {
		item, err := scanResume(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate resumes: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) Create(ctx context.Context, input Input) (Resume, error) {
	document, err := json.Marshal(input.Document)
	if err != nil {
		return Resume{}, fmt.Errorf("encode resume document: %w", err)
	}
	id := uuid.New()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO resumes (id, title, document)
		VALUES ($1, $2, $3)
		RETURNING id, title, document, created_at, updated_at`, id, input.Title, document)
	return scanResume(row)
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (Resume, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, title, document, created_at, updated_at
		FROM resumes WHERE id = $1`, id)
	return scanResume(row)
}

func (s *PostgresStore) Update(ctx context.Context, id uuid.UUID, input Input) (Resume, error) {
	document, err := json.Marshal(input.Document)
	if err != nil {
		return Resume{}, fmt.Errorf("encode resume document: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE resumes
		SET title = $2, document = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, title, document, created_at, updated_at`, id, input.Title, document)
	return scanResume(row)
}

func (s *PostgresStore) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM resumes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete resume: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Duplicate(ctx context.Context, id uuid.UUID) (Resume, error) {
	newID := uuid.New()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO resumes (id, title, document)
		SELECT $2, left(title || ' copy', 200), document
		FROM resumes WHERE id = $1
		RETURNING id, title, document, created_at, updated_at`, id, newID)
	return scanResume(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanResume(row rowScanner) (Resume, error) {
	var item Resume
	var document []byte
	if err := row.Scan(&item.ID, &item.Title, &document, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Resume{}, ErrNotFound
		}
		return Resume{}, fmt.Errorf("scan resume: %w", err)
	}
	var err error
	item.Document, err = DecodeDocument(document)
	if err != nil {
		return Resume{}, err
	}
	return item, nil
}
