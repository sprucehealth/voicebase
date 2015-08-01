package golog

import (
	"bytes"
	"testing"
	"time"
)

func TestLongFormFormatter(t *testing.T) {
	fmt := LongFormFormatter()
	cases := []struct {
		e *Entry
		m []byte
	}{
		{&Entry{}, []byte("Jan  1 00:00:00 [CRIT] \n")},
		{
			&Entry{
				Time: time.Date(2015, 1, 2, 3, 4, 5, 0, time.UTC),
				Lvl:  INFO,
				Msg:  "message",
				Src:  "file:123",
				Ctx: []interface{}{
					"foo", "bar\nooo",
					"int", 123,
				},
			},
			[]byte("Jan  2 03:04:05 [INFO] file:123 message\nfoo: bar\nooo\nint: 123\n"),
		},
	}
	for _, tc := range cases {
		m := fmt.Format(tc.e)
		if !bytes.Equal(m, tc.m) {
			t.Errorf("fmt(%+v) == %s, expected %s", tc.e, quoteASCII(string(m)), quoteASCII(string(tc.m)))
		}
	}
}
