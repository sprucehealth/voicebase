package dbutil

import "github.com/go-sql-driver/mysql"

// MySQL error codes
const (
	MySQLDuplicateEntry      = "1062"
	MySQLNoRangeOptimization = "3170" // Memory capacity of N bytes for 'range_optimizer_max_mem_size' exceeded. Range optimization was not done for this query.
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
