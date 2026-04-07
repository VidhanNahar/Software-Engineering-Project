package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func Connect(host, port, user, password, dbname string) (*sql.DB, error) {
	sslMode := os.Getenv("DB_SSLMODE")
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool for high concurrency
	db.SetMaxOpenConns(150)                // Max concurrent connections
	db.SetMaxIdleConns(30)                 // Min idle connections to keep open
	db.SetConnMaxLifetime(0)               // No lifetime limit
	db.SetConnMaxIdleTime(5 * time.Minute) // Close idle connections after 5 min

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func Close(db *sql.DB) {
	if db != nil {
		db.Close()
	}
}
