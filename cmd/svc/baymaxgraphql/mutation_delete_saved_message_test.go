package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestDeleteSavedMessageMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.SavedMessages, &threading.SavedMessagesRequest{
		By: &threading.SavedMessagesRequest_IDs{
			IDs: &threading.IDList{IDs: []string{"sm_1"}},
		},
	}).WithReturns(&threading.SavedMessagesResponse{
		SavedMessages: []*threading.SavedMessage{
			{
				ID:             "sm_1",
				Title:          "foo",
				Internal:       true,
				OwnerEntityID:  "org",
				OrganizationID: "org",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "account_1",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
	}).WithReturns(
		[]*directory.Entity{
			{
				ID:        "ent",
				AccountID: "account_1",
				Type:      directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   "org",
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.DeleteSavedMessage, &threading.DeleteSavedMessageRequest{
		SavedMessageID: "sm_1",
	}))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		mutation _ {
			deleteSavedMessage(input: {savedMessageID: "sm_1"}) {
				success
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"deleteSavedMessage": {
				"success": true
			}
		}
	}`, res)
}
