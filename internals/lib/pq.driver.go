package lib

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/lib/pq"
)

// CreateUser inserts a new user and handles a potential unique constraint violation.
func CreateUser(ctx context.Context, db *sql.DB, email string, name string) error {
	query := `INSERT INTO users (email, name) VALUES ($1, $2)`

	_, err := db.ExecContext(ctx, query, email, name)
	if err != nil {
		// Use errors.As to check if the error is a pq.Error
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			log.Printf("PostgreSQL error: Code=%s, Message=%s", pqErr.Code, pqErr.Message)
			// Check for the "unique_violation" error code.
			// See https://www.postgresql.org/docs/current/errcodes-appendix.html
			if pqErr.Code == "23505" {
				return errors.New("user with this email already exists")
			}
			// Handle other specific codes as needed.
		}
		// Return a generic error for other cases.
		return errors.New("failed to create user")
	}

	return nil
}
