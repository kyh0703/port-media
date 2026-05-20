package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/repository/model"
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
	if err := ensureSchema(ctx, db); err != nil {
		panic(err)
	}

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

func ensureSchema(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().
		Model((*model.Room)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("create rooms table: %w", err)
	}

	if _, err := db.NewCreateIndex().
		Model((*model.Room)(nil)).
		Index("idx_rooms_session_id").
		Column("session_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("create rooms session index: %w", err)
	}

	if err := migrateSchema(ctx, db); err != nil {
		return err
	}
	return nil
}

func migrateSchema(ctx context.Context, db *bun.DB) error {
	if err := ensureColumn(ctx, db, "rooms", "last_realtime_event_type", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(ctx, db, "rooms", "last_realtime_event_at", "TIMESTAMP"); err != nil {
		return err
	}
	return nil
}

func ensureColumn(ctx context.Context, db *bun.DB, table string, column string, definition string) error {
	exists, err := columnExists(ctx, db, table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if _, err := db.NewAddColumn().
		Table(table).
		ColumnExpr(fmt.Sprintf("%s %s", column, definition)).
		Exec(ctx); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}

func columnExists(ctx context.Context, db *bun.DB, table string, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("inspect table %s: %w", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, fmt.Errorf("scan table %s columns: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate table %s columns: %w", table, err)
	}
	return false, nil
}
