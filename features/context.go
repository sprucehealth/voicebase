package features

import (
	"golang.org/x/net/context"
)

type ctxKeyType int

const (
	ctxKey ctxKeyType = 1
)

// CtxSet returns the feature set from the context
func CtxSet(ctx context.Context) Set {
	s := ctx.Value(ctxKey)
	if s == nil {
		return nullSet{}
	}
	return s.(Set)
}

// CtxWithSet returns a new context with the feature set
func CtxWithSet(parent context.Context, s Set) context.Context {
	return context.WithValue(parent, ctxKey, s)
}
