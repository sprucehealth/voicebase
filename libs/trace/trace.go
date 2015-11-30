package trace

import (
	"sync"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"golang.org/x/net/context"
)

var tracePool = sync.Pool{New: func() interface{} { return &Trace{} }}

type ctxKey struct{}

type Trace struct {
	family  string
	name    string
	traceID uint64
	spanID  uint64
}

// New returns a new trace with the provided traceID and spanID. If either ID
// is 0 then a new ID is generated instead.
func New(family, name string, traceID, spanID uint64) *Trace {
	if traceID == 0 {
		var err error
		traceID, err = idgen.NewID()
		if err != nil {
			golog.Errorf("Failed to generate new trace ID: %s", err)
		}
	}
	if spanID == 0 {
		var err error
		spanID, err = idgen.NewID()
		if err != nil {
			golog.Errorf("Failed to generate new trace ID: %s", err)
		}
	}
	tr := tracePool.Get().(*Trace)
	*tr = Trace{
		family:  family,
		name:    name,
		traceID: traceID,
		spanID:  spanID,
	}
	return tr
}

// Family returns the family of the name
func (t *Trace) Family() string {
	return t.family
}

// Name returns the name of the trace (e.g. function name)
func (t *Trace) Name() string {
	return t.name
}

// TraceID returns the trace ID
func (t *Trace) TraceID() uint64 {
	return t.traceID
}

// SpanID returns the span ID
func (t *Trace) SpanID() uint64 {
	return t.spanID
}

// Finish declares that this trace is complete.
// The trace should not be used after calling this method.
func (t *Trace) Finish() {
	tracePool.Put(t)
	// TODO
}

func (t *Trace) LazyPrintf(fmt string, v ...interface{}) {
	// TODP
}

// FromContext returns the trace value from the context if it exists and
// a boolean to represent if the trace was found in the context.
func FromContext(ctx context.Context) (*Trace, bool) {
	t, ok := ctx.Value(ctxKey{}).(*Trace)
	return t, ok
}

// TraceContext returns a new context with the provided trace as a value.
func TraceContext(parent context.Context, t *Trace) context.Context {
	return context.WithValue(parent, ctxKey{}, t)
}
