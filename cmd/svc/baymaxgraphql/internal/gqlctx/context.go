package gqlctx

import (
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

type ctxKey int

const (
	ctxAccount ctxKey = iota
	ctxAccountEntities
	ctxEntities
	ctxClientEncryptionKey
	ctxQuery
	ctxAuthToken
)

// Clone created a new Background context and copies all relevent baymax values from the parent into the new context
func Clone(pCtx context.Context) context.Context {
	cCtx := context.Background()
	cCtx = devicectx.WithSpruceHeaders(pCtx, devicectx.SpruceHeaders(pCtx))
	cCtx = httputil.CtxWithRequestID(pCtx, httputil.RequestID(pCtx))
	cCtx = WithAccount(cCtx, Account(pCtx))
	cCtx = WithClientEncryptionKey(cCtx, ClientEncryptionKey(pCtx))
	cCtx = WithAccountEntities(cCtx, AccountEntities(pCtx))
	cCtx = WithEntities(cCtx, Entities(pCtx))
	return cCtx
}

// WithQuery attaches the query string to the context
func WithQuery(ctx context.Context, query string) context.Context {
	return context.WithValue(ctx, ctxQuery, query)
}

// Query returns the query string for the request
func Query(ctx context.Context) string {
	query, _ := ctx.Value(ctxQuery).(string)
	return query
}

// WithAuthToken attaches the provided auth token onto a copy of the provided context
func WithAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, ctxAuthToken, token)
}

// AuthToken returns the auth token which may be empty
func AuthToken(ctx context.Context) string {
	token, _ := ctx.Value(ctxAuthToken).(string)
	return token
}

// WithAccount attaches the provided account onto a copy of the provided context
func WithAccount(ctx context.Context, acc *auth.Account) context.Context {
	// Never set a nil account so that we can update it in place. It's kind
	// of gross, but can't think of a better way to deal with authenticate
	// needing to update the account at the moment. Ideally the GraphQL pkg would
	// have a way to update context as it went through the executor.. but alas..
	if acc == nil {
		acc = &auth.Account{}
	}
	return context.WithValue(ctx, ctxAccount, acc)
}

// InPlaceWithAccount attaches the provided account onto the provided context
func InPlaceWithAccount(ctx context.Context, acc *auth.Account) {
	if acc == nil {
		acc = &auth.Account{}
	}
	*ctx.Value(ctxAccount).(*auth.Account) = *acc
}

// Account returns the account from the context which may be nil
func Account(ctx context.Context) *auth.Account {
	acc, _ := ctx.Value(ctxAccount).(*auth.Account)
	if acc != nil && acc.ID == "" {
		return nil
	}
	return acc
}

// WithAccountEntities attaches a map of orgID (intent) to entity for all of the account's entities onto the provided context
func WithAccountEntities(ctx context.Context, entitiesByOrg *EntityCache) context.Context {
	return context.WithValue(ctx, ctxAccountEntities, entitiesByOrg)
}

// AccountEntities returns the mapping of between orgs and account entities from the provided context
func AccountEntities(ctx context.Context) *EntityCache {
	ec, _ := ctx.Value(ctxAccountEntities).(*EntityCache)
	if ec == nil {
		return nil
	}
	return ec
}

// WithClientEncryptionKey attaches the provided account onto a copy of the provided context
func WithClientEncryptionKey(ctx context.Context, clientEncryptionKey string) context.Context {
	// The client encryption key is generated at token validation check time, so we store it here to make it available to concerned parties
	return context.WithValue(ctx, ctxClientEncryptionKey, clientEncryptionKey)
}

// ClientEncryptionKey returns the clientEncryptionKey from the context which may be the empty string
func ClientEncryptionKey(ctx context.Context) string {
	cek, _ := ctx.Value(ctxClientEncryptionKey).(string)
	return cek
}

// WithEntities attaches an entity cache to the provided context to be used for the life of the request
func WithEntities(ctx context.Context, entitiesByID *EntityCache) context.Context {
	return context.WithValue(ctx, ctxEntities, entitiesByID)
}

// Entities returns the mapping of between entity id and entities from the provided context
func Entities(ctx context.Context) *EntityCache {
	ec, _ := ctx.Value(ctxEntities).(*EntityCache)
	if ec == nil {
		return nil
	}
	return ec
}
