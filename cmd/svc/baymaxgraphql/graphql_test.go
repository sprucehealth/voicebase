package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	invitemock "github.com/sprucehealth/backend/svc/invite/mock"
	layoutmock "github.com/sprucehealth/backend/svc/layout/mock"
	notificationmock "github.com/sprucehealth/backend/svc/notification/mock"
	settingsmock "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

type gql struct {
	inviteC       *invitemock.Client
	settingsC     *settingsmock.Client
	layoutC       *layoutmock.Client
	notificationC *notificationmock.Client
	svc           *service
	ra            *ramock.ResourceAccessor
	layoutStore   *layoutmock.Store
}

func newGQL(t *testing.T) *gql {
	t.Parallel()
	var g gql
	g.inviteC = invitemock.New(t)
	g.settingsC = settingsmock.New(t)
	g.notificationC = notificationmock.New(t)
	g.layoutC = layoutmock.New(t)
	g.ra = ramock.New(t)
	g.layoutStore = layoutmock.NewStore(t)
	g.svc = &service{
		invite:       g.inviteC,
		settings:     g.settingsC,
		notification: g.notificationC,
		spruceOrgID:  "spruce_org",
		segmentio:    &segmentIOWrapper{},
		media:        media.New(storage.NewTestStore(nil), storage.NewTestStore(nil), 100, 100),
		sns:          &awsutil.SNS{},
		layout:       g.layoutC,
		layoutStore:  g.layoutStore,
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
			"remoteAddr":     "127.0.0.1",
			"userAgent":      "test",
			raccess.ParamKey: g.ra,
		},
	})
}

func (g *gql) finish() {
	g.inviteC.Finish()
	g.settingsC.Finish()
	g.notificationC.Finish()
	g.ra.Finish()
	g.layoutC.Finish()
	g.layoutStore.Finish()
}

func responseEquals(t *testing.T, expected string, actual interface{}) {
	// Roundtrip response to normalize into basic types
	b, err := json.Marshal(actual)
	test.OK(t, err)
	var act interface{}
	test.OK(t, json.Unmarshal(b, &act))
	var exp interface{}
	test.OK(t, json.Unmarshal([]byte(expected), &exp))
	test.Equals(t, exp, act)
}

func expectEntityInOrgForAccountID(ra *ramock.ResourceAccessor, accountID string, results []*directory.Entity) {
	ra.Expect(mock.NewExpectation(ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(
		results, nil))
}
