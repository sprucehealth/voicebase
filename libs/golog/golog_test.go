package golog

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type TestHandler struct {
	Entries []*Entry
	mu      sync.Mutex
}

func (o *TestHandler) Log(e *Entry) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	// Make sure not to retain the provided Entry
	e2 := *e
	o.Entries = append(o.Entries, &e2)
	return nil
}

type NullHandler struct{}

func (NullHandler) Log(e *Entry) error { return nil }

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

	l.LogDepthf(0, ERR, "FOO")
	if !strings.HasPrefix(out.Entries[0].Src, "golog/golog_test.go") {
		t.Fatalf("Expected current function depth. Got '%s'", out.Entries[0].Src)
	}
	out.Entries = nil

	LogDepthf(0, ERR, "BAR")
	if !strings.HasPrefix(out.Entries[0].Src, "golog/golog_test.go") {
		t.Fatalf("Expected current function depth. Got '%s'", out.Entries[0].Src)
	}
	out.Entries = nil

	Errorf("BAR")
	if !strings.HasPrefix(out.Entries[0].Src, "golog/golog_test.go") {
		t.Fatalf("Expected current function depth. Got '%s'", out.Entries[0].Src)
	}
}

func TestWriter(t *testing.T) {
	out := &TestHandler{}
	Default().SetHandler(out)

	Writer.Write([]byte("testing"))
	if out.Entries[0].Lvl != INFO {
		t.Errorf("Expected default writer to use level INFO by default")
	}
	out.Entries = nil

	Writer.Write([]byte("[ERR] testing"))
	if out.Entries[0].Lvl != ERR {
		t.Errorf("Expected default writer to use level ERR when log has prefix [ERR]")
	}
}

func TestCaller(t *testing.T) {
	s := Caller(0)
	if !strings.HasPrefix(s, "golog/golog_test.go:") {
		t.Errorf("Unexpected caller %s", s)
	}
}

func TestStats(t *testing.T) {
	l := newLogger(nil, NullHandler{}, INFO, nil)
	var s Stats
	l.readStats(&s)
	if s.Info != 0 {
		t.Fatalf("Expected 0 info entries, got %d", s.Info)
	}
	l.Infof("")
	l.readStats(&s)
	if s.Info != 1 {
		t.Fatalf("Expected 1 info entry, got %d", s.Info)
	}

	// Make sure derived loggers are accounted for

	l2 := l.Context()
	l2.Infof("")
	l.readStats(&s)
	if s.Info != 2 {
		t.Fatalf("Expected 2 info entries, got %d", s.Info)
	}
}

func TestParallelContext(t *testing.T) {
	var l Logger
	h := &TestHandler{}
	l = newLogger(nil, h, INFO, nil)
	ctx := append(make([]interface{}, 0, 4), "a", "b")
	l = l.Context(ctx...)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		l.Context("c", "d").Infof("Foo")
		wg.Done()
	}()
	go func() {
		l.Context("e", "f").Infof("Bar")
		wg.Done()
	}()
	wg.Wait()
	if len(h.Entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(h.Entries))
	}
	sort.Sort(entriesByMsg(h.Entries))
	if h.Entries[0].Msg != "Bar" {
		t.Errorf("Expected 'Bar' got '%s'", h.Entries[0].Msg)
	}
	if !reflect.DeepEqual(h.Entries[0].Ctx, []interface{}{"a", "b", "e", "f"}) {
		t.Errorf(`Expected []interface{}{"a", "b", "e", "f"} got %v`, h.Entries[0].Ctx)
	}
	if h.Entries[1].Msg != "Foo" {
		t.Errorf("Expected 'Foo' got '%s'", h.Entries[0].Msg)
	}
	if !reflect.DeepEqual(h.Entries[1].Ctx, []interface{}{"a", "b", "c", "d"}) {
		t.Errorf(`Expected []interface{}{"a", "b", "c", "d"} got %v`, h.Entries[0].Ctx)
	}
}

func BenchmarkLogInfo(b *testing.B) {
	l := newLogger(nil, NullHandler{}, INFO, nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Infof("XXX")
	}
}

func BenchmarkLogError(b *testing.B) {
	l := newLogger(nil, NullHandler{}, INFO, nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Errorf("XXX")
	}
}

type entriesByMsg []*Entry

func (e entriesByMsg) Len() int           { return len(e) }
func (e entriesByMsg) Swap(a, b int)      { e[a], e[b] = e[b], e[a] }
func (e entriesByMsg) Less(a, b int) bool { return e[a].Msg < e[b].Msg }
