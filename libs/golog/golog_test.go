package golog

import (
	"testing"
)

type Entry struct {
	LogType string
	Level   Level
	Msg     []byte
}

type TestOutput struct {
	Entries []*Entry
}

func (o *TestOutput) Log(logType string, l Level, msg []byte) error {
	o.Entries = append(o.Entries, &Entry{
		LogType: logType,
		Level:   l,
		Msg:     msg,
	})
	return nil
}

func TestFileLine(t *testing.T) {
	out := &TestOutput{}
	SetOutput(out)

	Errorf("FOO")

	if len(out.Entries) != 1 {
		t.Fatalf("Expected 1 entry instead of %d", len(out.Entries))
	}
	ent := out.Entries[0]
	if ent.Level != ERR {
		t.Fatalf("Expected level ERR instead of %s", ent.Level)
	}
	// TODO: make this more robust as it will fail if the line number changes
	if string(ent.Msg) != `{"@message":"FOO","source_file":"golog/golog_test.go:30"}` {
		t.Fatalf("Invalid message: %s", ent.Msg)
	}
}
