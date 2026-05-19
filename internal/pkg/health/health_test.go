package health

import (
	"context"
	"database/sql"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestCheckerReturnsOKWhenDependenciesAreHealthy(t *testing.T) {
	db := newTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer redisServer.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer func() {
		_ = redisClient.Close()
	}()

	result := NewChecker(db, redisClient).Check(context.Background())

	if result.Status != StatusOK {
		t.Fatalf("status = %q, want %q", result.Status, StatusOK)
	}
	if result.Checks["sqlite"].Status != StatusOK {
		t.Fatalf("sqlite status = %q, want %q", result.Checks["sqlite"].Status, StatusOK)
	}
	if result.Checks["redis"].Status != StatusOK {
		t.Fatalf("redis status = %q, want %q", result.Checks["redis"].Status, StatusOK)
	}
}

func TestCheckerReturnsDegradedWhenRedisFails(t *testing.T) {
	db := newTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	addr := redisServer.Addr()
	redisServer.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: addr})
	defer func() {
		_ = redisClient.Close()
	}()

	result := NewChecker(db, redisClient).Check(context.Background())

	if result.Status != StatusDegraded {
		t.Fatalf("status = %q, want %q", result.Status, StatusDegraded)
	}
	if result.Checks["redis"].Status != StatusFailed {
		t.Fatalf("redis status = %q, want %q", result.Checks["redis"].Status, StatusFailed)
	}
	if result.Checks["redis"].Error == "" {
		t.Fatal("redis error is empty")
	}
}

func TestCheckerReturnsDegradedWhenSQLiteFails(t *testing.T) {
	db := newTestDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() error = %v", err)
	}

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer redisServer.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer func() {
		_ = redisClient.Close()
	}()

	result := NewChecker(db, redisClient).Check(context.Background())

	if result.Status != StatusDegraded {
		t.Fatalf("status = %q, want %q", result.Status, StatusDegraded)
	}
	if result.Checks["sqlite"].Status != StatusFailed {
		t.Fatalf("sqlite status = %q, want %q", result.Checks["sqlite"].Status, StatusFailed)
	}
	if result.Checks["sqlite"].Error == "" {
		t.Fatal("sqlite error is empty")
	}
}

func newTestDB(t *testing.T) *bun.DB {
	t.Helper()

	sqldb, err := sql.Open(sqliteshim.ShimName, "file:test-health?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	return bun.NewDB(sqldb, sqlitedialect.New())
}
