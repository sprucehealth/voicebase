package appmsg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

type appMessageWorker struct {
	started   bool
	sqsAPI    sqsiface.SQSAPI
	sqsURL    string
	directory directory.DirectoryClient
	excomms   excomms.ExCommsClient
}

// NewWorker returns a worker that consumes SQS messages
// to route *inapp* messages to the excomms service
// as SMS.
func NewWorker(
	sqsAPI sqsiface.SQSAPI,
	sqsURL string,
	directory directory.DirectoryClient,
	excomms excomms.ExCommsClient,
) worker.Worker {
	return &appMessageWorker{
		sqsAPI:    sqsAPI,
		sqsURL:    sqsURL,
		excomms:   excomms,
		directory: directory,
	}
}

func (a *appMessageWorker) Start() {
	if a.started {
		return
	}
	a.started = true
	go func() {
		for {

			sqsRes, err := a.sqsAPI.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            ptr.String(a.sqsURL),
				MaxNumberOfMessages: ptr.Int64(1),
				VisibilityTimeout:   ptr.Int64(60 * 5),
				WaitTimeSeconds:     ptr.Int64(20),
			})
			if err != nil {
				golog.Errorf(err.Error())
				continue
			}

			for _, item := range sqsRes.Messages {
				var m awsutil.SNSSQSMessage
				if err := json.Unmarshal([]byte(*item.Body), &m); err != nil {
					golog.Errorf(err.Error())
					continue
				}

				data, err := base64.StdEncoding.DecodeString(m.Message)
				if err != nil {
					golog.Errorf("Unable to decode string %s", err.Error())
					continue
				}

				var pti threading.PublishedThreadItem
				if err := pti.Unmarshal(data); err != nil {
					golog.Errorf(err.Error())
					continue
				}

				golog.Debugf("Process message %s", *item.ReceiptHandle)

				if err := a.process(&pti); err != nil {
					golog.Errorf(err.Error())
					continue
				}

				// delete the message just handled
				_, err = a.sqsAPI.DeleteMessage(
					&sqs.DeleteMessageInput{
						QueueUrl:      ptr.String(a.sqsURL),
						ReceiptHandle: item.ReceiptHandle,
					},
				)
				if err != nil {
					golog.Errorf(err.Error())
				}

				golog.Debugf("Delete message %s", *item.ReceiptHandle)
			}
		}
	}()
}

func (a *appMessageWorker) Stop(wait time.Duration) {
	// TODO
}

func (a *appMessageWorker) Started() bool {
	return a.started
}

