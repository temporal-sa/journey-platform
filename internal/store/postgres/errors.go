package postgres

import "errors"

var (
	// ErrNotFound is returned when a requested record is not found.
	ErrNotFound = errors.New("record not found")

	// ErrAlreadyExists is returned when inserting a record that violates a primary key or unique constraint.
	ErrAlreadyExists = errors.New("record already exists")

	// ErrConflict is returned on unique constraint boundary violations.
	ErrConflict = errors.New("unique constraint violation conflict")

	// ErrTxClosed is returned when operating on a closed transaction.
	ErrTxClosed = errors.New("transaction already closed")
)
