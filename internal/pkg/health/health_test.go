package health

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestCheckerReturnsOKWhenDependenciesAreHealthy(t *testing.T) {
	db, mock := newTestDB(t)
	defer func() {
		_ = db.Close()
	}()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))

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
	if result.Checks["postgres"].Status != StatusOK {
		t.Fatalf("postgres status = %q, want %q", result.Checks["postgres"].Status, StatusOK)
	}
	if result.Checks["redis"].Status != StatusOK {
		t.Fatalf("redis status = %q, want %q", result.Checks["redis"].Status, StatusOK)
	}
}

func TestCheckerReturnsDegradedWhenRedisFails(t *testing.T) {
	db, mock := newTestDB(t)
	defer func() {
		_ = db.Close()
	}()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))

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

func TestCheckerReturnsDegradedWhenPostgresFails(t *testing.T) {
	db, mock := newTestDB(t)
	defer func() {
		_ = db.Close()
	}()
	mock.ExpectExec("SELECT 1").WillReturnError(assertionError("postgres unavailable"))

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
	if result.Checks["postgres"].Status != StatusFailed {
		t.Fatalf("postgres status = %q, want %q", result.Checks["postgres"].Status, StatusFailed)
	}
	if result.Checks["postgres"].Error == "" {
		t.Fatal("postgres error is empty")
	}
}

func newTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations error = %v", err)
		}
	})

	return bun.NewDB(sqldb, pgdialect.New()), mock
}

type assertionError string

func (e assertionError) Error() string {
	return string(e)
}
