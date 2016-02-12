package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestPostMessage(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	threadID := "t1"
	itemID := "ti1"
	orgID := "o1"
	entID := "e1"
	extEntID := "e2"
	entPhoneNumber := "+1-555-555-1234"
	g.thC.Expect(mock.NewExpectation(g.thC.Thread, &threading.ThreadRequest{
		ThreadID: threadID,
	}).WithReturns(&threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:              threadID,
			OrganizationID:  orgID,
			PrimaryEntityID: extEntID,
		},
	}, nil))
	// Looking up the account's entity for the org
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   entID,
				Type: directory.EntityType_INTERNAL,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{ID: orgID, Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))
	// Looking up the primary entity on the thread
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: extEntID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   extEntID,
				Type: directory.EntityType_EXTERNAL,
				Info: &directory.EntityInfo{
					DisplayName: "Barro",
				},
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       entPhoneNumber,
					},
				},
			},
		},
	}, nil))
	// Posting the message
	now := uint64(123456789)
	g.thC.Expect(mock.NewExpectation(g.thC.Thread, &threading.PostMessageRequest{
		ThreadID:     threadID,
		UUID:         "abc",
		FromEntityID: entID,
		Source: &threading.Endpoint{
			Channel: threading.Endpoint_APP,
			ID:      entID,
		},
		Destinations: []*threading.Endpoint{
			{
				Channel: threading.Endpoint_SMS,
				ID:      entPhoneNumber,
			},
		},
		Text:    "foo",
		Title:   `SMS`,
		Summary: `Schmee: foo`,
	}).WithReturns(&threading.PostMessageResponse{
		Thread: &threading.Thread{
			ID:                   threadID,
			OrganizationID:       orgID,
			PrimaryEntityID:      extEntID,
			LastMessageTimestamp: now,
			LastMessageSummary:   "Schmee: foo",
		},
		Item: &threading.ThreadItem{
			ID:            itemID,
			Timestamp:     now,
			ActorEntityID: entID,
			Internal:      false,
			Type:          threading.ThreadItem_MESSAGE,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text:   "foo",
					Status: threading.Message_NORMAL,
					Source: &threading.Endpoint{
						Channel: threading.Endpoint_APP,
						ID:      entID,
					},
					Destinations: []*threading.Endpoint{
						{
							Channel: threading.Endpoint_SMS,
							ID:      entPhoneNumber,
						},
					},
					Title:   `SMS`,
					Summary: `Schmee: foo`,
					TextRefs: []*threading.Reference{
						{Type: threading.Reference_ENTITY, ID: entID},
						{Type: threading.Reference_ENTITY, ID: extEntID},
					},
				},
			},
		},
	}, nil))
	res := g.query(ctx, `
		mutation _ ($threadID: ID!) {
			postMessage(input: {
				clientMutationId: "a1b2c3",
				threadID: $threadID,
				msg: {
					uuid: "abc"
					text: "foo"
					destinations: [{
						channel: SMS
						id: "`+entPhoneNumber+`"
					}]
					internal: false
				}
			}) {
				clientMutationId
				success
				itemEdge {
					cursor
					node {
						id
						uuid
						actor {
							id
						}
						internal
						timestamp
						data {
							__typename
							... on Message {
								summaryMarkup
								textMarkup
							}
						}
					}
				}
				thread {
					id
					lastMessageTimestamp
					title
					subtitle
					allowInternalMessages
				}
			}
		}`, map[string]interface{}{
		"threadID": threadID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"postMessage": {
			"clientMutationId": "a1b2c3",
			"itemEdge": {
				"cursor": "ti1",
				"node": {
					"actor": {
						"id": "e1"
					},
					"data": {
						"__typename": "Message",
						"summaryMarkup": "SMS",
						"textMarkup": "foo"
					},
					"id": "ti1",
					"internal": false,
					"timestamp": 123456789,
					"uuid": "abc"
				}
			},
			"success": true,
			"thread": {
				"allowInternalMessages": true,
				"id": "t1",
				"lastMessageTimestamp": 123456789,
				"subtitle": "Schmee: foo",
				"title": "Barro"
			}
		}
	}
}`, string(b))
}

func TestPostMessageDestinationNotContactOfPrimary(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	threadID := "t1"
	orgID := "o1"
	entID := "e1"
	extEntID := "e2"
	entPhoneNumber := "+1-555-555-1234"
	g.thC.Expect(mock.NewExpectation(g.thC.Thread, &threading.ThreadRequest{
		ThreadID: threadID,
	}).WithReturns(&threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:              threadID,
			OrganizationID:  orgID,
			PrimaryEntityID: extEntID,
		},
	}, nil))
	// Looking up the account's entity for the org
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   entID,
				Type: directory.EntityType_INTERNAL,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{ID: orgID, Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))
	// Looking up the primary entity on the thread
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: extEntID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   extEntID,
				Type: directory.EntityType_EXTERNAL,
				Info: &directory.EntityInfo{
					DisplayName: "Barro",
				},
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       entPhoneNumber,
					},
				},
			},
		},
	}, nil))
	res := g.query(ctx, `
		mutation _ ($threadID: ID!) {
			postMessage(input: {
				clientMutationId: "a1b2c3",
				threadID: $threadID,
				msg: {
					uuid: "abc"
					text: "foo"
					destinations: [{
						channel: SMS
						id: "`+entPhoneNumber+`WRONG"
					}]
					internal: false
				}
			}) {
				clientMutationId
				itemEdge {
					cursor
					node {
						id
						uuid
						actor {
							id
						}
						internal
						timestamp
						data {
							__typename
							... on Message {
								textMarkup
								summaryMarkup
							}
						}
					}
				}
				thread {
					id
					lastMessageTimestamp
					title
					subtitle
				}
			}
		}`, map[string]interface{}{
		"threadID": threadID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": null,
	"errors": [
		{
			"message": "The provided destination contact info does not belong to the primary entity for this thread: \"SMS\", \"+1-555-555-1234WRONG\"",
			"locations": []
		}
	]
}`, string(b))
}
