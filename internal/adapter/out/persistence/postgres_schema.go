package persistence

import (
	"context"
	"fmt"

	"github.com/kyh0703/portfoilo-media/internal/adapter/out/repository/model"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"persistence",
	fx.Provide(NewDB),
	fx.Invoke(EnsurePostgresSchema),
)

func EnsurePostgresSchema(db *bun.DB) error {
	return ensureSchema(context.Background(), db)
}

func ensureSchema(ctx context.Context, db *bun.DB) error {
	if _, err := createRoomsTableQuery(db).Exec(ctx); err != nil {
		return fmt.Errorf("create rooms table: %w", err)
	}
	if err := ensureRoomsRuntimeEventColumns(ctx, db); err != nil {
		return err
	}

	if _, err := createRoomsSessionIndexQuery(db).Exec(ctx); err != nil {
		return fmt.Errorf("create rooms session index: %w", err)
	}

	return nil
}

func ensureRoomsRuntimeEventColumns(ctx context.Context, db *bun.DB) error {
	if _, err := db.ExecContext(ctx, `ALTER TABLE "rooms" ADD COLUMN IF NOT EXISTS "last_runtime_event_type" text`); err != nil {
		return fmt.Errorf("add rooms last_runtime_event_type column: %w", err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE "rooms" ADD COLUMN IF NOT EXISTS "last_runtime_event_at" timestamptz`); err != nil {
		return fmt.Errorf("add rooms last_runtime_event_at column: %w", err)
	}
	return nil
}

func createRoomsTableQuery(db *bun.DB) *bun.CreateTableQuery {
	return db.NewCreateTable().
		Model((*model.Room)(nil)).
		IfNotExists()
}

func createRoomsSessionIndexQuery(db *bun.DB) *bun.CreateIndexQuery {
	return db.NewCreateIndex().
		Model((*model.Room)(nil)).
		Index("idx_rooms_session_id").
		Column("session_id").
		IfNotExists()
}
