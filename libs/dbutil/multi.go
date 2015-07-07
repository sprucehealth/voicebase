package dbutil

import (
	"strings"
)

// MultiInsert implementations provide a way to generate multiple sets
// of values when generating an INSERT query.
type MultiInsert interface {
	Append(vals ...interface{})
	IsEmpty() bool
	NumColumns() int
	Query() string
	Values() []interface{}
}

type multiInsert struct {
	nCols       int
	vals        []interface{}
	rowEstimate int
}

type mySQLMultiInsert struct {
	multiInsert
}

type postgresMultiInsert struct {
	multiInsert
	startIndex int
}

// MySQLMultiInsert provides a MySQL specific implementation of MultiInsert. The number
// of rows is an adviser and is not required.
func MySQLMultiInsert(rowEstimate int) MultiInsert {
	return &mySQLMultiInsert{
		multiInsert: multiInsert{rowEstimate: rowEstimate},
	}
}

// PostgresMultiInsert provides a Postgres specific implementation of MultiInsert.
func PostgresMultiInsert(rowEstimate, startIndex int) MultiInsert {
	return &postgresMultiInsert{
		multiInsert: multiInsert{rowEstimate: rowEstimate},
		startIndex:  startIndex,
	}
}

func (mi *multiInsert) Append(vals ...interface{}) {
	if len(vals) == 0 {
		panic("dbutil.multiInsert.Append: must provide at least one column")
	}
	if len(mi.vals) == 0 {
		if mi.vals == nil {
			mi.vals = make([]interface{}, 0, len(vals)*mi.rowEstimate)
		}
		mi.nCols = len(vals)
	} else {
		if mi.nCols != len(vals) {
			panic("dbutil.multiInsert.Append: column count changed between calls")
		}
	}
	mi.vals = append(mi.vals, vals...)
}

func (mi *multiInsert) NumColumns() int {
	return mi.nCols
}

func (mi *multiInsert) Values() []interface{} {
	return mi.vals
}

func (mi *multiInsert) IsEmpty() bool {
	return len(mi.vals) == 0
}

func (mi *mySQLMultiInsert) Query() string {
	if len(mi.vals) == 0 {
		return ""
	}

	rows := len(mi.vals) / mi.nCols

	// TODO: optimize for allocs
	r := "(" + MySQLArgs(mi.nCols) + ")"
	reps := make([]string, rows)
	for i := range reps {
		reps[i] = r
	}
	return strings.Join(reps, ",")
}

func (mi *postgresMultiInsert) Query() string {
	if len(mi.vals) == 0 {
		return ""
	}

	rows := len(mi.vals) / mi.nCols

	// TODO: optimize for allocs
	reps := make([]string, rows)
	for i := range reps {
		reps[i] = "(" + PostgresArgs(mi.startIndex+i*mi.nCols, mi.nCols) + ")"
	}
	return strings.Join(reps, ",")
}
