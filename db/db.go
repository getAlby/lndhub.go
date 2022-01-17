package db

import (
	"database/sql"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

func Open(dsn string) (*bun.DB, error) {
	var db *bun.DB
	switch {
	case strings.HasPrefix(dsn, "postgres"):
		dbConn := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db = bun.NewDB(dbConn, pgdialect.New())
	default:
		dbConn, err := sql.Open(sqliteshim.ShimName, dsn)
		if err != nil {
			return nil, err
		}
		db = bun.NewDB(dbConn, sqlitedialect.New())
	}

	db.AddQueryHook(bundebug.NewQueryHook(
		// disable the hook
		bundebug.WithEnabled(false),
		// BUNDEBUG=1 logs failed queries
		// BUNDEBUG=2 logs all queries
		bundebug.FromEnv("BUNDEBUG"),
	))

	return db, nil
}
