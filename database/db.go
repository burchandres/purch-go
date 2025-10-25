package database

import (
	"database/sql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var db *bun.DB

func Init(connString string) error {
	// Create a pgdriver connector
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(connString)))

	// Configure connection pool (adjust these to match your needs)
	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(25)

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
