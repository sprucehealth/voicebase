package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestScheduleMessage(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	threadID := "t1"
	orgID := "o1"
	entID := "e1"
	extEntID := "e2"
	entPhoneNumber := "+1-555-555-1234"
	scheduleFor := time.Now().Add(1 * time.Hour).Unix()

	g.ra.Expect(mock.NewExpectation(g.ra.AssertIsEntity, entID).WithReturns(
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

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		ID:              threadID,
		OrganizationID:  orgID,
		PrimaryEntityID: extEntID,
		SystemTitle:     "Barro",
		Type:            threading.THREAD_TYPE_EXTERNAL,
	}, nil))

	// Looking up the primary entity on the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: extEntID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
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
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.MediaInfo, "mediaID").WithReturns(&media.MediaInfo{
		ID:  "mediaID",
		URL: "URL",
		MIME: &media.MIME{
			Type:    "image",
			Subtype: "png",
		},
	}, nil))

	// Posting the message
	g.ra.Expect(mock.NewExpectation(g.ra.CreateScheduledMessage, &threading.CreateScheduledMessageRequest{
		ThreadID:      threadID,
		ActorEntityID: entID,
		ScheduledFor:  uint64(scheduleFor),
		Content: &threading.CreateScheduledMessageRequest_Message{
			Message: &threading.MessagePost{
				Source: &threading.Endpoint{
					Channel: threading.ENDPOINT_CHANNEL_APP,
					ID:      entID,
				},
				Destinations: []*threading.Endpoint{
					{
						Channel: threading.ENDPOINT_CHANNEL_SMS,
						ID:      entPhoneNumber,
					},
				},
				Text:    "foo",
				Title:   `SMS`,
				Summary: `Schmee: foo`,
				Attachments: []*threading.Attachment{
					{
						ContentID: "mediaID",
						Title:     "",
						URL:       "mediaID",
						Data: &threading.Attachment_Image{
							Image: &threading.ImageAttachment{
								Mimetype: "image/png",
								MediaID:  "mediaID",
							},
						},
					},
				},
			},
		},
	}))

	g.ra.Expect(mock.NewExpectation(g.ra.ScheduledMessages, &threading.ScheduledMessagesRequest{
		LookupKey: &threading.ScheduledMessagesRequest_ThreadID{
			ThreadID: threadID,
		},
		Status: []threading.ScheduledMessageStatus{threading.SCHEDULED_MESSAGE_STATUS_PENDING},
	}).WithReturns(&threading.ScheduledMessagesResponse{}, nil))

	res := g.query(ctx, `
		mutation _ ($threadID: ID!, $entityID: String!, $scheduledForTimestamp: Int!) {
			scheduleMessage(input: {
				clientMutationId: "a1b2c3",
				threadID: $threadID,
				scheduledForTimestamp: $scheduledForTimestamp,
				actingEntityID: $entityID,
				message: {
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
			}
		}`, map[string]interface{}{
		"threadID":              threadID,
		"entityID":              entID,
		"scheduledForTimestamp": scheduleFor,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"scheduleMessage": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))
}
