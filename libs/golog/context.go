package golog

import "context"

type ctxKey int

const (
	ctxLogger ctxKey = iota
)

// WithLogger attaches the logger into to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, ctxLogger, logger)
}

// ContextLogger returns the logger associated with the provided context
func ContextLogger(ctx context.Context) Logger {
	logger, _ := ctx.Value(ctxLogger).(Logger)
	if logger == nil {
		logger = Default()
	}
	return logger
}
