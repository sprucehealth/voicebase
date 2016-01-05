package appmsg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal/worker"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
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

	// send an SMS from org to external entity
	// TODO: Respect list of destinationContacts if present for where to route message.
	orgEntity := orgLookupRes.Entities[0]
	externalEntity := externalEntityLookupRes.Entities[0]

	if len(orgEntity.Contacts) == 0 {
		golog.Warningf("Organization id %d does not have contacts. Dropping message...", orgEntity.ID)
		return nil
	}
	orgContact := orgEntity.Contacts[0]

	if len(externalEntity.Contacts) == 0 {
		golog.Warningf("Externaal entity %d does not have contacts. Dropping message...", orgEntity.ID)
		return nil
	}
	externalContact := externalEntity.Contacts[0]

	_, err = a.excomms.SendMessage(
		ctx,
		&excomms.SendMessageRequest{
			FromChannelID: orgContact.Value,
			ToChannelID:   externalContact.Value,
			Text:          pti.GetItem().GetMessage().Text,
			Channel:       excomms.ChannelType_SMS,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	golog.Debugf("Sent SMS %s → %s. Text %s", orgContact.Value, externalContact.Value, pti.GetItem().GetMessage().Text)
	return nil
}
