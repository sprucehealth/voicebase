package externalmsg

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

type externalMessageWorker struct {
	started   bool
	sqsAPI    sqsiface.SQSAPI
	sqsURL    string
	directory directory.DirectoryClient
	threading threading.ThreadsClient
}

// NewWorker returns a worker that consumes SQS messages
// to route them to the thread service.
func NewWorker(
	sqsAPI sqsiface.SQSAPI,
	sqsURL string,
	directory directory.DirectoryClient,
	threading threading.ThreadsClient,
) worker.Worker {
	return &externalMessageWorker{
		sqsAPI:    sqsAPI,
		sqsURL:    sqsURL,
		directory: directory,
		threading: threading,
	}
}

func (r *externalMessageWorker) Start() {
	if r.started {
		return
	}

	r.started = true
	go func() {
		for {

			sqsRes, err := r.sqsAPI.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            ptr.String(r.sqsURL),
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
					golog.Errorf("Unable to decode base64 encoded string: %s", err.Error())
					return
				}

				var pem excomms.PublishedExternalMessage
				if err := pem.Unmarshal(data); err != nil {
					golog.Errorf(err.Error())
					continue
				}
				golog.Debugf("Process message %s.", *item.ReceiptHandle)

				if err := r.process(&pem); err != nil {
					golog.Errorf(err.Error())
					continue
				}

				// delete the message just handled
				_, err = r.sqsAPI.DeleteMessage(
					&sqs.DeleteMessageInput{
						QueueUrl:      ptr.String(r.sqsURL),
						ReceiptHandle: item.ReceiptHandle,
					},
				)
				if err != nil {
					golog.Errorf(err.Error())
					continue
				}
				golog.Debugf("Delete message %s", *item.ReceiptHandle)
			}
		}
	}()
}

func (r *externalMessageWorker) Started() bool {
	return r.started
}

