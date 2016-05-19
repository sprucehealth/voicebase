package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestPostMessage(t *testing.T) {
	g := newGQL(t)
	defer g.finish()
	g.svc.media = media.New(storage.NewTestStore(map[string]*storage.TestObject{
		"mediaID": &storage.TestObject{
			Headers: map[string][]string{"Content-Type": []string{"image/jpeg"}},
		},
	}), storage.NewTestStore(nil), 100, 100)

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PROVIDER,
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
		SystemTitle:     "Barro",
		Type:            threading.ThreadType_EXTERNAL,
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.CanPostMessage, threadID))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			ID:   entID,
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{ID: orgID, Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	// Looking up the primary entity on the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: extEntID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
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
				Title: "",
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
			Type:                 threading.ThreadType_EXTERNAL,
			SystemTitle:          "Barro",
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
         				mediaID: "mediaID"
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

func TestPostMessage_VisitAttachment(t *testing.T) {
	g := newGQL(t)
	defer g.finish()
	g.svc.media = media.New(storage.NewTestStore(map[string]*storage.TestObject{
		"mediaID": &storage.TestObject{
			Headers: map[string][]string{"Content-Type": []string{"image/jpeg"}},
		},
	}), storage.NewTestStore(nil), 100, 100)
	g.svc.webDomain = "test.com"

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PROVIDER,
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
		SystemTitle:     "Barro",
		Type:            threading.ThreadType_SECURE_EXTERNAL,
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.CanPostMessage, threadID))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			ID:   entID,
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{ID: orgID, Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	// Looking up the primary entity on the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: extEntID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
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
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.VisitLayout, &layout.GetVisitLayoutRequest{
		ID: "visitLayout_12345",
	}).WithReturns(&layout.GetVisitLayoutResponse{
		VisitLayout: &layout.VisitLayout{
			ID:   "visitLayout_12345",
			Name: "Sinus Infection",
			Version: &layout.VisitLayoutVersion{
				ID: "visitLayoutVersion_123456",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateVisit, &care.CreateVisitRequest{
		EntityID:        extEntID,
		LayoutVersionID: "visitLayoutVersion_123456",
		Name:            "Sinus Infection",
		OrganizationID:  orgID,
	}).WithReturns(&care.CreateVisitResponse{
		Visit: &care.Visit{
			ID:   "visit_12345",
			Name: "Sinus Infection",
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
		Text:    "foo",
		Title:   "Shared a visit:",
		Summary: `Schmee: foo`,
		Attachments: []*threading.Attachment{
			&threading.Attachment{
				Type:  threading.Attachment_VISIT,
				Title: "Sinus Infection",
				URL:   "https://test.com/thread/t1/visit/visit_12345",
				Data: &threading.Attachment_Visit{
					Visit: &threading.VisitAttachment{
						VisitID:   "visit_12345",
						VisitName: "Sinus Infection",
					},
				},
			},
		},
	}).WithReturns(&threading.PostMessageResponse{
		Thread: &threading.Thread{
			ID:                   threadID,
			OrganizationID:       orgID,
			Type:                 threading.ThreadType_SECURE_EXTERNAL,
			SystemTitle:          "Barro",
			PrimaryEntityID:      extEntID,
			MessageCount:         10,
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
					Title:   `Shared a visit:`,
					Summary: `Schmee: Shared a visit:`,
					Attachments: []*threading.Attachment{
						{
							Type:  threading.Attachment_VISIT,
							Title: "Sinus Infection",
							URL:   "https://test.com/visit/visit_12345",
							Data: &threading.Attachment_Visit{
								Visit: &threading.VisitAttachment{
									VisitID:   "visit_12345",
									VisitName: "Sinus Infection",
								},
							},
						},
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
					internal: false
					attachments: [{
         				attachmentType: VISIT
         				mediaID: "visitLayout_12345"
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
						"summaryMarkup": "Shared a visit:",
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
				"isDeletable": false,
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
	acc := &auth.Account{
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
	g.ra.Expect(mock.NewExpectation(g.ra.CanPostMessage, threadID))
	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			ID:   entID,
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{ID: orgID, Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	// Looking up the primary entity on the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: extEntID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
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

func TestPostMessagePatientSecureExternal(t *testing.T) {
	g := newGQL(t)
	defer g.finish()
	g.svc.media = media.New(storage.NewTestStore(map[string]*storage.TestObject{
		"mediaID": &storage.TestObject{
			Headers: map[string][]string{"Content-Type": []string{"image/jpeg"}},
		},
	}), storage.NewTestStore(nil), 100, 100)

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PATIENT,
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
		SystemTitle:     "Barro",
		Type:            threading.ThreadType_SECURE_EXTERNAL,
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.CanPostMessage, threadID))
	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			ID:   entID,
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{ID: orgID, Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	// Looking up the primary entity on the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: extEntID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
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
		Text:    "foo",
		Title:   ``,
		Summary: `Schmee: foo`,
		Attachments: []*threading.Attachment{
			&threading.Attachment{
				Type:  threading.Attachment_IMAGE,
				Title: "",
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
			Type:                 threading.ThreadType_SECURE_EXTERNAL,
			SystemTitle:          "Barro",
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
					Title:   ``,
					Summary: `Schmee: foo`,
					TextRefs: []*threading.Reference{
						{Type: threading.Reference_ENTITY, ID: entID},
						{Type: threading.Reference_ENTITY, ID: extEntID},
					},
				},
			},
		},
	}, nil))

	// since this is a patient thread there should be an org lookup
	// Looking up the primary entity on the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			ID:   extEntID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "OrganizationName",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			ID:   extEntID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "OrganizationName",
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
         				mediaID: "mediaID"
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
					typeIndicator
					emptyStateTextMarkup
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
						"summaryMarkup": "",
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
				"allowInternalMessages": false,
				"emptyStateTextMarkup": "Welcome to your conversation with OrganizationName.",
				"id": "t1",
				"isDeletable": false,
				"lastMessageTimestamp": 123456789,
				"subtitle": "Schmee: foo",
				"title": "OrganizationName",
				"typeIndicator": "NONE"
			}
		}
	}
}`, string(b))
}
