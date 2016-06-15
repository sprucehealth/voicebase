package golog

import (
	"bytes"
	"reflect"
	"testing"
)

type testSyslogWriter struct {
	crit    []string
	debug   []string
	err     []string
	info    []string
	warning []string
}

func (w *testSyslogWriter) Crit(m string) (err error)    { w.crit = append(w.crit, m); return nil }
func (w *testSyslogWriter) Debug(m string) (err error)   { w.debug = append(w.debug, m); return nil }
func (w *testSyslogWriter) Err(m string) (err error)     { w.err = append(w.err, m); return nil }
func (w *testSyslogWriter) Info(m string) (err error)    { w.info = append(w.info, m); return nil }
func (w *testSyslogWriter) Warning(m string) (err error) { w.warning = append(w.warning, m); return nil }

func TestWriterHandler(t *testing.T) {
	b := &bytes.Buffer{}
	fmtr := LogfmtFormatter()
	h := WriterHandler(b, fmtr)
	if err := h.Log(&Entry{Lvl: ERR, Msg: "Test"}); err != nil {
		t.Fatal(err)
	}
	if acc, exp := b.String(), "t=0001-01-01T00:00:00+0000 lvl=ERR msg=Test\n"; acc != exp {
		t.Fatalf("Expected '%s', got '%s'", exp, acc)
	}
}

func TestSyslogHandler(t *testing.T) {
	w := &testSyslogWriter{}
	fmtr := LogfmtFormatter()
	h := &syslogHandler{w: w, fmtr: fmtr}
	if err := h.Log(&Entry{Lvl: CRIT, Msg: "critical"}); err != nil {
		t.Fatal(err)
	}
	if err := h.Log(&Entry{Lvl: ERR, Msg: "error"}); err != nil {
		t.Fatal(err)
	}
	if err := h.Log(&Entry{Lvl: WARN, Msg: "warning"}); err != nil {
		t.Fatal(err)
	}
	if err := h.Log(&Entry{Lvl: INFO, Msg: "info"}); err != nil {
		t.Fatal(err)
	}
	if err := h.Log(&Entry{Lvl: DEBUG, Msg: "debug"}); err != nil {
		t.Fatal(err)
	}
	if err := h.Log(&Entry{Lvl: -1, Msg: "bad level"}); err != nil {
		t.Fatal(err)
	}
	if acc, exp := w.crit, []string{"t=0001-01-01T00:00:00+0000 lvl=CRIT msg=critical\n"}; !reflect.DeepEqual(acc, exp) {
		t.Fatalf("Expected %v, got %v", exp, acc)
	}
	if acc, exp := w.err, []string{"t=0001-01-01T00:00:00+0000 lvl=ERR msg=error\n"}; !reflect.DeepEqual(acc, exp) {
		t.Fatalf("Expected %v, got %v", exp, acc)
	}
	if acc, exp := w.warning, []string{"t=0001-01-01T00:00:00+0000 lvl=WARN msg=warning\n"}; !reflect.DeepEqual(acc, exp) {
		t.Fatalf("Expected %v, got %v", exp, acc)
	}
	if acc, exp := w.info, []string{"t=0001-01-01T00:00:00+0000 lvl=INFO msg=info\n"}; !reflect.DeepEqual(acc, exp) {
		t.Fatalf("Expected %v, got %v", exp, acc)
	}
	if acc, exp := w.debug, []string{"t=0001-01-01T00:00:00+0000 lvl=DEBUG msg=debug\n", "t=0001-01-01T00:00:00+0000 lvl=-1 msg=\"bad level\"\n"}; !reflect.DeepEqual(acc, exp) {
		t.Fatalf("Expected %v, got %v", exp, acc)
	}
}

func TestSplitHandler(t *testing.T) {
	l := ""
	low := HandlerFunc(func(e *Entry) error { l = "low"; return nil })
	high := HandlerFunc(func(e *Entry) error { l = "high"; return nil })
	h := SplitHandler(WARN, low, high)
	tcs := []struct {
		lvl Level
		exp string
	}{
		{DEBUG, "low"},
		{INFO, "low"},
		{WARN, "high"},
		{ERR, "high"},
		{CRIT, "high"},
	}
	for _, tc := range tcs {
		l = ""
		if err := h.Log(&Entry{Lvl: tc.lvl}); err != nil {
			t.Fatal(err)
		}
		if l != tc.exp {
			t.Fatalf("Expected %s for level %s, got %s", tc.exp, tc.lvl, l)
		}
	}
}