func (r *externalMessageWorker) process(pem *excomms.PublishedExternalMessage) error {

	golog.Debugf("Processing incoming external message: %s → %s. Direction: %s", pem.FromChannelID, pem.ToChannelID, pem.Direction.String())

	ctx := context.Background()

	toEntityLookupRes, err := lookupEntitiesByContact(ctx, pem.ToChannelID, r.directory)
	if err != nil {
		return errors.Trace(err)
	}

	fromEntityLookupRes, err := lookupEntitiesByContact(ctx, pem.FromChannelID, r.directory)
	if err != nil {
		return errors.Trace(err)
	}

	var toEntity, fromEntity, externalEntity *directory.Entity
	var organizationID, externalChannelID string

	switch pem.Direction {

	case excomms.PublishedExternalMessage_INBOUND:
		toEntity = determineProviderOrOrgEntity(toEntityLookupRes, pem.ToChannelID)
		if toEntity == nil {
			return errors.Trace(fmt.Errorf("No organization or provider found for %s", pem.ToChannelID))
		}
		organizationID = determineOrganizationID(toEntity)
		fromEntity = determineExternalEntity(fromEntityLookupRes, organizationID)
		externalEntity = fromEntity
		externalChannelID = pem.FromChannelID

	case excomms.PublishedExternalMessage_OUTBOUND:
		fromEntity = determineProviderOrOrgEntity(fromEntityLookupRes, pem.FromChannelID)
		if fromEntity == nil {
			return errors.Trace(fmt.Errorf("No organization or provider found for %s", pem.FromChannelID))
		}
		organizationID = determineOrganizationID(fromEntity)
		toEntity = determineExternalEntity(toEntityLookupRes, organizationID)
		externalEntity = toEntity
		externalChannelID = pem.ToChannelID
	}

	if externalEntity != nil {
		golog.Debugf("FromEntity for %s found. ID = %s", pem.FromChannelID, fromEntity.ID)
	} else {
		golog.Debugf("External entity for %s not found. Creating new entity...", pem.FromChannelID)
		res, err := r.directory.CreateEntity(
			ctx,
			&directory.CreateEntityRequest{
				Type: directory.EntityType_EXTERNAL,
				InitialMembershipEntityID: organizationID,
				RequestedInformation: &directory.RequestedInformation{
					Depth: 1,
					EntityInformation: []directory.EntityInformation{
						directory.EntityInformation_MEMBERSHIPS,
						directory.EntityInformation_CONTACTS,
					},
				},
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       externalChannelID,
					},
				},
			})
		if err != nil {
			return errors.Trace(err)
		} else if !res.Success {
			return errors.Trace(fmt.Errorf("Unable to create entity. Reason %s. Message %s", res.Failure.Reason.String(), res.Failure.Message))
		}

		externalEntity = res.Entity
	}

	// now that to and from entities have been resolved, post the message to the appropriate
	// thread.
	threadsForMemberRes, err := r.threading.ThreadsForMember(
		ctx,
		&threading.ThreadsForMemberRequest{
			EntityID:    externalEntity.ID,
			PrimaryOnly: true,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// Note: assumption is that there is only one external thread for the entity.
	var externalThread *threading.Thread
	for _, thread := range threadsForMemberRes.Threads {
		if thread.PrimaryEntityID == externalEntity.ID {
			externalThread = thread
			break
		}
	}

	var endpointChannel threading.Endpoint_Channel
	var attachments []*threading.Attachment
	var text string

	// TODO: The creation of this mesage should not be the responsibility
	// of the routing layer. It should probably be the responsibility of the
	// external communications layer before it is published.
	switch pem.Type {

	case excomms.PublishedExternalMessage_SMS:
		endpointChannel = threading.Endpoint_SMS
		text = pem.GetSMSItem().Text

		// populate attachments
		attachments = make([]*threading.Attachment, len(pem.GetSMSItem().Attachments))
		for i, a := range pem.GetSMSItem().Attachments {
			attachments[i] = &threading.Attachment{
				Type: threading.Attachment_IMAGE,
				Data: &threading.Attachment_Image{
					Image: &threading.ImageAttachment{
						Mimetype: a.ContentType,
						URL:      a.URL,
					},
				},
			}
		}

	case excomms.PublishedExternalMessage_CALL_EVENT:
		endpointChannel = threading.Endpoint_VOICE
		switch pem.GetCallEventItem().Type {
		case excomms.CallEventItem_INCOMING_ANSWERED:
			text = fmt.Sprintf("%s called %s, answered.", pem.FromChannelID, toEntity.Name)
		case excomms.CallEventItem_INCOMING_UNANSWERED:
			text = fmt.Sprintf("%s called %s, did not answer.", pem.FromChannelID, toEntity.Name)
		case excomms.CallEventItem_INCOMING_LEFT_VOICEMAIL:
			text = fmt.Sprintf("%s called %s, left voicemail.", pem.FromChannelID, toEntity.Name)
			attachments = []*threading.Attachment{
				{
					Type: threading.Attachment_AUDIO,
					Data: &threading.Attachment_Audio{
						Audio: &threading.AudioAttachment{
							URL:               pem.GetCallEventItem().URL,
							DurationInSeconds: pem.GetCallEventItem().DurationInSeconds,
						},
					},
				},
			}
		case excomms.CallEventItem_OUTGOING_PLACED:
			text = fmt.Sprintf("%s called %s.", fromEntity.Name, pem.ToChannelID)
		case excomms.CallEventItem_OUTGOING_ANSWERED:
			text = fmt.Sprintf("%s called %s, answered.", fromEntity.Name, pem.ToChannelID)
		case excomms.CallEventItem_OUTGOING_UNANSWERED:
			text = fmt.Sprintf("%s called %s, did not answer.", fromEntity.Name, pem.ToChannelID)
		}
	}

	if externalThread == nil {
		golog.Debugf("External thread for %s not found. Creating...", externalEntity.Contacts[0].Value)

		// create thread if one doesn't exist
		_, err := r.threading.CreateThread(
			ctx,
			&threading.CreateThreadRequest{
				OrganizationID: organizationID,
				FromEntityID:   externalEntity.ID,
				Source: &threading.Endpoint{
					Channel: endpointChannel,
					ID:      pem.FromChannelID,
				},
				Destinations: []*threading.Endpoint{
					{
						Channel: endpointChannel,
						ID:      pem.ToChannelID,
					},
				},
				Text:        text,
				Attachments: attachments,
			},
		)
		if err != nil {
			return errors.Trace(err)
		}
	} else {
		golog.Debugf("External thread for %s found. Posting to existing thread...", externalEntity.Contacts[0].Value)
		// post message if thread exists
		_, err = r.threading.PostMessage(
			ctx,
			&threading.PostMessageRequest{
				ThreadID:     externalThread.ID,
				FromEntityID: fromEntity.ID,
				Source: &threading.Endpoint{
					Channel: endpointChannel,
					ID:      pem.FromChannelID,
				},
				Destinations: []*threading.Endpoint{
					{
						Channel: endpointChannel,
						ID:      pem.ToChannelID,
					},
				},
				Internal:    false,
				Text:        text,
				Attachments: attachments,
			},
		)
		if err != nil {
			return errors.Trace(err)
		}
	}

	golog.Debugf("Message posted from %s → %s : %s", pem.FromChannelID, pem.ToChannelID, text)
	return nil
}

func lookupEntitiesByContact(ctx context.Context, contactValue string, dir directory.DirectoryClient) (*directory.LookupEntitiesByContactResponse, error) {
	res, err := dir.LookupEntitiesByContact(
		ctx,
		&directory.LookupEntitiesByContactRequest{
			ContactValue: contactValue,
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	} else if !res.Success && res.Failure.Reason != directory.LookupEntitiesByContactResponse_Failure_NOT_FOUND {
		return nil, errors.Trace(fmt.Errorf("Unable to lookup entity by contact. Reason %s. Message %s", res.Failure.Reason.String(), res.Failure.Message))
	}
	return res, nil
}

func determineOrganizationID(entity *directory.Entity) string {
	if entity.Type == directory.EntityType_ORGANIZATION {
		return entity.ID
	} else {
		for _, m := range entity.Memberships {
			if m.Type == directory.EntityType_ORGANIZATION {
				return m.ID
			}
		}
	}

	return ""
}

func determineProviderOrOrgEntity(res *directory.LookupEntitiesByContactResponse, phoneNumber string) *directory.Entity {
	for _, entity := range res.Entities {
		switch entity.Type {
		case directory.EntityType_ORGANIZATION, directory.EntityType_INTERNAL:
		case directory.EntityType_EXTERNAL:
			continue
		}
		for _, c := range entity.Contacts {
			if c.Value == phoneNumber {
				return entity
			}
		}
	}
	return nil
}

func determineExternalEntity(res *directory.LookupEntitiesByContactResponse, organizationID string) *directory.Entity {
	for _, entity := range res.Entities {
		if entity.Type != directory.EntityType_EXTERNAL {
			continue
		}
		// if entity is external, determine membership to the specified organization.
		for _, m := range entity.Memberships {
			if m.Type == directory.EntityType_ORGANIZATION && m.ID == organizationID {
				return entity
			}
		}
	}
	return nil
}
