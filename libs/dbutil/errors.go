package dbutil

import "github.com/go-sql-driver/mysql"

// MySQL error codes
const (
	MySQLDuplicateEntry = "1062"
)

// IsMySQLWarning returns true if the err represents a MySQL warning of the provided code
func IsMySQLWarning(err error, code string) bool {
	warns, ok := err.(mysql.MySQLWarnings)
	return ok && len(warns) == 1 && warns[0].Code == code
}
