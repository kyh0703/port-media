package persistence

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const (
	defaultDatabaseMaxOpenConn = 10
	defaultDatabaseMaxIdleConn = 5
)

func NewDB(cfg *configs.Config) *bun.DB {
	sqldb, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		panic(err)
	}

	sqldb.SetMaxOpenConns(defaultDatabaseMaxOpenConn)
	sqldb.SetMaxIdleConns(defaultDatabaseMaxIdleConn)

	db := bun.NewDB(sqldb, pgdialect.New())
	return db
}
