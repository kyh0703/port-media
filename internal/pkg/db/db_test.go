package db

import (
	"database/sql"
	"testing"

	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestMigrateSchemaAddsRealtimeEventColumnsToExistingRoomsTable(t *testing.T) {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file:test-migrate-schema?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		_ = sqldb.Close()
	}()

	_, err = sqldb.Exec(`
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

	if err := migrateSchema(sqldb); err != nil {
		t.Fatalf("migrateSchema() error = %v", err)
	}

	if exists, err := columnExists(sqldb, "rooms", "last_realtime_event_type"); err != nil || !exists {
		t.Fatalf("last_realtime_event_type exists=%v err=%v, want true nil", exists, err)
	}
	if exists, err := columnExists(sqldb, "rooms", "last_realtime_event_at"); err != nil || !exists {
		t.Fatalf("last_realtime_event_at exists=%v err=%v, want true nil", exists, err)
	}
}

func TestMigrateSchemaIsIdempotent(t *testing.T) {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file:test-migrate-schema-idempotent?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		_ = sqldb.Close()
	}()

	if _, err := sqldb.Exec(Schema); err != nil {
		t.Fatalf("exec schema error = %v", err)
	}
	if err := migrateSchema(sqldb); err != nil {
		t.Fatalf("first migrateSchema() error = %v", err)
	}
	if err := migrateSchema(sqldb); err != nil {
		t.Fatalf("second migrateSchema() error = %v", err)
	}
}
