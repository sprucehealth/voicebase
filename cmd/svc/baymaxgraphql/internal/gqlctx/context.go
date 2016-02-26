package gqlctx

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/device"
	"golang.org/x/net/context"
)

type ctxKey int

const (
	ctxAccount ctxKey = iota
	ctxSpruceHeaders
	ctxClientEncryptionKey
	ctxRequestID
	ctxAuthToken
)

// WithSpruceHeaders attaches the provided spruce headers onto a copy of the provided context
func WithSpruceHeaders(ctx context.Context, sh *device.SpruceHeaders) context.Context {
	return context.WithValue(ctx, ctxSpruceHeaders, sh)
}

// SpruceHeaders returns the spruce headers which may be nil
func SpruceHeaders(ctx context.Context) *device.SpruceHeaders {
	sh, _ := ctx.Value(ctxSpruceHeaders).(*device.SpruceHeaders)
	if sh == nil {
		return &device.SpruceHeaders{}
	}
	return sh
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

// WithRequestID attaches the provided request id onto a copy of the provided context
func WithRequestID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, ctxRequestID, id)
}

// RequestID returns the request id which may be empty
func RequestID(ctx context.Context) uint64 {
	id, _ := ctx.Value(ctxRequestID).(uint64)
	return id
}

// WithAccount attaches the provided account onto a copy of the provided context
func WithAccount(ctx context.Context, acc *models.Account) context.Context {
	// Never set a nil account so that we can update it in place. It's kind
	// of gross, but can't think of a better way to deal with authenticate
	// needing to update the account at the moment. Ideally the GraphQL pkg would
	// have a way to update context as it went through the executor.. but alas..
	if acc == nil {
		acc = &models.Account{}
	}
	return context.WithValue(ctx, ctxAccount, acc)
}

// InPlaceWithAccount attaches the provided account onto the provided context
func InPlaceWithAccount(ctx context.Context, acc *models.Account) {
	if acc == nil {
		acc = &models.Account{}
	}
	*ctx.Value(ctxAccount).(*models.Account) = *acc
}

// Account returns the account from the context which may be nil
func Account(ctx context.Context) *models.Account {
	acc, _ := ctx.Value(ctxAccount).(*models.Account)
	if acc != nil && acc.ID == "" {
		return nil
	}
	return acc
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
