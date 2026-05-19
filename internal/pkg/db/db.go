package db

import (
	"database/sql"
	"fmt"

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

	if _, err := sqldb.Exec(Schema); err != nil {
		panic(err)
	}
	if err := migrateSchema(sqldb); err != nil {
		panic(err)
	}

	return bun.NewDB(sqldb, sqlitedialect.New())
}

func migrateSchema(db *sql.DB) error {
	if err := ensureColumn(db, "rooms", "last_realtime_event_type", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "rooms", "last_realtime_event_at", "TIMESTAMP"); err != nil {
		return err
	}
	return nil
}

func ensureColumn(db *sql.DB, table string, column string, definition string) error {
	exists, err := columnExists(db, table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}

func columnExists(db *sql.DB, table string, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
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
