package database

import (
	"context"
	"database/sql"
	"fmt"

	migrate "github.com/rubenv/sql-migrate"
	_ "modernc.org/sqlite"

	embedded "github.com/juho05/sheetopia-sync/sql"
)

func autoMigrate(db *sql.DB) error {
	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: embedded.MigrationsFS,
		Root:       "migrations",
	}
	_, err := migrate.Exec(db, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("exec migrations: %w", err)
	}
	return nil
}

func Open(ctx context.Context, filePath string) (db *sql.DB, queries *Queries, err error) {
	db, err = sql.Open("sqlite", fmt.Sprintf("%s?_pragma=journal_mode=WAL&_pragma=foreign_keys=ON&_pragma=busy_timeout=3000&_txlock=immediate&_time_integer_format=unix&_inttotime=1", filePath))
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite db: %w", err)
	}

	err = autoMigrate(db)
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("auto migrate: %w", err)
	}

	return db, New(db), nil
}
