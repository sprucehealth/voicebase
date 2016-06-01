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
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/cmd/svc/routing/internal/dal"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/operational"
	"github.com/sprucehealth/backend/svc/settings"
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
	settings              settings.SettingsClient
	excomms               excomms.ExCommsClient
	webDomain             string
	dal                   dal.DAL
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
	settings settings.SettingsClient,
	excomms excomms.ExCommsClient,
	webDomain string,
	dal dal.DAL,
) worker.Worker {
	return &externalMessageWorker{
		sqsAPI:                sqsAPI,
		sqsURL:                sqsURL,
		snsAPI:                snsAPI,
		blockAccountsTopicARN: blockAccountsTopicARN,
		directory:             directory,
		threading:             threading,
		settings:              settings,
		excomms:               excomms,
		webDomain:             webDomain,
		dal:                   dal,
	}
}

var errLogMessageAsErrored = errors.New("errored external message")
var errLogMessageAsSpam = errors.New("spam external message")

const (
	statusProcessed = "PROCESSED"
	statusError     = "ERROR"
	statusSpam      = "SPAM"
)

func (r *externalMessageWorker) Start() {
	if r.started {
		return
	}

	r.started = true
	go func() {
		for {
			sqsRes, err := r.sqsAPI.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            &r.sqsURL,
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

				status := statusProcessed
				if err := r.process(&pem); err != nil {
					switch err {
					case errLogMessageAsErrored:
						status = statusError
					case errLogMessageAsSpam:
						status = statusSpam
					default:
						golog.Context("handle", *item.ReceiptHandle).Errorf(err.Error())
						continue
					}
				}

				if err := r.dal.LogExternalMessage(data, pem.Type.String(), pem.FromChannelID, pem.ToChannelID, status); err != nil {
					golog.Errorf("Unable to persist message to database: %s", err.Error())
					continue
				}

				// delete the message just handled
				_, err = r.sqsAPI.DeleteMessage(
					&sqs.DeleteMessageInput{
						QueueUrl:      &r.sqsURL,
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
	var orgEntity *directory.Entity
	var externalChannelID string
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
			golog.Errorf(`No organization or provider found for %s.
				Note that this message will be considered processed and will be marked as errored in the database.
				If this message should be routed to a thread, then manual intervention will be required to put the message back into the SQS queue after the issue has been resolved.`, pem.ToChannelID)
			return errLogMessageAsErrored
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
					TopicArn: &r.blockAccountsTopicARN,
				})
				if err != nil {
					golog.Errorf("Unable to publish message to block accounts topic for account %s: %s", accountID, err.Error())
					continue
				}
			}

			// routing complete if message is spam
			return errLogMessageAsSpam
		}

		orgEntity = determineOrganization(toEntity)
		fromEntities = determineExternalEntities(fromEntityLookupRes, orgEntity.ID)
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

		orgEntity = determineOrganization(fromEntities[0])
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
				InitialMembershipEntityID: orgEntity.ID,
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
		var urgentVoicemail bool

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
							MediaID:  a.MediaID,
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
			case excomms.IncomingCallEventItem_LEFT_VOICEMAIL, excomms.IncomingCallEventItem_LEFT_URGENT_VOICEMAIL:
				title = bml.BML{"Voicemail"}
				if pem.GetIncoming().Type == excomms.IncomingCallEventItem_LEFT_URGENT_VOICEMAIL {
					summary = "Called, left URGENT voicemail"
					title = bml.BML{"URGENT Voicemail"}
					urgentVoicemail = true
				} else {
					summary = "Called, left voicemail"
				}

				if len(strings.TrimSpace(pem.GetIncoming().TranscriptionText)) > 0 {
					text = "Transcription: " + strconv.Quote(strings.TrimSpace(pem.GetIncoming().TranscriptionText))
				}
				attachments = []*threading.Attachment{
					{
						Type: threading.Attachment_AUDIO,
						Data: &threading.Attachment_Audio{
							Audio: &threading.AudioAttachment{
								MediaID:    pem.GetIncoming().VoicemailMediaID,
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
								MediaID:  attachmentItem.MediaID,
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
								URL:      attachmentItem.MediaID,
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
			threadRes, err := r.threading.CreateThread(
				ctx,
				&threading.CreateThreadRequest{
					OrganizationID: orgEntity.ID,
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
					SystemTitle:  externalEntity.Info.DisplayName,
				},
			)
			if err != nil {
				return errors.Trace(err)
			}
			externalThread = threadRes.Thread
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

		if urgentVoicemail {
			r.notifyOfUrgentVoicemail(ctx, orgEntity, externalThread)
		}
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
				RootTypes:  []directory.EntityType{directory.EntityType_ORGANIZATION},
				ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL},
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

func (e *externalMessageWorker) notifyOfUrgentVoicemail(ctx context.Context, orgEntity *directory.Entity, thread *threading.Thread) error {
	var sprucePhoneNumber string
	for _, c := range orgEntity.Contacts {
		if c.Provisioned && c.ContactType == directory.ContactType_PHONE {
			sprucePhoneNumber = c.Value
			break
		}
	}

	forwardingListValue, err := settings.GetStringListValue(
		context.Background(),
		e.settings,
		&settings.GetValuesRequest{
			NodeID: orgEntity.ID,
			Keys: []*settings.ConfigKey{
				{
					Key:    excommsSettings.ConfigKeyForwardingList,
					Subkey: sprucePhoneNumber,
				},
			},
		})
	if err != nil {
		return errors.Trace(err)
	}

	for _, item := range forwardingListValue.Values {

		if _, err := e.excomms.SendMessage(ctx, &excomms.SendMessageRequest{
			Channel: excomms.ChannelType_SMS,
			Message: &excomms.SendMessageRequest_SMS{
				SMS: &excomms.SMSMessage{
					Text:            "You have received an urgent voicemail on Spruce.",
					FromPhoneNumber: sprucePhoneNumber,
					ToPhoneNumber:   item,
				},
			},
		}); err != nil {
			golog.Warningf("Unable to send sms from %s to %s for urgent voicemail", sprucePhoneNumber, item)
		}

		// 1 second later send a second text with the deeplink in there
		toPhoneNumber := item
		time.AfterFunc(time.Second, func() {
			if _, err := e.excomms.SendMessage(ctx, &excomms.SendMessageRequest{
				Channel: excomms.ChannelType_SMS,
				Message: &excomms.SendMessageRequest_SMS{
					SMS: &excomms.SMSMessage{
						Text:            fmt.Sprintf("Urgent voicemail here: %s", deeplink.ThreadURLShareable(e.webDomain, orgEntity.ID, thread.ID)),
						FromPhoneNumber: sprucePhoneNumber,
						ToPhoneNumber:   toPhoneNumber,
					},
				},
			}); err != nil {
				golog.Warningf("Unable to send sms from %s to %s for urgent voicemail", sprucePhoneNumber, item)
			}
		})
	}

	return nil
}
