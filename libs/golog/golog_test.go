package golog

import (
	"strings"
	"testing"
	"time"
)

type TestHandler struct {
	Entries []*Entry
}

func (o *TestHandler) Log(e *Entry) error {
	o.Entries = append(o.Entries, e)
	return nil
}

func TestBasic(t *testing.T) {
	out := &TestHandler{}

	l := Default()
	l.SetHandler(out)
	l.Context("id", 123).Errorf("FOO")

	if len(out.Entries) != 1 {
		t.Fatalf("Expected 1 entry instead of %d", len(out.Entries))
	}
	ent := out.Entries[0]
	if ent.Lvl != ERR {
		t.Fatalf("Expected level ERR instead of %s", ent.Lvl)
	}
	if want := "FOO"; ent.Msg != want {
		t.Fatalf("Got '%s'. Expected '%s'", ent.Msg, want)
	}
	if len(ent.Ctx) != 2 {
		t.Fatalf("Got context of %d. Expeceted %d", len(ent.Ctx), 2)
	}
}

func TestLogfmtFormatter(t *testing.T) {
	e := &Entry{
		Time: time.Time{},
		Msg:  "msg",
		Lvl:  INFO,
		Ctx:  []interface{}{"num", 123, "str", `needs quotes`, "str2", "noquotes"},
		Src:  "golog_test.go:123",
	}
	fmtr := LogfmtFormatter()
	b := fmtr.Format(e)
	if s, want := string(b), `t=0001-01-01T00:00:00+0000 lvl=INFO msg=msg src=golog_test.go:123 num=123 str="needs quotes" str2=noquotes`+"\n"; want != s {
		t.Fatalf("Got '%s'. Expected '%s'", s, want)
	}
}

func TestStackTraceDepth(t *testing.T) {
	out := &TestHandler{}

	l := Default()
	l.SetHandler(out)

	l.Errorf("FOO")
	if !strings.HasPrefix(out.Entries[0].Src, "golog/golog_test.go") {
		t.Fatalf("Expected current function depth. Got '%s'", out.Entries[0].Src)
	}
	out.Entries = nil

	l.Logf(0, ERR, "FOO")
	if !strings.HasPrefix(out.Entries[0].Src, "golog/golog_test.go") {
		t.Fatalf("Expected current function depth. Got '%s'", out.Entries[0].Src)
	}
	out.Entries = nil

	Logf(0, ERR, "BAR")
	if !strings.HasPrefix(out.Entries[0].Src, "golog/golog_test.go") {
		t.Fatalf("Expected current function depth. Got '%s'", out.Entries[0].Src)
	}
	out.Entries = nil

	Errorf("BAR")
	if !strings.HasPrefix(out.Entries[0].Src, "golog/golog_test.go") {
		t.Fatalf("Expected current function depth. Got '%s'", out.Entries[0].Src)
	}
}
