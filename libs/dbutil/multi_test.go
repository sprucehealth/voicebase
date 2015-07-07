package dbutil

import (
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func ExampleMySQLMultiInsert() {
	args := MySQLMultiInsert(0)
	args.Append("joe", 88)
	args.Append("sue", 77)
	fmt.Println(args.Query())
	fmt.Printf("%#v\n", args.Values())
	// Output:
	// (?,?),(?,?)
	// []interface {}{"joe", 88, "sue", 77}
}

func ExamplePostgresMultiInsert() {
	args := PostgresMultiInsert(0, 7)
	args.Append("joe", 88)
	args.Append("sue", 77)
	fmt.Println(args.Query())
	fmt.Printf("%#v\n", args.Values())
	// Output:
	// ($7,$8),($9,$10)
	// []interface {}{"joe", 88, "sue", 77}
}

func TestMySQLMultiInsert(t *testing.T) {
	insert := MySQLMultiInsert(0)
	test.Equals(t, true, insert.IsEmpty())
	test.Equals(t, 0, insert.NumColumns())
	test.Equals(t, "", insert.Query())
	test.Equals(t, 0, len(insert.Values()))

	insert.Append("test", 123)
	test.Equals(t, false, insert.IsEmpty())
	test.Equals(t, 2, insert.NumColumns())
	test.Equals(t, "(?,?)", insert.Query())
	test.Equals(t, 2, len(insert.Values()))
	test.Equals(t, "test", insert.Values()[0])
	test.Equals(t, 123, insert.Values()[1])

	insert.Append("foo", 444)
	test.Equals(t, false, insert.IsEmpty())
	test.Equals(t, 2, insert.NumColumns())
	test.Equals(t, "(?,?),(?,?)", insert.Query())
	test.Equals(t, 4, len(insert.Values()))
	test.Equals(t, "foo", insert.Values()[2])
	test.Equals(t, 444, insert.Values()[3])
}

func TestPostgresMultiInsert(t *testing.T) {
	insert := PostgresMultiInsert(0, 3)
	test.Equals(t, true, insert.IsEmpty())
	test.Equals(t, 0, insert.NumColumns())
	test.Equals(t, "", insert.Query())
	test.Equals(t, 0, len(insert.Values()))

	insert.Append("test", 123)
	test.Equals(t, false, insert.IsEmpty())
	test.Equals(t, 2, insert.NumColumns())
	test.Equals(t, "($3,$4)", insert.Query())
	test.Equals(t, 2, len(insert.Values()))
	test.Equals(t, "test", insert.Values()[0])
	test.Equals(t, 123, insert.Values()[1])

	insert.Append("foo", 444)
	test.Equals(t, false, insert.IsEmpty())
	test.Equals(t, 2, insert.NumColumns())
	test.Equals(t, "($3,$4),($5,$6)", insert.Query())
	test.Equals(t, 4, len(insert.Values()))
	test.Equals(t, "foo", insert.Values()[2])
	test.Equals(t, 444, insert.Values()[3])
}
