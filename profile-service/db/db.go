package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

func New(addr string, maxOpenConns, maxIdleConns int, maxIdleTime string) (*sql.DB, error) {
	db, err := sql.Open("postgres", addr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	duration, err := time.ParseDuration(maxIdleTime)
	if err == nil {
		db.SetConnMaxIdleTime(duration)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func BeginTx(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	return db.BeginTx(ctx, nil)
}
