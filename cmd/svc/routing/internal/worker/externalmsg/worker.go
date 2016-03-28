package externalmsg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/operational"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type externalMessageWorker struct {
	started               bool
	sqsAPI                sqsiface.SQSAPI
	sqsURL                string
	snsAPI                snsiface.SNSAPI
	blockAccountsTopicARN string
	directory             directory.DirectoryClient
	threading             threading.ThreadsClient
}

// NewWorker returns a worker that consumes SQS messages
// to route them to the thread service.
func NewWorker(
	sqsAPI sqsiface.SQSAPI,
	sqsURL string,
	snsAPI snsiface.SNSAPI,
	blockAccountsTopicARN string,
	directory directory.DirectoryClient,
	threading threading.ThreadsClient,
) worker.Worker {
	return &externalMessageWorker{
		sqsAPI:                sqsAPI,
		sqsURL:                sqsURL,
		snsAPI:                snsAPI,
		blockAccountsTopicARN: blockAccountsTopicARN,
		directory:             directory,
		threading:             threading,
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

func (r *externalMessageWorker) Stop(wait time.Duration) {
	// TODO
}

func (r *externalMessageWorker) Started() bool {
	return r.started
}

func (r *externalMessageWorker) process(pem *excomms.PublishedExternalMessage) error {

	golog.Debugf("Processing incoming external message: %s → %s. Direction: %s", pem.FromChannelID, pem.ToChannelID, pem.Direction.String())

	ctx := context.Background()

	var toEntities, fromEntities, externalEntities []*directory.Entity
	var organizationID, externalChannelID string
	var contactType directory.ContactType
	switch pem.Type {
	case excomms.PublishedExternalMessage_SMS, excomms.PublishedExternalMessage_OUTGOING_CALL_EVENT, excomms.PublishedExternalMessage_INCOMING_CALL_EVENT:
		contactType = directory.ContactType_PHONE
	case excomms.PublishedExternalMessage_EMAIL:
		contactType = directory.ContactType_EMAIL
	default:
		return errors.Trace(fmt.Errorf("Unknown message type %s", pem.Type.String()))
	}

	switch pem.Direction {

	case excomms.PublishedExternalMessage_INBOUND:

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

		toEntity := determineProviderOrOrgEntity(toEntityLookupRes, pem.ToChannelID)
		if toEntity == nil {
			return errors.Trace(fmt.Errorf("No organization or provider found for %s", pem.ToChannelID))
		}
		toEntities = []*directory.Entity{toEntity}

		if isMessageSpam(pem) {

			accountIDs, err := r.determineAccountIDsOfProvidersInOrg(ctx, toEntity)
			if err != nil {
				return errors.Trace(err)
			}

			for _, accountID := range accountIDs {

				bar := operational.BlockAccountRequest{
					AccountID: accountID,
				}

				data, err := bar.Marshal()
				if err != nil {
					golog.Errorf("Unable to marshal block account request for account %s: %s", accountID, err.Error())
					continue
				}

				_, err = r.snsAPI.Publish(&sns.PublishInput{
					Message:  ptr.String(base64.StdEncoding.EncodeToString(data)),
					TopicArn: ptr.String(r.blockAccountsTopicARN),
				})
				if err != nil {
					golog.Errorf("Unable to publish message to block accounts topic for account %s: %s", accountID, err.Error())
					continue
				}
			}

			// routing complete if message is spam
			return nil
		}

		organizationID = determineOrganizationID(toEntity)
		fromEntities = determineExternalEntities(fromEntityLookupRes, organizationID)
		externalEntities = fromEntities
		externalChannelID = pem.FromChannelID
	case excomms.PublishedExternalMessage_OUTBOUND:
		var err error
		fromEntities, err = lookupEntities(ctx, pem.GetOutgoing().CallerEntityID, r.directory)
		if err != nil {
			return errors.Trace(err)
		} else if len(fromEntities) != 1 {
			return errors.Trace(fmt.Errorf("Expected 1 internal/organization entity but got back %d", len(fromEntities)))
		}

		toEntities, err = lookupEntities(ctx, pem.GetOutgoing().CalleeEntityID, r.directory)
		if err != nil {
			return errors.Trace(err)
		}

		organizationID = determineOrganizationID(fromEntities[0])
		externalEntities = toEntities
		externalChannelID = pem.ToChannelID
	}

	if len(externalEntities) > 0 {
		if golog.Default().L(golog.DEBUG) {
			externalEntityIDs := make([]string, len(externalEntities))
			for i, externalEntity := range externalEntities {
				externalEntityIDs[i] = externalEntity.ID
			}
			golog.Debugf("FromEntities for %s found. %s", pem.FromChannelID, externalEntities)
		}
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
						ContactType: contactType,
						Value:       externalChannelID,
					},
				},
			})
		if err != nil {
			return errors.Trace(err)
		}

		externalEntities = []*directory.Entity{res.Entity}
		if pem.Direction == excomms.PublishedExternalMessage_INBOUND {
			fromEntities = externalEntities
		}
	}

	for _, externalEntity := range externalEntities {

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
		var fromName, toName string
		var fromEntity, toEntity *directory.Entity

		switch pem.Direction {
		case excomms.PublishedExternalMessage_INBOUND:
			fromName = determineDisplayName(pem.FromChannelID, contactType, externalEntity)
			fromEntity = externalEntity
			toName = determineDisplayName(pem.ToChannelID, contactType, toEntities[0])
			toEntity = toEntities[0]
		case excomms.PublishedExternalMessage_OUTBOUND:
			fromName = determineDisplayName(pem.FromChannelID, contactType, fromEntities[0])
			fromEntity = fromEntities[0]
			toName = determineDisplayName(pem.ToChannelID, contactType, externalEntity)
			toEntity = externalEntity
		}

		// TODO: The creation of this mesage should not be the responsibility
		// of the routing layer. It should probably be the responsibility of the
		// external communications layer before it is published.
		switch pem.Type {

		case excomms.PublishedExternalMessage_SMS:
			endpointChannel = threading.Endpoint_SMS
			text = pem.GetSMSItem().Text
			title = bml.BML{"SMS"}
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

		case excomms.PublishedExternalMessage_INCOMING_CALL_EVENT:
			endpointChannel = threading.Endpoint_VOICE
			switch pem.GetIncoming().Type {
			case excomms.IncomingCallEventItem_ANSWERED:
				if d := pem.GetIncoming().DurationInSeconds; d != 0 {
					title = bml.BML{fmt.Sprintf("Inbound call, %d:%02ds", d/60, d%60)}
				} else {
					title = bml.BML{"Inbound call, answered"}
				}
				summary = "Called, answered"
			case excomms.IncomingCallEventItem_UNANSWERED:
				title = bml.BML{"Inbound call, no answer"}
				summary = "Called, no answer"
			case excomms.IncomingCallEventItem_LEFT_VOICEMAIL:
				title = bml.BML{"Voicemail"}
				summary = "Called, left voicemail"

				if len(strings.TrimSpace(pem.GetIncoming().TranscriptionText)) > 0 {
					text = "Transcription: " + strconv.Quote(strings.TrimSpace(pem.GetIncoming().TranscriptionText))
				}
				attachments = []*threading.Attachment{
					{
						Type: threading.Attachment_AUDIO,
						Data: &threading.Attachment_Audio{
							Audio: &threading.AudioAttachment{
								URL:        pem.GetIncoming().VoicemailURL,
								DurationNS: pem.GetIncoming().VoicemailDurationNS,
							},
						},
					},
				}
			}
		case excomms.PublishedExternalMessage_OUTGOING_CALL_EVENT:
			endpointChannel = threading.Endpoint_VOICE

			switch pem.GetOutgoing().Type {
			case excomms.OutgoingCallEventItem_PLACED:
				title = bml.BML{"Outbound call"}
				summary = fmt.Sprintf("%s called %s", fromName, toName)
			case excomms.OutgoingCallEventItem_ANSWERED:
				if d := pem.GetOutgoing().DurationInSeconds; d != 0 {
					title = bml.BML{fmt.Sprintf("Outbound call, answered. %d:%02ds", d/60, d%60)}
				} else {
					title = bml.BML{"Outbound call, answered"}
				}
				summary = fmt.Sprintf("%s called %s, answered", fromName, toName)
			case excomms.OutgoingCallEventItem_UNANSWERED:
				title = bml.BML{"Outbound call, no answer"}
				summary = fmt.Sprintf("%s called %s, no answer", fromName, toName)
			}

		case excomms.PublishedExternalMessage_EMAIL:
			endpointChannel = threading.Endpoint_EMAIL
			subject := pem.GetEmailItem().Subject
			if len(strings.TrimSpace(subject)) == 0 {
				subject = "No Subject"
			}
			text = fmt.Sprintf("Subject: %s\n\n%s", subject, pem.GetEmailItem().Body)
			title = bml.BML{"Email"}
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
			summary = fmt.Sprintf("%s: %s", fromName, text)
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
						ID:      fromName,
					},
					Destinations: []*threading.Endpoint{
						{
							Channel: endpointChannel,
							ID:      pem.ToChannelID,
						},
					},
					Internal:     false,
					MessageTitle: titleStr,
					Text:         plainText,
					Attachments:  attachments,
					Summary:      summary,
					Type:         threading.ThreadType_EXTERNAL,
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
						ID:      fromName,
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
		golog.Debugf("Message posted from %s → %s : %s", fromEntity.ID, toEntity.ID, text)
	}

	return nil
}

