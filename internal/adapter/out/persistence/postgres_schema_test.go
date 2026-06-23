package persistence

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestCreateRoomsTableQueryUsesPostgresDDL(t *testing.T) {
	db := newMockPostgresDB(t)
	defer func() {
		_ = db.Close()
	}()

	query := createRoomsTableQuery(db).String()

	for _, want := range []string{
		`CREATE TABLE IF NOT EXISTS "rooms"`,
		`"id" text`,
		`"session_id" text NOT NULL`,
		`PRIMARY KEY ("id")`,
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("create table query = %q, missing %q", query, want)
		}
	}
}

func TestEnsureSchemaCreatesRoomsTableAndIndex(t *testing.T) {
	db, mock := newMockPostgresDBWithMock(t)
	defer func() {
		_ = db.Close()
	}()

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS "rooms"`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE INDEX IF NOT EXISTS "idx_rooms_session_id" ON "rooms" \("session_id"\)`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := ensureSchema(context.Background(), db); err != nil {
		t.Fatalf("ensureSchema() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations error = %v", err)
	}
}

func newMockPostgresDB(t *testing.T) *bun.DB {
	t.Helper()

	db, _ := newMockPostgresDBWithMock(t)
	return db
}

func newMockPostgresDBWithMock(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqldb.Close()
	})
	return bun.NewDB(sqldb, pgdialect.New()), mock
}
