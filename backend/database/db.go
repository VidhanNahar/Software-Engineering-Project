package database

import (
	"database/sql"
	"fmt"
	"os"

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
