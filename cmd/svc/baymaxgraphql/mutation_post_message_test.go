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

func TestPostMessage(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &models.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	threadID := "t1"
	itemID := "ti1"
	orgID := "o1"
	entID := "e1"
	extEntID := "e2"
	entPhoneNumber := "+1-555-555-1234"
	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		ID:              threadID,
		OrganizationID:  orgID,
		PrimaryEntityID: extEntID,
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

	// Looking up the primary entity on the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entity, extEntID, []directory.EntityInformation{
		directory.EntityInformation_CONTACTS,
	}, int64(0)).WithReturns(&directory.Entity{
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
	}, nil))
	// Posting the message
	now := uint64(123456789)
	g.ra.Expect(mock.NewExpectation(g.ra.PostMessage, &threading.PostMessageRequest{
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
		Attachments: []*threading.Attachment{
			&threading.Attachment{
				Type:  threading.Attachment_IMAGE,
				Title: "title",
				URL:   "mediaID",
				Data: &threading.Attachment_Image{
					Image: &threading.ImageAttachment{
						Mimetype: "image/jpeg",
						URL:      "mediaID",
					},
				},
			},
		},
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
					attachments: [{
         				attachmentType: IMAGE
         				contentType: "image/jpeg"
         				mediaID: "mediaID"
         				title: "title"
        			}]
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
					isDeletable
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
				"isDeletable": true,
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
	acc := &models.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	threadID := "t1"
	orgID := "o1"
	entID := "e1"
	extEntID := "e2"
	entPhoneNumber := "+1-555-555-1234"
	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		ID:              threadID,
		OrganizationID:  orgID,
		PrimaryEntityID: extEntID,
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
	// Looking up the primary entity on the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entity, extEntID, []directory.EntityInformation{
		directory.EntityInformation_CONTACTS,
	}, int64(0)).WithReturns(&directory.Entity{
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
