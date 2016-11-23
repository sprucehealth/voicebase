package dbutil

import "github.com/go-sql-driver/mysql"

// MySQL error and warning codes
var (
	// MySQLDuplicateEntry is returned for inserts that fail a unique key constraint.
	MySQLDuplicateEntry = MySQLErrorCode{s: "1062", n: 1062}
	// MySQLDeadlock means a deadlock was found when trying to get lock; try restarting transaction
	MySQLDeadlock = MySQLErrorCode{s: "1213", n: 1213}
	// MySQLNoRangeOptimization means the memory capacity of N bytes for 'range_optimizer_max_mem_size' exceeded. Range optimization was not done for this query.
	MySQLNoRangeOptimization = MySQLErrorCode{s: "3170", n: 3170}
)

// MySQLErrorCode is an error or warning code returned from MySQL
type MySQLErrorCode struct {
	s string
	n uint16
}

// IsMySQLError returns true if the err represents a MySQL error of the provided code
func IsMySQLError(err error, code MySQLErrorCode) bool {
	e, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return e.Number == code.n
}

// IsMySQLWarning returns true if the err represents a MySQL warning of the provided code
func IsMySQLWarning(err error, code MySQLErrorCode) bool {
	warns, ok := err.(mysql.MySQLWarnings)
	if !ok {
		return false
	}
	for _, w := range warns {
		if w.Code != code.s {
			return false
		}
	}
	return true
}
