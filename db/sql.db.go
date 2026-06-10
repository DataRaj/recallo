package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// DB is the global database connection pool, unexported to enforce access via package functions.
var DB *sql.DB

// Config holds the tuning knobs for the connection pool.
type Config struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns sensible production defaults for the connection pool.
func DefaultConfig() Config {
	return Config{
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}
}

// InitDB opens a PostgreSQL connection pool using the provided DSN and applies
// the given pool Config. It performs a connectivity check (Ping) before returning.
//
// DSN format: postgres://user:password@host:port/dbname?sslmode=disable
func InitDB(dsn string, cfg Config) error {
	if dsn == "" {
		return fmt.Errorf("database DSN must not be empty")
	}

	var err error
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Apply connection pool settings.
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify the connection is alive.
	if err = db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	pragmas := []string{
		"SET synchronous_commit = 'local';",
		"SET lock_timeout = 5000;",
		"SET statement_timeout = 10000;", // "PRAGMA temp_store=MEMORY;",
	}

	for _, pragma := range pragmas {
		_, err := db.Exec(pragma)
		if err != nil {
			_ = db.Close()
			return fmt.Errorf("failed to set pragma '%s': %w", pragma, err)
		}
	}

	tables := []string{
		// Users
		`CREATE TABLE IF NOT EXISTS users (
        id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        name TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL,
        refresh_token_web TEXT,
        refresh_token_web_at TIMESTAMP,
        refresh_token_mobile TEXT,
        refresh_token_mobile_at TIMESTAMP,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`,

		// Privates
		`CREATE TABLE IF NOT EXISTS privates (
       id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        user1_id INTEGER NOT NULL,
        user2_id INTEGER NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(user1_id, user2_id),
        CHECK(user1_id < user2_id),
        FOREIGN KEY(user1_id) REFERENCES users(id) ON DELETE CASCADE,
        FOREIGN KEY(user2_id) REFERENCES users(id) ON DELETE CASCADE
    );`,

		// Messages
		`CREATE TABLE IF NOT EXISTS messages (
        id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		from_id INTEGER NOT NULL,
        private_id INTEGER,
        message_type TEXT NOT NULL,
        content TEXT NOT NULL,
        delivered INTEGER NOT NULL DEFAULT 0, -- Keeps 0/1 integers or you could switch to BOOLEAN DEFAULT FALSE
        read INTEGER NOT NULL DEFAULT 0,      -- Keeps 0/1 integers or you could switch to BOOLEAN DEFAULT FALSE
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(from_id) REFERENCES users(id) ON DELETE CASCADE,
        FOREIGN KEY(private_id) REFERENCES privates(id) ON DELETE CASCADE
    );`,
	}
	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			_ = db.Close()
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_messages_private_id ON messages(private_id);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_from_id ON messages(from_id);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_privates_user1_id ON privates(user1_id);`,
		`CREATE INDEX IF NOT EXISTS idx_privates_user2_id ON privates(user2_id);`,
	}
	for _, index := range indexes {
		_, err := db.Exec(index)
		if err != nil {
			_ = db.Close()
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	DB = db

	log.Println("Database connection pool established successfully")
	return nil
}

// GetDB returns the global *sql.DB instance.
// Callers should not hold on to this reference beyond a single request lifecycle.
func GetDB() (*sql.DB, error) {
	return DB, nil
}

// CloseDBConnection drains and closes the connection pool gracefully.
// Should be called via defer in main() after a shutdown signal is received.
func CloseDBConnection() {
	if DB == nil {
		return
	}

	if err := DB.Close(); err != nil {
		log.Printf("Error closing database connection pool: %v", err)
	} else {
		log.Println("Database connection pool closed successfully")
	}
}
