package talks

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	conn *pgx.Conn
}

// NewRepository creates a new repository. It will receive a context and the PostgreSQL connection string.
func NewRepository(ctx context.Context, connStr string) (*Repository, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}

	return &Repository{
		conn: conn,
	}, nil
}

// Create creates a new talk in the database.
// It uses value semantics at the method receiver to avoid mutating the original repository.
// It uses pointer semantics at the talk parameter to avoid copying the struct, modifying it and returning it.
func (r Repository) Create(ctx context.Context, talk *Talk) error {
	query := "INSERT INTO talks (uuid, title) VALUES ($1, $2) RETURNING id"

	return r.conn.QueryRow(ctx, query, talk.UUID, talk.Title).Scan(&talk.ID)
}

// GetByUUID retrieves a talk from the database by its UUID.
func (r Repository) GetByUUID(ctx context.Context, uid string) (Talk, error) {
	query := "SELECT id, uuid, title FROM talks WHERE uuid = $1"

	var talk Talk
	err := r.conn.QueryRow(ctx, query, uid).Scan(&talk.ID, &talk.UUID, &talk.Title)
	if err != nil {
		return Talk{}, err
	}

	return talk, nil
}