func (e *externalMessageWorker) determineAccountIDsOfProvidersInOrg(ctx context.Context, ent *directory.Entity) ([]string, error) {

	var accountIDs []string

	switch ent.Type {

	case directory.EntityType_INTERNAL:
		accountIDs = []string{determineAccountIDFromEntityExternalID(ent)}

	case directory.EntityType_ORGANIZATION:

		orgLookupRes, err := e.directory.LookupEntities(
			ctx,
			&directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: ent.ID,
				},
				RequestedInformation: &directory.RequestedInformation{
					Depth: 1,
					EntityInformation: []directory.EntityInformation{
						directory.EntityInformation_MEMBERS,
						directory.EntityInformation_EXTERNAL_IDS,
					},
				},
			})
		if err != nil {
			return nil, errors.Trace(err)
		} else if len(orgLookupRes.Entities) != 1 {
			return nil, errors.Trace(fmt.Errorf("Expected 1 entity but got %d for %s", len(orgLookupRes.Entities), ent.ID))
		}

		for _, member := range orgLookupRes.Entities[0].Members {
			if member.Type == directory.EntityType_INTERNAL {
				if accountID := determineAccountIDFromEntityExternalID(member); accountID != "" {
					accountIDs = append(accountIDs, accountID)
				}
			}
		}
	}

	return accountIDs, nil
}