func (a *appMessageWorker) process(pti *threading.PublishedThreadItem) error {

	// Only process external thread messages sent via app. Ignore everything else.
	if pti.GetItem().Internal {
		golog.Debugf("Internal message posted. Ignoring...")
		return nil
	} else if pti.GetItem().Type != threading.ThreadItem_MESSAGE {
		golog.Debugf("Thread item is not a message, it is of type %s. Ignoring...", pti.GetItem().Type.String())
		return nil
	} else if pti.GetItem().GetMessage().Source.Channel != threading.Endpoint_APP {
		golog.Debugf("SourceContact has to have type APP, but has %s. Ignoring...", pti.GetItem().GetMessage().Source.Channel)
		return nil
	}

	// TODO: Remove this filterings once the APP destination is no longer valid
	destinations := make([]*threading.Endpoint, 0, len(pti.GetItem().GetMessage().Destinations))
	for _, d := range pti.GetItem().GetMessage().Destinations {
		if d.Channel != threading.Endpoint_APP {
			destinations = append(destinations, d)
		}
	}

	// Do this circuit break after the above debug logging since it may be useful
	// If there are no destinations then we don't care about this message
	if len(destinations) == 0 {
		golog.Debugf("Message recieved with no valid destinations. Item ID: %s. Ignoring...", pti.Item.ID)
		return nil
	}

	organizationID := pti.OrganizationID
	ctx := context.Background()

	// look up the entity for the org
	orgLookupRes, err := a.directory.LookupEntities(
		ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: organizationID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERS,
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	)
	if err != nil {
		return errors.Trace(err)
	} else if len(orgLookupRes.Entities) == 0 {
		return errors.Trace(fmt.Errorf("Expected organization to exist for id %s", organizationID))
	}

	// determine external entity that belongs to this organization
	externalEntityLookupRes, err := a.directory.LookupEntities(
		ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: pti.PrimaryEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERS,
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if err != nil {
		return errors.Trace(err)
	} else if len(externalEntityLookupRes.Entities) == 0 {
		return errors.Trace(fmt.Errorf("Expected external entity to exist for id %s", pti.PrimaryEntityID))
	}

	// Parse text and render as plain text.
	textBML, err := bml.Parse(pti.GetItem().GetMessage().Text)
	if e, ok := err.(bml.ErrParseFailure); ok {
		return fmt.Errorf("failed to parse text at pos %d: %s", e.Offset, e.Reason)
	} else if err != nil {
		return errors.New("text is not valid markup")
	}
	plainText, err := textBML.PlainText()
	if err != nil {
		golog.Errorf("Unable to render plain text version for message item %s: "+err.Error(), pti.GetItem().ID)
		// Shouldn't fail here since the parsing should have done validation
		return errors.Trace(err)
	}

	// Perform the outbound operations for any remaining valid destinations
	orgEntity := orgLookupRes.Entities[0]
	for _, d := range destinations {
		switch d.Channel {
		case threading.Endpoint_APP:
			// Note: Do nothing in this case since it should already be in the app.
			// TODO: Remove this case when Endpoint_APP is removed from the system
		case threading.Endpoint_SMS:
			// determine org phone number
			orgContact := determineProvisionedContact(orgEntity, directory.ContactType_PHONE)
			if orgContact == nil {
				golog.Errorf("Unable to determine organization provisioned phone number for org %s. Dropping message...", organizationID)
				return nil
			}

			_, err := a.excomms.SendMessage(
				ctx,
				&excomms.SendMessageRequest{
					UUID:    pti.GetItem().ID,
					Channel: excomms.ChannelType_SMS,
					Message: &excomms.SendMessageRequest_SMS{
						SMS: &excomms.SMSMessage{
							FromPhoneNumber: orgContact.Value,
							ToPhoneNumber:   d.ID,
							Text:            plainText,
						},
					},
				},
			)
			if err != nil {
				return errors.Trace(err)
			}
			golog.Debugf("Sent SMS %s → %s. Text %s", orgContact.Value, d.ID, pti.GetItem().GetMessage().Text)
		case threading.Endpoint_EMAIL:
			// determine org email address
			orgContact := determineProvisionedContact(orgEntity, directory.ContactType_EMAIL)
			if orgContact == nil {
				golog.Errorf("Unable to determine organization provisioned email for org %s. Dropping message...", organizationID)
				return nil
			}

			// determine provider (sender of message) to include in the email
			providerLookupRes, err := a.directory.LookupEntities(
				ctx,
				&directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: pti.GetItem().ActorEntityID,
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth: 0,
					},
				},
			)
			if err != nil {
				return errors.Trace(err)
			} else if len(providerLookupRes.Entities) != 1 {
				return errors.Trace(fmt.Errorf("Expected 1 provider to exist for id %s, but got %d", pti.GetItem().ActorEntityID, len(providerLookupRes.Entities)))
			}
			providerEntity := providerLookupRes.Entities[0]

			_, err = a.excomms.SendMessage(
				ctx,
				&excomms.SendMessageRequest{
					UUID:    pti.GetItem().ID,
					Channel: excomms.ChannelType_EMAIL,
					Message: &excomms.SendMessageRequest_Email{
						Email: &excomms.EmailMessage{
							Subject:          fmt.Sprintf("Message from %s", orgEntity.Info.DisplayName),
							Body:             plainText,
							FromName:         providerEntity.Info.DisplayName,
							FromEmailAddress: orgContact.Value,
							ToEmailAddress:   d.ID,
						},
					},
				},
			)
			if err != nil {
				return errors.Trace(err)
			}
			golog.Debugf("Sent Email %s → %s. Text %s", orgContact.Value, d.ID, pti.GetItem().GetMessage().Text)
		default:
			golog.Warningf("Dropping destination %s. Unknown how to send message.", d.Channel.String())
		}
	}

	return nil
}

func determineProvisionedContact(entity *directory.Entity, contactType directory.ContactType) *directory.Contact {
	if len(entity.Contacts) == 0 {
		return nil
	}

	for _, c := range entity.Contacts {
		if !c.Provisioned {
			continue
		}
		if c.ContactType == contactType {
			return c
		}

	}
	return nil
}
