package gqlctx

import (
	"context"
	"sync"
)

// FeatureKey represents an available feature of an organization or account
type FeatureKey int

const (
	// VideoCalling enables video calling for an organization
	VideoCalling FeatureKey = iota
)

type lazyFeature struct {
	fn func(ctx context.Context) bool
	v  bool
	o  sync.Once
}

// WithLazyFeature returns a new context with a feature that is lazily evaluated
func WithLazyFeature(ctx context.Context, ft FeatureKey, fn func(ctx context.Context) bool) context.Context {
	return context.WithValue(ctx, ft, &lazyFeature{fn: fn})
}

// WithFeature returns a new context with a feature set to the provided value
func WithFeature(ctx context.Context, ft FeatureKey, v bool) context.Context {
	return context.WithValue(ctx, ft, v)
}

// FeatureEnabled returns true iff a feature is enabled
func FeatureEnabled(ctx context.Context, ft FeatureKey) bool {
	f := ctx.Value(ft)
	switch f := f.(type) {
	case nil:
		return false
	case bool:
		return f
	case *lazyFeature:
		f.o.Do(func() {
			f.v = f.fn(ctx)
		})
		return f.v
	}
	panic("Unknown type for feature")
}
