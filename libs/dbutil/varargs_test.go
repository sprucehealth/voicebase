package dbutil

import (
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func ExampleMySQLVarArgs() {
	args := MySQLVarArgs()
	args.Append("name", "joe")
	args.Append("age", 62)
	fmt.Println(args.Columns())
	fmt.Printf("%#v\n", args.Values())
	// Output:
	// name=?,age=?
	// []interface {}{"joe", 62}
}

func ExamplePostgresVarArgs() {
	args := PostgresVarArgs(3)
	args.Append("name", "joe")
	args.Append("age", 62)
	fmt.Println(args.Columns())
	fmt.Printf("%#v\n", args.Values())
	// Output:
	// name=$3,age=$4
	// []interface {}{"joe", 62}
}

func TestMySQLVarArgs(t *testing.T) {
	args := MySQLVarArgs()
	test.Equals(t, true, args.IsEmpty())
	test.Equals(t, "", args.Columns())
	test.Equals(t, 0, len(args.Values()))

	args.Append("col1", 123)
	test.Equals(t, false, args.IsEmpty())
	vals := args.Values()
	test.Equals(t, "col1=?", args.Columns())
	test.Equals(t, 1, len(vals))
	test.Equals(t, 123, vals[0])

	args.Append("col2", "foo")
	vals = args.Values()
	test.Equals(t, "col1=?,col2=?", args.Columns())
	test.Equals(t, 2, len(vals))
	test.Equals(t, 123, vals[0])
	test.Equals(t, "foo", vals[1])
}

func TestPostgresVarArgs(t *testing.T) {
	args := PostgresVarArgs(4)
	test.Equals(t, true, args.IsEmpty())
	test.Equals(t, "", args.Columns())
	test.Equals(t, 0, len(args.Values()))

	args.Append("col1", 123)
	test.Equals(t, false, args.IsEmpty())
	vals := args.Values()
	test.Equals(t, "col1=$4", args.Columns())
	test.Equals(t, 1, len(vals))
	test.Equals(t, 123, vals[0])

	args.Append("col2", "foo")
	vals = args.Values()
	test.Equals(t, "col1=$4,col2=$5", args.Columns())
	test.Equals(t, 2, len(vals))
	test.Equals(t, 123, vals[0])
	test.Equals(t, "foo", vals[1])
}

func BenchmarkMySQLVarArgs(b *testing.B) {
	b.ReportAllocs()
	args := MySQLVarArgs()
	args.Append("col1", 123)
	args.Append("col2", "foo")
	args.Append("col3", 1.23)
	for i := 0; i < b.N; i++ {
		_ = args.Columns()
	}
}

func BenchmarkPostgresVarArgs(b *testing.B) {
	b.ReportAllocs()
	args := PostgresVarArgs(1)
	args.Append("col1", 123)
	args.Append("col2", "foo")
	args.Append("col3", 1.23)
	for i := 0; i < b.N; i++ {
		_ = args.Columns()
	}
}
