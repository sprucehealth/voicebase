package appmsg

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	exsettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	rsettings "github.com/sprucehealth/backend/cmd/svc/routing/internal/settings"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

type appMessageWorker struct {
	started   bool
	sqsAPI    sqsiface.SQSAPI
	sqsURL    string
	directory directory.DirectoryClient
	excomms   excomms.ExCommsClient
	settings  settings.SettingsClient
}

// NewWorker returns a worker that consumes SQS messages
// to route *inapp* messages to the excomms service
// as SMS.
func NewWorker(
	sqsAPI sqsiface.SQSAPI,
	sqsURL string,
	directory directory.DirectoryClient,
	excomms excomms.ExCommsClient,
	settings settings.SettingsClient,
) worker.Worker {
	return &appMessageWorker{
		sqsAPI:    sqsAPI,
		sqsURL:    sqsURL,
		excomms:   excomms,
		settings:  settings,
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
				golog.Errorf("Unable to receive message: %s", err)
				continue
			}

			for _, item := range sqsRes.Messages {
				log := golog.Context("handle", *item.ReceiptHandle)

				var m awsutil.SNSSQSMessage
				if err := json.Unmarshal([]byte(*item.Body), &m); err != nil {
					log.Errorf("Unable to unmarshal SQS message: %s", err)
					continue
				}

				data, err := base64.StdEncoding.DecodeString(m.Message)
				if err != nil {
					log.Errorf("Unable to decode string %s", err)
					continue
				}

				var pti threading.PublishedThreadItem
				if err := pti.Unmarshal(data); err != nil {
					log.Errorf("Unable to unmarshal published thread item: %s", err)
					continue
				}

				log.Debugf("Process message %s", *item.ReceiptHandle)

				if err := a.process(&pti); err != nil {
					log.Errorf("Unable to process item: %s", err)
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
					log.Errorf("Unable to delete message: %s", err)
				}

				log.Debugf("Delete message %s", *item.ReceiptHandle)
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
	item := pti.Item
	// Only process external thread messages sent via app. Ignore everything else.
	if item.Internal {
		golog.Debugf("Internal message posted. Ignoring...")
		return nil
	}
	msgItem, ok := item.Item.(*threading.ThreadItem_Message)
	if !ok {
		golog.Debugf("Thread item is not a message, it is of type %T. Ignoring...", item.Item)
		return nil
	}
	msg := msgItem.Message
	if !(msg.Source == nil || msg.Source.Channel == threading.ENDPOINT_CHANNEL_APP) {
		golog.Debugf("SourceContact has to have type APP, but has %s. Ignoring...", msg.Source.Channel)
		return nil
	}

	// TODO: Remove this filterings once the APP destination is no longer valid
	destinations := make([]*threading.Endpoint, 0, len(msg.Destinations))
	for _, d := range msg.Destinations {
		if d.Channel != threading.ENDPOINT_CHANNEL_APP {
			destinations = append(destinations, d)
		}
	}

	// Do this circuit break after the above debug logging since it may be useful
	// If there are no destinations then we don't care about this message
	if len(destinations) == 0 {
		golog.Debugf("Message received with no valid destinations. Item ID: %s. Ignoring...", pti.Item.ID)
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
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
			RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		},
	)
	if err != nil {
		return errors.Trace(err)
	} else if len(orgLookupRes.Entities) == 0 {
		return errors.Errorf("Expected organization to exist for id %s", organizationID)
	}
	orgEntity := orgLookupRes.Entities[0]

	// Parse text and render as plain text.
	textBML, err := bml.Parse(msg.Text)
	if e, ok := err.(bml.ErrParseFailure); ok {
		return fmt.Errorf("failed to parse text at pos %d: %s", e.Offset, e.Reason)
	} else if err != nil {
		return errors.New("text is not valid markup")
	}
	plainText, err := textBML.PlainText()
	if err != nil {
		golog.Errorf("Unable to render plain text version for message item %s: %s", item.ID, err)
		// Shouldn't fail here since the parsing should have done validation
		return errors.Trace(err)
	}

	var revealSender bool
	res, err := a.settings.GetValues(ctx, &settings.GetValuesRequest{
		NodeID: organizationID,
		Keys: []*settings.ConfigKey{
			{
				Key: rsettings.ConfigKeyRevealSenderAcrossExcomms,
			},
		},
	})
	if err != nil {
		golog.Errorf("Unable to read settings for reveling sender for organizationID %s: %s", organizationID, err)
	} else if len(res.Values) == 0 {
		golog.Errorf("No value specified for revealing sender for %s", organizationID)
	} else if len(res.Values) != 1 {
		golog.Errorf("Expected 1 value for revealing sender instead got %d for %s", len(res.Values), organizationID)
	} else if res.Values[0].GetBoolean() == nil {
		golog.Errorf("Expected boolean value for revealing sender instead got %T for %s", res.Values[0].Value, organizationID)
	} else {
		revealSender = res.Values[0].GetBoolean().Value
	}

	var mediaIDs []string
	for _, at := range msg.Attachments {
		// TODO: Add async video support?
		if d, ok := at.Data.(*threading.Attachment_Image); ok {
			mediaIDs = append(mediaIDs, d.Image.MediaID)
		}
	}

	// Perform the outbound operations for any remaining valid destinations
	for _, d := range destinations {
		switch d.Channel {
		case threading.ENDPOINT_CHANNEL_APP:
			// Note: Do nothing in this case since it should already be in the app.
			// TODO: Remove this case when Endpoint_APP is removed from the system
		case threading.ENDPOINT_CHANNEL_SMS:
			var provisionedPhoneNumber string
			val, err := settings.GetTextValue(ctx, a.settings, &settings.GetValuesRequest{
				NodeID: item.ActorEntityID,
				Keys: []*settings.ConfigKey{
					{
						Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
					},
				},
			})
			if err == nil {
				provisionedPhoneNumber = val.Value
			} else if errors.Cause(err) != settings.ErrValueNotFound {
				return errors.Errorf("unable to get default number setting for entity %s: %s", item.ActorEntityID, err)
			} else {
				orgContact := determineProvisionedContact(orgEntity, directory.ContactType_PHONE)
				if orgContact == nil {
					golog.Errorf("Unable to determine organization provisioned phone number for org %s. Dropping message...", organizationID)
					return nil
				}
				provisionedPhoneNumber = orgContact.Value
			}

			if revealSender {
				providerEntity, err := determineActorEntity(ctx, a.directory, item.ActorEntityID)
				if err != nil {
					return errors.Trace(err)
				}
				plainText = providerEntity.Info.DisplayName + ": " + plainText
			}

			_, err = a.excomms.SendMessage(
				ctx,
				&excomms.SendMessageRequest{
					UUID:              item.ID,
					DeprecatedChannel: excomms.ChannelType_SMS,
					Message: &excomms.SendMessageRequest_SMS{
						SMS: &excomms.SMSMessage{
							FromPhoneNumber: provisionedPhoneNumber,
							ToPhoneNumber:   d.ID,
							Text:            plainText,
							MediaIDs:        mediaIDs,
						},
					},
				},
			)
			if err != nil {
				switch grpc.Code(err) {
				case excomms.ErrorCodeMessageLengthExceeded:
					golog.Errorf("Unable to send message as the message length was exceeded. Dropping message for now as handling this situation requires manual intervention. Support team should inform user of the situation. Error: %s", err)
					return nil
				case excomms.ErrorCodeSMSIncapableFromPhoneNumber:
					golog.Errorf("Unable to send message as the from phone number does not have SMS capabilities. Error :%s", err)
				case excomms.ErrorCodeMessageDeliveryFailed:
					golog.Errorf("Message %s cannot be delivered. Not going to retry as the error is permanent. Manual intervention required by Support team to report issue to customer. Error = '%s", item.ID, err)
					return nil
				}
				return errors.Errorf("Unable to send message originating from thread item id %s: %s", item.ID, err)
			}
			golog.Debugf("Sent SMS %s → %s. Text %s", provisionedPhoneNumber, d.ID, msg.Text)
		case threading.ENDPOINT_CHANNEL_EMAIL:
			// determine org email address
			orgContact := determineProvisionedContact(orgEntity, directory.ContactType_EMAIL)
			if orgContact == nil {
				golog.Errorf("Unable to determine organization provisioned email for org %s. Dropping message...", organizationID)
				return nil
			}

			fromName := orgEntity.Info.DisplayName
			if revealSender {
				providerEntity, err := determineActorEntity(ctx, a.directory, item.ActorEntityID)
				if err != nil {
					return errors.Trace(err)
				}
				fromName = providerEntity.Info.DisplayName
			}

			_, err = a.excomms.SendMessage(
				ctx,
				&excomms.SendMessageRequest{
					UUID:              item.ID,
					DeprecatedChannel: excomms.ChannelType_EMAIL,
					Message: &excomms.SendMessageRequest_Email{
						Email: &excomms.EmailMessage{
							Subject:          fmt.Sprintf("Message from %s", orgEntity.Info.DisplayName),
							Body:             plainText,
							FromName:         fromName,
							FromEmailAddress: orgContact.Value,
							ToEmailAddress:   d.ID,
							MediaIDs:         mediaIDs,
						},
					},
				},
			)
			if err != nil {
				return errors.Trace(err)
			}
			golog.Debugf("Sent Email %s → %s. Text %s", orgContact.Value, d.ID, msg.Text)
		default:
			golog.Errorf("Dropping destination %s. Unknown how to send message.", d.Channel)
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

func determineActorEntity(ctx context.Context, directoryClient directory.DirectoryClient, actorEntityID string) (*directory.Entity, error) {
	// determine provider (sender of message) to include in the email
	providerLookupRes, err := directoryClient.LookupEntities(
		ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: actorEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
			},
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(providerLookupRes.Entities) != 1 {
		return nil, errors.Errorf("Expected 1 provider to exist for id %s, but got %d", actorEntityID, len(providerLookupRes.Entities))
	}
	return providerLookupRes.Entities[0], nil
}
