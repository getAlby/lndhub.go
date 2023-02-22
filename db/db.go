package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
)

func Open(config *service.Config) (*bun.DB, error) {
	var db *bun.DB
	dsn := config.DatabaseUri
	switch {
	case strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") || strings.HasPrefix(dsn, "unix://"):
		var dbConn *sql.DB
		//if Datadog is configured, send sql traces there
		if config.DatadogAgentUrl != "" {
			sqltrace.Register("postgres", pgdriver.Driver{}, sqltrace.WithServiceName("lndhub.go"))
			dbConn = sqltrace.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		} else {
			dbConn = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		}
		db = bun.NewDB(dbConn, pgdialect.New())
		db.SetMaxOpenConns(config.DatabaseMaxConns)
		db.SetMaxIdleConns(config.DatabaseMaxIdleConns)
		db.SetConnMaxLifetime(time.Duration(config.DatabaseConnMaxLifetime) * time.Second)
	default:
		return nil, fmt.Errorf("Invalid database connection string %s, only (postgres|postgresql|unix):// is supported", dsn)
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
