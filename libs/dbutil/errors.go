package dbutil

import "github.com/go-sql-driver/mysql"

// MySQL error codes
const (
	MySQLDuplicateEntry = "1062"
)

// IsMySQLWarning returns true if the err represents a MySQL warning of the provided code
func IsMySQLWarning(err error, code string) bool {
	warns, ok := err.(mysql.MySQLWarnings)
	if !ok {
		return false
	}
	for _, w := range warns {
		if w.Code != code {
			return false
		}
	}
	return true
}
