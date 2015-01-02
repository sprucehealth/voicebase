package dbutil

import "testing"

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

	expected = "$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11"
	args = PostgresArgs(11)
	if expected != args {
		t.Fatalf("Expected %#v, got %#v", expected, args)
	}

	expected = "$1"
	args = PostgresArgs(1)
	if expected != args {
		t.Fatalf("Expected %#v, got %#v", expected, args)
	}

}
