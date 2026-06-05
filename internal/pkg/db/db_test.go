package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestConfigureSQLiteEnablesWALForFileDatabase(t *testing.T) {
	sqldb := newTestSQLDB(t, "file:"+filepath.Join(t.TempDir(), "media.db")+"?cache=shared")
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

func newTestSQLDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	sqldb, err := sql.Open(sqliteshim.ShimName, dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	return sqldb
}
