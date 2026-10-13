package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"time"
	"uptime_app/backend/internal/config"
)

func Open(ctx context.Context, c config.Config) (*bun.DB, error) {
	connector, err := newConnector(c.DatabaseURL)
	if err != nil {
		return nil, err
	}
	sqldb := sql.OpenDB(connector)
	sqldb.SetMaxOpenConns(c.MaxOpen)
	sqldb.SetMaxIdleConns(c.MaxIdle)
	sqldb.SetConnMaxLifetime(c.ConnLifetime)
	db := bun.NewDB(sqldb, pgdialect.New())
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database connection unavailable")
	}
	return db, nil
}

// WithDSN panics on malformed driver options. Convert only constructor panics
// into a configuration error without exposing a DSN containing credentials.
func newConnector(dsn string) (connector *pgdriver.Connector, err error) {
	defer func() {
		if recover() != nil {
			connector = nil
			err = fmt.Errorf("invalid DATABASE_URL driver options")
		}
	}()
	return pgdriver.NewConnector(pgdriver.WithDSN(dsn)), nil
}
