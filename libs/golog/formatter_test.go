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

func TestJSONFormatter(t *testing.T) {
	fmt := JSONFormatter(false)
	cases := []*struct {
		e *Entry
		m []byte
	}{
		{&Entry{}, []byte("{\"level\":\"CRIT\",\"msg\":\"\",\"t\":\"0001-01-01T00:00:00+0000\"}")},
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
			[]byte("{\"foo\":\"bar\\nooo\",\"int\":123,\"level\":\"INFO\",\"msg\":\"message\",\"src\":\"file:123\",\"t\":\"2015-01-02T03:04:05+0000\"}"),
		},
	}
	for _, tc := range cases {
		m := fmt.Format(tc.e)
		if !bytes.Equal(m, tc.m) {
			t.Errorf("fmt(%+v) == %s, expected %s", tc.e, quoteASCII(string(m)), quoteASCII(string(tc.m)))
		}
	}

	// With newlines

	fmt = JSONFormatter(true)
	for _, tc := range cases {
		tc.m = append(tc.m, '\n')
	}
	for _, tc := range cases {
		m := fmt.Format(tc.e)
		if !bytes.Equal(m, tc.m) {
			t.Errorf("fmt(%+v) == %s, expected %s", tc.e, quoteASCII(string(m)), quoteASCII(string(tc.m)))
		}
	}
}

func TestTerminalFormatter(t *testing.T) {
	fmt := TerminalFormatter()
	cases := []*struct {
		e *Entry
		m []byte
	}{
		{&Entry{}, []byte("\x1b[0;35m[00:00:00] [CRIT] \x1b[0m\n")},
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
			[]byte("\x1b[0;32m[03:04:05] [INFO] file:123 foo=bar\\nooo int=123 message\x1b[0m\n"),
		},
	}
	for _, tc := range cases {
		m := fmt.Format(tc.e)
		if !bytes.Equal(m, tc.m) {
			t.Errorf("fmt(%+v) == %s, expected %s", tc.e, quoteASCII(string(m)), quoteASCII(string(tc.m)))
		}
	}
}

func TestFormatContext(t *testing.T) {
	acc := string(FormatContext([]interface{}{
		"str", "testing",
		"int", 123,
		"bool", true,
		"float64", 0.123,
		"int64", int64(999),
		340, "xx",
	}, '|'))
	exp := "str=testing|int=123|bool=true|float64=0.123|int64=999|_error=xx"
	if acc != exp {
		t.Errorf("expected %s, got %s", quoteASCII(exp), quoteASCII(acc))
	}
}
