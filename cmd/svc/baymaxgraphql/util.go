package main

import (
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/sprucehealth/backend/apiservice"
	"golang.org/x/net/context"
)

type ctxKey int

const (
	ctxAccount       ctxKey = 0
	ctxSpruceHeaders ctxKey = 1
)

func ctxWithSpruceHeaders(ctx context.Context, sh *apiservice.SpruceHeaders) context.Context {
	return context.WithValue(ctx, ctxSpruceHeaders, sh)
}

// spruceHeadersFromContext returns the spruce headers which may be nil
func spruceHeadersFromContext(ctx context.Context) *apiservice.SpruceHeaders {
	sh, _ := ctx.Value(ctxSpruceHeaders).(*apiservice.SpruceHeaders)
	return sh
}

func ctxWithAccount(ctx context.Context, acc *account) context.Context {
	// Never set a nil account so that we can update it in place. It's kind
	// of gross, but can't think of a better way to deal with authenticate
	// needing to update the account at the moment. Ideally the GraphQL pkg would
	// have a way to update context as it went through the executor.. but alas..
	if acc == nil {
		acc = &account{}
	}
	return context.WithValue(ctx, ctxAccount, acc)
}

// accountFromContext returns the account from the context which may be nil
func accountFromContext(ctx context.Context) *account {
	acc, _ := ctx.Value(ctxAccount).(*account)
	if acc != nil && acc.ID == "" {
		return nil
	}
	return acc
}

func serviceFromParams(p graphql.ResolveParams) *service {
	return p.Info.RootValue.(map[string]interface{})["service"].(*service)
}

func selectedFields(p graphql.ResolveParams) []string {
	f := p.Info.FieldASTs[0]
	fields := make([]string, 0, len(f.SelectionSet.Selections))
	for _, s := range f.SelectionSet.Selections {
		if f, ok := s.(*ast.Field); ok && f.Name != nil {
			fields = append(fields, f.Name.Value)
		}
	}
	return fields
}

func selectingOnlyID(p graphql.ResolveParams) bool {
	f := p.Info.FieldASTs[0]
	if len(f.SelectionSet.Selections) > 1 {
		return false
	}
	for _, s := range f.SelectionSet.Selections {
		if f, ok := s.(*ast.Field); ok && f.Name != nil {
			if f.Name.Value == "id" {
				return true
			}
		}
	}
	return false
}

func nodePrefix(nodeID string) string {
	i := strings.IndexByte(nodeID, '_')
	prefix := nodeID[:i]

	return prefix
}
