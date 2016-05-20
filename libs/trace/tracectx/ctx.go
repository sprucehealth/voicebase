package tracectx

import "golang.org/x/net/context"

type ctxKey string

const (
	// CtxRequestID maps to the request id attached to this context
	CtxRequestID ctxKey = "TraceCtxRequestID"
)

// WithRequestID attaches the provided request id onto a copy of the provided context
func WithRequestID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, CtxRequestID, id)
}

// RequestID returns the request id which may be empty
func RequestID(ctx context.Context) uint64 {
	id, _ := ctx.Value(CtxRequestID).(uint64)
	return id
}
