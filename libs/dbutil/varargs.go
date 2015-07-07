package dbutil

import (
	"strconv"
)

// VarArgs implementations provide a way to generate lists of columns
// and values when generating a query. For instance if you want to set
// a number of columns during UPDATE you can use VarArgs to collect
// a list of column names and corresponding values.
type VarArgs interface {
	// Append adds a column and corresponding values to the list
	Append(column string, value interface{})
	// Columns returns a query snippet that includes the columns and placeholders
	// separated by commas.
	Columns() string
	// Values returns an interface list with the values that were given to Append.
	Values() []interface{}
	// IsEmpty returns true iff no values have been appended.
	IsEmpty() bool
}

type varArgs struct {
	cols []string
	vals []interface{}
}

type mySQLVarArgs struct {
	varArgs
}

type postgresVarArgs struct {
	varArgs
	startIndex int
}

// MySQLVarArgs provides a MySQL specific implementation of VarArgs
func MySQLVarArgs() VarArgs {
	return &mySQLVarArgs{}
}

// PostgresVarArgs provides a Postgres specific implementation of VarArgs.
// Placeholder numbers start at the provided index.
func PostgresVarArgs(startIndex int) VarArgs {
	if startIndex < 0 {
		panic("dbutil.PostgresVarArgs: Postgres index must start at 1")
	}
	return &postgresVarArgs{startIndex: startIndex}
}

func (v *varArgs) Append(column string, value interface{}) {
	v.cols = append(v.cols, column)
	v.vals = append(v.vals, value)
}

func (v *varArgs) Values() []interface{} {
	return v.vals
}

func (v *varArgs) IsEmpty() bool {
	return len(v.vals) == 0
}

func (v *mySQLVarArgs) Columns() string {
	if len(v.cols) == 0 {
		return ""
	}
	colLen := len(v.cols) - 1 // account for separating commas
	for _, c := range v.cols {
		colLen += len(c) + 2 // "column=?"
	}
	buf := make([]byte, 0, colLen)
	for i, c := range v.cols {
		if i != 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, c...)
		buf = append(buf, '=', '?')
	}
	return string(buf)
}

func (v *postgresVarArgs) Columns() string {
	if len(v.cols) == 0 {
		return ""
	}

	// Count the digits in greatest index we'll reach.
	digits := 1
	i := v.startIndex + len(v.cols)
	for i > 10 {
		digits++
		i /= 10
	}

	colLen := len(v.cols) - 1 // account for separating commas
	for _, c := range v.cols {
		colLen += len(c) + digits + 2 // "column=$123"
	}
	buf := make([]byte, 0, colLen)
	index := v.startIndex
	for i, c := range v.cols {
		if i != 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, c...)
		buf = append(buf, '=', '$')
		buf = strconv.AppendInt(buf, int64(index+i), 10)
	}
	return string(buf)
}
