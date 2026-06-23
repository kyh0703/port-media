package persistence

import (
	"testing"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/uptrace/bun/dialect"
)

func TestNewDBUsesPostgresDialect(t *testing.T) {
	db := NewDB(&configs.Config{
		Database: configs.DatabaseConfig{
			URL: "postgres://postgres:postgres@localhost:5432/portfoilo_media?sslmode=disable",
		},
	})
	defer func() {
		_ = db.Close()
	}()

	if db.Dialect().Name() != dialect.PG {
		t.Fatalf("dialect = %s, want %s", db.Dialect().Name(), dialect.PG)
	}
}
