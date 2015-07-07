package dbutil

import (
	"strconv"
	"strings"
)

// MySQLArgs returns n mysql arguments for a database query.
func MySQLArgs(n int) string {
	if n <= 0 {
		return ""
	}

	result := make([]byte, 2*n-1)
	for i := 0; i < len(result)-1; i += 2 {
		result[i] = '?'
		result[i+1] = ','
	}
	result[len(result)-1] = '?'
	return string(result)
}

// PostgresArgs returns n postgres arguments for a database query.
func PostgresArgs(si, n int) string {
	if n <= 0 {
		return ""
	}
	if si < 1 {
		panic("dbutil.PostgresArgs start index must be > 0")
	}

	// Count the digits in greatest index we'll reach.
	digits := 1
	i := si + n
	for i > 10 {
		digits++
		i /= 10
	}

	res := make([]byte, 0, (digits+2)*n)
	for i := 0; i < n; i++ {
		res = append(res, '$')
		res = strconv.AppendInt(res, int64(i+si), 10)
		if i < n-1 {
			res = append(res, ',')
		}
	}
	return string(res)
}

// EscapeMySQLName escapes column, table, and index names (among others).
// TODO: Make this secure. DO NOT currently use for external (user) provided values.
func EscapeMySQLName(name string) string {
	return "`" + strings.Replace(name, "`", "``", -1) + "`"
}
