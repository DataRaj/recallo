package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// DB is the global database connection pool, unexported to enforce access via package functions.
var db *sql.DB

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
	db, err = sql.Open("postgres", dsn)
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

	log.Println("Database connection pool established successfully")
	return nil
}

// GetDB returns the global *sql.DB instance.
// Callers should not hold on to this reference beyond a single request lifecycle.
func GetDB() *sql.DB {
	return db
}

// CloseDBConnection drains and closes the connection pool gracefully.
// Should be called via defer in main() after a shutdown signal is received.
func CloseDBConnection() {
	if db == nil {
		return
	}

	if err := db.Close(); err != nil {
		log.Printf("Error closing database connection pool: %v", err)
	} else {
		log.Println("Database connection pool closed successfully")
	}
}
