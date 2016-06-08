package mediactx

import (
	"errors"

	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

type ctxKey int

const (
	ctxAccount ctxKey = iota
	ctxRequiresAuthorization
	ctxAuthToken
)

// Clone created a new Background context and copies all relevent baymax values from the parent into the new context
func Clone(pCtx context.Context) context.Context {
	cCtx := context.Background()
	cCtx = devicectx.WithSpruceHeaders(cCtx, devicectx.SpruceHeaders(pCtx))
	cCtx = httputil.CtxWithRequestID(cCtx, httputil.RequestID(pCtx))
	acc, _ := Account(pCtx)
	cCtx = WithAccount(cCtx, acc)
	cCtx = WithAuthToken(cCtx, AuthToken(pCtx))
	return cCtx
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
	return context.WithValue(ctx, ctxAccount, acc)
}

// Account returns the account from the context which may be nil
func Account(ctx context.Context) (*auth.Account, error) {
	acc, _ := ctx.Value(ctxAccount).(*auth.Account)
	if acc == nil {
		return nil, errors.New("no account in context")
	}
	return acc, nil
}

// WithRequiresAuthorization attaches a flag to the context that determines if the call requires authorization
func WithRequiresAuthorization(ctx context.Context, requiresAuthorization bool) context.Context {
	return context.WithValue(ctx, ctxRequiresAuthorization, requiresAuthorization)
}

// RequiresAuthorization returns the auth token which may be empty
func RequiresAuthorization(ctx context.Context) bool {
	requiresAuthorization, ok := ctx.Value(ctxRequiresAuthorization).(bool)
	if ok {
		return requiresAuthorization
	}
	return true
}
