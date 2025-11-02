package database

import (
	"database/sql"
	"runtime"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var db *bun.DB

func Init(connString string) error {
	// Create a pgdriver connector
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(connString)))

	// Configure connection pool (adjust these to match your needs)
	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	// Wrap with Bun
	db = bun.NewDB(sqldb, pgdialect.New())

	// Verify connection (equivalent to pool.Ping)
	return db.Ping()
}

func GetPool() *bun.DB {
	return db
}

func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
