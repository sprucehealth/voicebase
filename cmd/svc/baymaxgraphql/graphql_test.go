package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	"github.com/sprucehealth/backend/libs/conc"
	invitemock "github.com/sprucehealth/backend/svc/invite/mock"
	notificationmock "github.com/sprucehealth/backend/svc/notification/mock"
	settingsmock "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

type gql struct {
	inviteC       *invitemock.Client
	settingsC     *settingsmock.Client
	notificationC *notificationmock.Client
	svc           *service
	ra            *ramock.ResourceAccessor
}

func newGQL(t *testing.T) *gql {
	t.Parallel()
	var g gql
	g.inviteC = invitemock.New(t)
	g.settingsC = settingsmock.New(t)
	g.notificationC = notificationmock.New(t)
	g.ra = ramock.New(t)
	g.svc = &service{
		invite:       g.inviteC,
		settings:     g.settingsC,
		notification: g.notificationC,
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
			"service":        g.svc,
			"result":         result,
			raccess.ParamKey: g.ra,
		},
	})
}

func (g *gql) finish() {
	g.inviteC.Finish()
	g.settingsC.Finish()
	g.notificationC.Finish()
	g.ra.Finish()
}
