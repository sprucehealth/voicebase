package trace

import (
	"testing"

	"context"
)

func TestTraceContext(t *testing.T) {
	ctx := context.Background()
	tr, ok := FromContext(ctx)
	if ok {
		t.Error("Missing trace in context should return !ok")
	} else if tr != nil {
		t.Error("Missing trace should be nil")
	}

	tr = New("family", "name", 0, 0)
	ctx = TraceContext(ctx, tr)
	tr2, ok := FromContext(ctx)
	if !ok {
		t.Error("Trace missing")
	} else if tr2 == nil {
		t.Error("Trace is nil")
	} else if tr2.TraceID() != tr.TraceID() || tr2.SpanID() != tr.SpanID() {
		t.Error("Trace or span ID mismatch")
	}
}
