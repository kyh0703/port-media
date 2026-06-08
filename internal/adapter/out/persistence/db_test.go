package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestNewDBCreatesParentDirectoryForSQLiteFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data", "media.sqlite")
	db := NewDB(&configs.Config{
		Database: configs.DatabaseConfig{
			URL: "file:" + dbPath + "?cache=shared",
		},
	})
	defer func() {
		_ = db.Close()
	}()

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("sqlite file stat error = %v", err)
	}
}

func TestConfigureSQLiteEnablesWALForFileDatabase(t *testing.T) {
	sqldb := newTestSQLDBForConnection(t, "file:"+filepath.Join(t.TempDir(), "media.db")+"?cache=shared")
	defer func() {
		_ = sqldb.Close()
	}()

	if err := configureSQLite(context.Background(), sqldb, 5*time.Second); err != nil {
		t.Fatalf("configureSQLite() error = %v", err)
	}

	var journalMode string
	if err := sqldb.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode error = %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	var busyTimeout int
	if err := sqldb.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout error = %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busyTimeout)
	}
}

func newTestSQLDBForConnection(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	sqldb, err := sql.Open(sqliteshim.ShimName, dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	return sqldb
}
