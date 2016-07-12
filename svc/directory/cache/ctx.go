package cache

import "context"

type ctxKey string

const (
	// CtxEntities maps to the entity cache
	CtxEntities ctxKey = "DirectoryCtxEntities"
)

// InitEntityCache initializes the context with the needed machanisms for entity cacheing
func InitEntityCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, CtxEntities, NewEntityGroupCache(nil))
}

// WithEntities attaches an entity cache to the provided context to be used for the life of the request
func WithEntities(ctx context.Context, entities *EntityGroupCache) context.Context {
	return context.WithValue(ctx, CtxEntities, entities)
}

// Entities returns the mapping of between key and entities from the provided context
func Entities(ctx context.Context) *EntityGroupCache {
	ec, _ := ctx.Value(CtxEntities).(*EntityGroupCache)
	if ec == nil {
		return nil
	}
	return ec
}
