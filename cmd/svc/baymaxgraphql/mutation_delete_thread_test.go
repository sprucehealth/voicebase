package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestDeleteThread(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &models.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	threadID := "t1"
	orgID := "o1"
	entID := "e1"

	// Fetch thread
	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		ID:             threadID,
		OrganizationID: orgID,
	}, nil))

	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.EntityForAccountID, orgID, acc.ID).WithReturns(
		&directory.Entity{
			ID:   entID,
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{ID: orgID, Type: directory.EntityType_ORGANIZATION},
			},
		}, nil))

	// Delete thread
	g.ra.Expect(mock.NewExpectation(g.ra.DeleteThread, threadID, entID).WithReturns(nil))

	res := g.query(ctx, `
		mutation _ ($threadID: ID!) {
			deleteThread(input: {
				clientMutationId: "a1b2c3",
				threadID: $threadID,
			}) {
				clientMutationId
			}
		}`, map[string]interface{}{
		"threadID": threadID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"deleteThread": {
			"clientMutationId": "a1b2c3"
		}
	}
}`, string(b))
}
