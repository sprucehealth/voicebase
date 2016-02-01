package externalmsg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
		if grpc.Code(errors.Cause(err)) != codes.NotFound {
			return errors.Trace(err)
		}
	}

	fromEntityLookupRes, err := lookupEntitiesByContact(ctx, pem.FromChannelID, r.directory)
	if err != nil {
		if grpc.Code(errors.Cause(err)) != codes.NotFound {
			return errors.Trace(err)
		}
	}

	var toEntity, fromEntity, externalEntity *directory.Entity
	var organizationID, externalChannelID string
	var contactType directory.ContactType
	switch pem.Type {
	case excomms.PublishedExternalMessage_SMS, excomms.PublishedExternalMessage_CALL_EVENT:
		contactType = directory.ContactType_PHONE
	case excomms.PublishedExternalMessage_EMAIL:
		contactType = directory.ContactType_EMAIL
	default:
		return errors.Trace(fmt.Errorf("Unknown message type %s", pem.Type.String()))
	}

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
				EntityInfo: &directory.EntityInfo{
					DisplayName: pem.FromChannelID,
				},
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
						ContactType: contactType,
						Value:       externalChannelID,
					},
				},
			})
		if err != nil {
			return errors.Trace(err)
		}

		externalEntity = res.Entity
		if pem.Direction == excomms.PublishedExternalMessage_INBOUND {
			fromEntity = externalEntity
		}
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
	var summary string
	var title bml.BML

	fromName := pem.FromChannelID
	if fromEntity.Info != nil && fromEntity.Info.DisplayName != "" {
		fromName = fromEntity.Info.DisplayName
	} else if contactType == directory.ContactType_PHONE {
		formattedPhone, err := phone.Format(fromName, phone.Pretty)
		if err == nil {
			fromName = formattedPhone
		}
	}
	toName := pem.ToChannelID
	if toEntity.Info != nil && toEntity.Info.DisplayName != "" {
		toName = toEntity.Info.DisplayName
	} else if contactType == directory.ContactType_PHONE {
		formattedPhone, err := phone.Format(toName, phone.Pretty)
		if err == nil {
			toName = formattedPhone
		}
	}

	// TODO: The creation of this mesage should not be the responsibility
	// of the routing layer. It should probably be the responsibility of the
	// external communications layer before it is published.
	switch pem.Type {

	case excomms.PublishedExternalMessage_SMS:
		endpointChannel = threading.Endpoint_SMS
		text = pem.GetSMSItem().Text
		title = bml.Parsef("%s texted %s",
			&bml.Ref{Type: bml.EntityRef, ID: fromEntity.ID, Text: fromName},
			&bml.Ref{Type: bml.EntityRef, ID: toEntity.ID, Text: toName})
		summary = fmt.Sprintf("%s: %s", fromName, text)

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
		title = bml.Parsef("%s called %s",
			&bml.Ref{Type: bml.EntityRef, ID: fromEntity.ID, Text: fromName},
			&bml.Ref{Type: bml.EntityRef, ID: toEntity.ID, Text: toName})
		switch pem.GetCallEventItem().Type {
		case excomms.CallEventItem_INCOMING_ANSWERED:
			title = append(title, ", answered")
			summary = "Called, answered"
		case excomms.CallEventItem_INCOMING_UNANSWERED:
			title = append(title, ", no answer")
			summary = "Called, no answer"
		case excomms.CallEventItem_INCOMING_LEFT_VOICEMAIL:
			title = append(title, ", left voicemail")
			summary = "Called, left voicemail"
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
		case excomms.CallEventItem_OUTGOING_ANSWERED:
			title = append(title, ", answered")
			summary = fmt.Sprintf("%s called %s, answered", fromName, toName)
		case excomms.CallEventItem_OUTGOING_UNANSWERED:
			title = append(title, ", no answer")
			summary = fmt.Sprintf("%s called %s, no answer", fromName, toName)
		}
	case excomms.PublishedExternalMessage_EMAIL:
		endpointChannel = threading.Endpoint_EMAIL
		text = pem.GetEmailItem().Body
		subject := pem.GetEmailItem().Subject
		title = bml.Parsef("%s emailed %s, Subject: %s",
			&bml.Ref{Type: bml.EntityRef, ID: fromEntity.ID, Text: fromName},
			&bml.Ref{Type: bml.EntityRef, ID: toEntity.ID, Text: toName},
			subject,
		)
		summary = fmt.Sprintf("Subject: %s", subject)

		for _, attachmentItem := range pem.GetEmailItem().Attachments {
			if strings.HasPrefix(attachmentItem.ContentType, "image") {
				attachments = append(attachments, &threading.Attachment{
					Type:  threading.Attachment_IMAGE,
					Title: attachmentItem.Name,
					Data: &threading.Attachment_Image{
						Image: &threading.ImageAttachment{
							Mimetype: attachmentItem.ContentType,
							URL:      attachmentItem.URL,
						},
					},
				})
			} else {
				attachments = append(attachments, &threading.Attachment{
					Type:  threading.Attachment_GENERIC_URL,
					Title: attachmentItem.Name,
					Data: &threading.Attachment_GenericURL{
						GenericURL: &threading.GenericURLAttachment{
							Mimetype: attachmentItem.ContentType,
							URL:      attachmentItem.URL,
						},
					},
				})
			}
		}
	}

	titleStr, err := title.Format()
	if err != nil {
		return errors.Trace(err)
	}
	plainText, err := bml.BML{text}.Format()
	if err != nil {
		return errors.Trace(err)
	}
	if summary == "" {
		summary = fmt.Sprintf("%s: %s", fromName, titleStr)
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
				Internal:    false,
				Title:       titleStr,
				Text:        plainText,
				Attachments: attachments,
				Summary:     summary,
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
				Title:       titleStr,
				Text:        plainText,
				Attachments: attachments,
				Summary:     summary,
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
	}
	return res, nil
}

func determineOrganizationID(entity *directory.Entity) string {
	if entity.Type == directory.EntityType_ORGANIZATION {
		return entity.ID
	}

	for _, m := range entity.Memberships {
		if m.Type == directory.EntityType_ORGANIZATION {
			return m.ID
		}
	}

	return ""
}

func determineProviderOrOrgEntity(res *directory.LookupEntitiesByContactResponse, value string) *directory.Entity {
	if res == nil {
		return nil
	}
	for _, entity := range res.Entities {
		switch entity.Type {
		case directory.EntityType_ORGANIZATION, directory.EntityType_INTERNAL:
		case directory.EntityType_EXTERNAL:
			continue
		}
		for _, c := range entity.Contacts {
			if c.Value == value {
				return entity
			}
		}
	}
	return nil
}

func determineExternalEntity(res *directory.LookupEntitiesByContactResponse, organizationID string) *directory.Entity {
	if res == nil {
		return nil
	}
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
