package main

import (
	"testing"

	"github.com/sprucehealth/backend/libs/conc"
	authmock "github.com/sprucehealth/backend/svc/auth/mock"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	excmock "github.com/sprucehealth/backend/svc/excomms/mock"
	invitemock "github.com/sprucehealth/backend/svc/invite/mock"
	settingsmock "github.com/sprucehealth/backend/svc/settings/mock"
	thmock "github.com/sprucehealth/backend/svc/threading/mock"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

type gql struct {
	authC     *authmock.Client
	dirC      *dirmock.Client
	exC       *excmock.Client
	inviteC   *invitemock.Client
	settingsC *settingsmock.Client
	thC       *thmock.Client
	svc       *service
}

func newGQL(t *testing.T) *gql {
	var g gql
	g.authC = authmock.New(t)
	g.dirC = dirmock.New(t)
	g.exC = excmock.New(t)
	g.inviteC = invitemock.New(t)
	g.settingsC = settingsmock.New(t)
	g.thC = thmock.New(t)
	g.svc = &service{
		auth:      g.authC,
		directory: g.dirC,
		exComms:   g.exC,
		invite:    g.inviteC,
		settings:  g.settingsC,
		threading: g.thC,
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
	g.authC.Finish()
	g.dirC.Finish()
	g.exC.Finish()
	g.inviteC.Finish()
	g.settingsC.Finish()
	g.thC.Finish()
}
