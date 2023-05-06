package sqlc

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

var _ DBTX = (*wrappedDB)(nil)

func Wrap(db DBTX) DBTX {
	return &wrappedDB{db}
}

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type wrappedDB struct {
	DBTX
}

func (w *wrappedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var err error
	if b, ok := BuilderFrom(ctx); ok {
		query, args, err = b.Build(query, args...)
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not build query")
	}
	return w.DBTX.ExecContext(ctx, query, args...)
}

func (w *wrappedDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return w.DBTX.PrepareContext(ctx, query)
}

func (w *wrappedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	if b, ok := BuilderFrom(ctx); ok {
		query, args, err = b.Build(query, args...)
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not build query")
	}
	return w.DBTX.QueryContext(ctx, query, args...)
}

func (w *wrappedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	var err error
	if b, ok := BuilderFrom(ctx); ok {
		if queryNew, argsNew, err := b.Build(query, args...); err == nil {
			query = queryNew
			args = argsNew
		}
	}
	if err != nil {
		fmt.Printf("could not build query: %s", err)
	}
	return w.DBTX.QueryRowContext(ctx, query, args...)
}
