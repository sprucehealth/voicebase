package dbutil

import "strconv"

// MYSQLArgs returns n mysql argumetns for a database query.
func MySQLArgs(n int) string {
	if n == 0 {
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
func PostgresArgs(n int) string {
	if n == 0 {
		return ""
	}

	var result string
	for i := 0; i < n; i++ {
		result += "$" + strconv.Itoa(i+1)
		if i < n-1 {
			result += ","
		}
	}
	return result
}
