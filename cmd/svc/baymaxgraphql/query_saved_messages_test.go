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

func TestSavedMessagesQuery(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

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
				ID:   "ent",
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   "org",
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SavedMessages, &threading.SavedMessagesRequest{
		By: &threading.SavedMessagesRequest_EntityIDs{
			EntityIDs: &threading.IDList{IDs: []string{"ent", "org"}},
		},
	}).WithReturns(&threading.SavedMessagesResponse{
		SavedMessages: []*threading.SavedMessage{
			{
				ID:            "sm_1",
				Title:         "foo",
				Internal:      true,
				OwnerEntityID: "org",
				Content: &threading.SavedMessage_Message{
					Message: &threading.Message{
						Text: "one",
					},
				},
			},
			{
				ID:            "sm_2",
				Title:         "bar",
				OwnerEntityID: "ent",
				Content: &threading.SavedMessage_Message{
					Message: &threading.Message{
						Text: "two",
					},
				},
			},
		},
	}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		query _ {
			savedMessages(organizationID: "org") {
				title
				messages {
					id
					title
					shared
					threadItem {
						id
						internal
						data {
							... on Message {
								textMarkup
							}
						}
					}
				}
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"savedMessages": [{
				"title": "Your Saved Messages",
				"messages": [
					{
						"id": "sm_2",
						"title": "bar",
						"shared": false,
						"threadItem": {
							"id": "sm_2",
							"internal": false,
							"data": {
								"textMarkup": "two"
							}
						}
					}
				]
			}, {
				"title": "Team Saved Message",
				"messages": [
					{
						"id": "sm_1",
						"title": "foo",
						"shared": true,
						"threadItem": {
							"id": "sm_1",
							"internal": true,
							"data": {
								"textMarkup": "one"
							}
						}
					}
				]
			}]
		}
	}`, res)
}
