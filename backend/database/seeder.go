package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

var migrationFiles = []string{
	"schema.sql",
	"user_role_kyc_upgrade.sql",
	"stock_bhavcopy_upgrade.sql",
	"alerts_notifications_upgrade.sql",
}

// RunAutoSeeder applies SQL migrations at startup so new developers do not need
// to run migration commands manually.
func RunAutoSeeder(db *sql.DB) error {
	basePaths := []string{
		"migrations",
		filepath.Join("backend", "migrations"),
	}

	migrationsPath := ""
	for _, p := range basePaths {
		if _, err := os.Stat(p); err == nil {
			migrationsPath = p
			break
		}
	}

	if migrationsPath == "" {
		return fmt.Errorf("migrations directory not found; checked: %v", basePaths)
	}

	for _, fileName := range migrationFiles {
		fullPath := filepath.Join(migrationsPath, fileName)
		sqlBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", fullPath, err)
		}

		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", fullPath, err)
		}
	}

	return nil
}
