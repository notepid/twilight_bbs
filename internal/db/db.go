package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection.
type DB struct {
	*sql.DB
}

// Open creates or opens a SQLite database at the given path.
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}

	// Enable WAL mode for better concurrent read performance.
	// NOTE: On some filesystems (notably Windows bind mounts under Docker Desktop),
	// changing journal modes can fail with "disk I/O error". In that case, we log
	// and continue with SQLite's default journaling rather than refusing to start.
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Warning: failed to enable WAL mode (%v); continuing without WAL", err)
	}

	// Enable foreign keys
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	db := &DB{DB: sqlDB}

	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// migrate runs all database migrations.
func (db *DB) migrate() error {
	// Create migrations tracking table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	for i, m := range migrations {
		version := i + 1
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if count > 0 {
			continue
		}

		log.Printf("Running migration %d: %s", version, m.name)
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("migration %d (%s): %w", version, m.name, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("record migration %d: %w", version, err)
		}
	}

	return nil
}
