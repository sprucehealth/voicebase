package apiaccess

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/graphql"
)

// Authenticated wraps a graphql Resolve function and asserts that the caller is authenticated
func Authenticated(f func(p graphql.ResolveParams) (interface{}, error),
) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}
		return f(p)
	}
}

// Provider wraps a graphql Resolve function and asserts that the caller is a provider
func Provider(f func(p graphql.ResolveParams) (interface{}, error),
) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc.Type != auth.AccountType_PROVIDER {
			return nil, errors.ErrNotAuthorized(ctx, "PROVIDER_ONLY_API")
		}
		return f(p)
	}
}

// Patient wraps a graphql Resolve function and asserts that the caller is a patient
func Patient(f func(p graphql.ResolveParams) (interface{}, error),
) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc.Type != auth.AccountType_PATIENT {
			return nil, errors.ErrNotAuthorized(ctx, "PATIENT_ONLY_API")
		}
		return f(p)
	}
}