func determineAccountIDFromEntityExternalID(ent *directory.Entity) string {
	for _, externalID := range ent.ExternalIDs {
		if strings.HasPrefix(externalID, auth.AccountIDPrefix) {
			return externalID
		}
	}
	return ""
}

func determineDisplayName(channelID string, contactType directory.ContactType, entity *directory.Entity) string {
	fromName := channelID
	if entity.Info != nil && entity.Info.DisplayName != "" {
		return entity.Info.DisplayName
	} else if contactType == directory.ContactType_PHONE {
		formattedPhone, err := phone.Format(fromName, phone.Pretty)
		if err == nil {
			return formattedPhone
		}
	}
	return fromName
}

func lookupEntities(ctx context.Context, entityID string, dir directory.DirectoryClient) ([]*directory.Entity, error) {
	res, err := dir.LookupEntities(
		ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: entityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res.Entities, nil
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
					directory.EntityInformation_EXTERNAL_IDS,
				},
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
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

			if strings.EqualFold(c.Value, value) {
				return entity
			}
		}
	}
	return nil
}

func determineExternalEntities(res *directory.LookupEntitiesByContactResponse, organizationID string) []*directory.Entity {
	if res == nil {
		return nil
	}

	externalEntities := make([]*directory.Entity, 0, len(res.Entities))
	for _, entity := range res.Entities {
		if entity.Type != directory.EntityType_EXTERNAL {
			continue
		}
		// if entity is external, determine membership to the specified organization.
		for _, m := range entity.Memberships {
			if m.Type == directory.EntityType_ORGANIZATION && m.ID == organizationID {
				externalEntities = append(externalEntities, entity)
			}
		}
	}
	return externalEntities
}

func isMessageSpam(pem *excomms.PublishedExternalMessage) bool {

	if pem.Type == excomms.PublishedExternalMessage_SMS && pem.Direction == excomms.PublishedExternalMessage_INBOUND {
		text := pem.GetSMSItem().Text
		if strings.Contains(text, "(WeChat Verification Code)") {
			return true
		} else if strings.Contains(text, "Your TALK2 verification code is") {
			return true
		} else if strings.Contains(text, "is your verification code for Instanumber") {
			return true
		} else if strings.Contains(text, "Your Swytch PIN :") {
			return true
		}
	}
	return false
}
