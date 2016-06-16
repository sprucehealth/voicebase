package dronboard

import "testing"

func TestParseAmount(t *testing.T) {
	cases := []struct {
		Input string
		Cents int
	}{
		{"12", 12},
		{".09", 9},
		{"1.31", 131},
		{"$0.01", 1},
		{"$.99", 99},
		{"  0.11 ", 11},
		{"$ 1.23", 123},
		{"1.", 100},
		{"02", 2},
	}
	for _, c := range cases {
		if x, err := parseAmount(c.Input); err != nil {
			t.Errorf("parseAmount(%s) error %s", c.Input, err.Error())
		} else if x != c.Cents {
			t.Errorf("parseAmount(%s) == %d. Expected %d", c.Input, x, c.Cents)
		}
	}
}
