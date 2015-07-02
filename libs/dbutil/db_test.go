package dbutil

import (
	"fmt"
	"testing"
)

func ExampleMySQLArgs() {
	fmt.Println(MySQLArgs(3))
	// Output:
	// ?,?,?
}

func ExamplePostgresArgs() {
	fmt.Println(PostgresArgs(5, 3))
	// Output:
	// $5,$6,$7
}

func TestDBArgs(t *testing.T) {
	expected := "?,?,?,?,?"
	args := MySQLArgs(5)
	if expected != args {
		t.Fatalf("Expected %#v, got %#v", expected, args)
	}

	expected = "?"
	args = MySQLArgs(1)
	if expected != args {
		t.Fatalf("Expected %#v, got %#v", expected, args)
	}
}

func TestPostgresArgs(t *testing.T) {
	tests := []struct {
		si int
		n  int
		e  string
	}{
		{si: 2, n: 11, e: "$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12"},
		{si: 1, n: 1, e: "$1"},
		{si: 8, n: 2, e: "$8,$9"},
		{si: 8, n: 3, e: "$8,$9,$10"},
		{si: 9, n: 11, e: "$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19"},
	}

	for _, tc := range tests {
		args := PostgresArgs(tc.si, tc.n)
		if tc.e != args {
			t.Fatalf(`PostgresArgs(%d, %d) = "%s", expected "%s"`, tc.si, tc.n, args, tc.e)
		}
	}
}

func BenchmarkPostgresArgs1(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = PostgresArgs(1, 1)
	}
}

func BenchmarkPostgresArgs4(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = PostgresArgs(1, 8)
	}
}

func BenchmarkPostgresArgs8(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = PostgresArgs(1, 8)
	}
}

func BenchmarkPostgresArgs32(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = PostgresArgs(1, 32)
	}
}
