package api

import "database/sql"

// The db interface can be used when a method can accept either
// a *Tx or *DB.
type db interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}
