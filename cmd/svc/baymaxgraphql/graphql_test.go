package main

import (
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/conc"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	thmock "github.com/sprucehealth/backend/svc/threading/mock"
	"golang.org/x/net/context"
)

type gql struct {
	dirC *dirmock.Client
	thC  *thmock.Client
	svc  *service
}

func newGQL(t *testing.T) *gql {
	var g gql
	g.dirC = dirmock.New(t)
	g.thC = thmock.New(t)
	g.svc = &service{
		// auth      auth.AuthClient
		directory: g.dirC,
		threading: g.thC,
		// exComms   excomms.ExCommsClient
	}
	return &g
}

func (g *gql) query(ctx context.Context, query string, vars map[string]interface{}) *graphql.Result {
	result := conc.NewMap()
	return graphql.Do(graphql.Params{
		Schema:         gqlSchema,
		RequestString:  query,
		VariableValues: vars,
		Context:        ctx,
		RootObject: map[string]interface{}{
			"service": g.svc,
			"result":  result,
		},
	})
}

func (g *gql) finish() {
	g.dirC.Finish()
	g.thC.Finish()
}
