package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

const (
	defaultDatabaseBusyTimeout = 5 * time.Second
	defaultDatabaseMaxOpenConn = 10
	defaultDatabaseMaxIdleConn = 5
)

func NewDB(cfg *configs.Config) *bun.DB {
	if err := ensureSQLiteFileDirectory(cfg.Database.URL); err != nil {
		panic(err)
	}

	sqldb, err := sql.Open(sqliteshim.ShimName, cfg.Database.URL)
	if err != nil {
		panic(err)
	}

	sqldb.SetMaxOpenConns(defaultDatabaseMaxOpenConn)
	sqldb.SetMaxIdleConns(defaultDatabaseMaxIdleConn)

	ctx := context.Background()
	if err := configureSQLite(ctx, sqldb, defaultDatabaseBusyTimeout); err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	return db
}

func ensureSQLiteFileDirectory(dsn string) error {
	dbPath := sqliteFilePath(dsn)
	if dbPath == "" {
		return nil
	}

	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create sqlite directory %s: %w", dir, err)
	}
	return nil
}

func sqliteFilePath(dsn string) string {
	if dsn == "" || dsn == ":memory:" || strings.Contains(dsn, "mode=memory") {
		return ""
	}

	path := dsn
	if strings.HasPrefix(path, "file:") {
		path = strings.TrimPrefix(path, "file:")
	}
	if beforeQuery, _, ok := strings.Cut(path, "?"); ok {
		path = beforeQuery
	}
	if path == "" || strings.HasPrefix(path, ":") {
		return ""
	}
	return path
}

func configureSQLite(ctx context.Context, db *sql.DB, busyTimeout time.Duration) error {
	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode = WAL").Scan(&journalMode); err != nil {
		return fmt.Errorf("enable sqlite WAL journal mode: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("enable sqlite WAL journal mode: got %q", journalMode)
	}
	if busyTimeout > 0 {
		timeoutMs := busyTimeout.Milliseconds()
		if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", timeoutMs)); err != nil {
			return fmt.Errorf("set sqlite busy timeout: %w", err)
		}
	}
	return nil
}
