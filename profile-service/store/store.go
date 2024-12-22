package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sing3demons/profile-service/logger"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource already exists")
	QueryTimeoutDuration = time.Second * 5
)

type Users interface {
	GetByID(context.Context, string, logger.DetailLog, logger.SummaryLog) (*User, error)
	GetByEmail(context.Context, string, logger.DetailLog, logger.SummaryLog) (*User, error)
	Create(context.Context, *sql.Tx, *User, logger.DetailLog, logger.SummaryLog) error
	Delete(context.Context, string, logger.DetailLog, logger.SummaryLog) error
}

type Storer struct {
	Users Users
}

func withTx(db *sql.DB, ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func NewStorer(db *sql.DB) *Storer {
	return &Storer{
		Users: &UserStore{db: db},
	}
}
