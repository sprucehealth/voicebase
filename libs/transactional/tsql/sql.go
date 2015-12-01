package tsql

import (
	"database/sql"
	"errors"
)

// DB defines the sql.DB methods that are on both the DB and TX structs
type DB interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Begin() (Tx, error)
}

// AsDB wraps a sql.DB struct to conform the the tsql interfaces
func AsDB(s *sql.DB) DB {
	return &db{s}
}

type db struct {
	*sql.DB
}

func (db *db) Begin() (Tx, error) {
	t, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &tx{t}, nil
}

// Why does this exist?
// Inorder to use the new transaction related functinality we need to assumed that there are other secionts of code calling .Begin()
// We return an instance of this struct that behaves as a normal transaction but noops on all nested operations
type tx struct {
	*sql.Tx
}

// Begin returns a reference to this transactions self
func (tx *tx) Begin() (Tx, error) {
	return nil, errors.New("Cannot call Begin() on an existing transaction")
}

// Tx defines the transaction interface
type Tx interface {
	DB
	Rollback() error
	Commit() error
}

// SafeTx is a transaction that cannot be rolled back or commited and Begin returns a reference to itself
type safeTx struct {
	Tx
}

// AsSafeTx converts a Tx into a safeTx
func AsSafeTx(t Tx) Tx {
	return &safeTx{t}
}

// Begin returns a reference to this transactions self
func (tx *safeTx) Begin() (Tx, error) {
	return tx, nil
}

// Rollback noops
func (tx *safeTx) Rollback() error {
	return nil
}

// Commit noops
func (tx *safeTx) Commit() error {
	return nil
}
