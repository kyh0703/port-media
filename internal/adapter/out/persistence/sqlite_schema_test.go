package persistence

import (
	"context"
	"database/sql"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestEnsureSchemaCreatesRoomsTableAndIndexFromBunModel(t *testing.T) {
	db := newTestDB(t, "file:test-ensure-schema?mode=memory&cache=shared")
	defer func() {
		_ = db.Close()
	}()

	if err := ensureSchema(context.Background(), db); err != nil {
		t.Fatalf("ensureSchema() error = %v", err)
	}

	if exists, err := columnExists(context.Background(), db, "rooms", "session_id"); err != nil || !exists {
		t.Fatalf("session_id exists=%v err=%v, want true nil", exists, err)
	}
	if exists, err := indexExists(context.Background(), db, "idx_rooms_session_id"); err != nil || !exists {
		t.Fatalf("idx_rooms_session_id exists=%v err=%v, want true nil", exists, err)
	}
}

func TestMigrateSchemaAddsRealtimeEventColumnsToExistingRoomsTable(t *testing.T) {
	sqldb := newTestSQLDB(t, "file:test-migrate-schema?mode=memory&cache=shared")
	defer func() {
		_ = sqldb.Close()
	}()

	_, err := sqldb.Exec(`
CREATE TABLE rooms (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  user_id TEXT,
  status TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
)`)
	if err != nil {
		t.Fatalf("create old rooms table error = %v", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	if err := migrateSchema(context.Background(), db); err != nil {
		t.Fatalf("migrateSchema() error = %v", err)
	}

	if exists, err := columnExists(context.Background(), db, "rooms", "last_realtime_event_type"); err != nil || !exists {
		t.Fatalf("last_realtime_event_type exists=%v err=%v, want true nil", exists, err)
	}
	if exists, err := columnExists(context.Background(), db, "rooms", "last_realtime_event_at"); err != nil || !exists {
		t.Fatalf("last_realtime_event_at exists=%v err=%v, want true nil", exists, err)
	}
}

func TestMigrateSchemaIsIdempotent(t *testing.T) {
	db := newTestDB(t, "file:test-migrate-schema-idempotent?mode=memory&cache=shared")
	defer func() {
		_ = db.Close()
	}()

	if err := ensureSchema(context.Background(), db); err != nil {
		t.Fatalf("first ensureSchema() error = %v", err)
	}
	if err := ensureSchema(context.Background(), db); err != nil {
		t.Fatalf("second ensureSchema() error = %v", err)
	}
}

func newTestDB(t *testing.T, dsn string) *bun.DB {
	t.Helper()

	return bun.NewDB(newTestSQLDB(t, dsn), sqlitedialect.New())
}

func newTestSQLDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	sqldb, err := sql.Open(sqliteshim.ShimName, dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	return sqldb
}

func indexExists(ctx context.Context, db *bun.DB, name string) (bool, error) {
	var found string
	err := db.QueryRowContext(
		ctx,
		"SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?",
		name,
	).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return found == name, nil
}
