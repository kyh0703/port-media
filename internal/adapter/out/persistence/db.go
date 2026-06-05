package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func NewDB(cfg *configs.Config) *bun.DB {
	sqldb, err := sql.Open(sqliteshim.ShimName, cfg.Database.DSN)
	if err != nil {
		panic(err)
	}

	sqldb.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	ctx := context.Background()
	if err := configureSQLite(ctx, sqldb, cfg.Database.BusyTimeout); err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	return db
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
