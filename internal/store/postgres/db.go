package postgres

import (
	"context"
	"database/sql"
)

// DBTX is the interface that abstracts database operations for both database connections and transactions.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Queries provides access to raw SQL queries.
type Queries struct {
	db DBTX
}

// NewQueries creates a new Queries instance wrapping a DBTX.
func NewQueries(db DBTX) *Queries {
	return &Queries{db: db}
}

// WithTx returns a Queries instance bounded to the given transaction DBTX.
func (q *Queries) WithTx(tx DBTX) *Queries {
	return &Queries{
		db: tx,
	}
}
