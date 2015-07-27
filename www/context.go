package www

import (
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/common"
)

type contextKey int

const (
	ckAccount contextKey = iota
	ckPermissions
)

// MustCtxAccount returns the account from the context or panics if it's missing or wrong type.
func MustCtxAccount(ctx context.Context) *common.Account {
	return ctx.Value(ckAccount).(*common.Account)
}

// MustCtxPermissions returns the account permissions from the context or panics if it's missing or wrong type.
func MustCtxPermissions(ctx context.Context) Permissions {
	return ctx.Value(ckPermissions).(Permissions)
}

// CtxAccount returns the account from the context and true. Otherwise it returns false.
func CtxAccount(ctx context.Context) (*common.Account, bool) {
	v := ctx.Value(ckAccount)
	if v == nil {
		return nil, false
	}
	return v.(*common.Account), true
}

// CtxPermissions returns the account permissions from the context and true. Otherwise it returns false.
func CtxPermissions(ctx context.Context) (Permissions, bool) {
	v := ctx.Value(ckPermissions)
	if v == nil {
		return nil, false
	}
	return v.(Permissions), true
}

// CtxWithAccount returns a new context that includes the account
func CtxWithAccount(ctx context.Context, a *common.Account) context.Context {
	return context.WithValue(ctx, ckAccount, a)
}

// CtxWithPermissions returns a new context that includes the account
func CtxWithPermissions(ctx context.Context, p Permissions) context.Context {
	return context.WithValue(ctx, ckPermissions, p)
}
