package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"strconv"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/recapco/emailreplyparser"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/cleaner"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/utils"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/excomms"
)

type IncomingRawMessageWorker struct {
	started              bool
	sqsAPI               sqsiface.SQSAPI
	sqsURL               string
	externalMessageTopic string
	snsAPI               snsiface.SNSAPI
	dal                  dal.DAL
	store                storage.Store
	twilioAccountSID     string
	twilioAuthToken      string
	resourceCleanerTopic string
}

func NewWorker(
	incomingRawMessageQueueName string,
	snsAPI snsiface.SNSAPI,
	sqsAPI sqsiface.SQSAPI,
	externalMessageTopic string,
	dal dal.DAL,
	store storage.Store,
	twilioAccountSID, twilioAuthToken string,
	resourceCleanerTopic string) (*IncomingRawMessageWorker, error) {

	res, err := sqsAPI.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: ptr.String(incomingRawMessageQueueName),
	})
	if err != nil {
		return nil, err
	}
	return &IncomingRawMessageWorker{
		sqsAPI:               sqsAPI,
		sqsURL:               *res.QueueUrl,
		externalMessageTopic: externalMessageTopic,
		snsAPI:               snsAPI,
		dal:                  dal,
		store:                store,
		twilioAccountSID:     twilioAccountSID,
		twilioAuthToken:      twilioAuthToken,
		resourceCleanerTopic: resourceCleanerTopic,
	}, nil
}

func (w *IncomingRawMessageWorker) Start() {
	if w.started {
		return
	}
	w.started = true

	go func() {
		for {

			sqsRes, err := w.sqsAPI.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            ptr.String(w.sqsURL),
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

				var notif sns.IncomingRawMessageNotification
				if err := json.Unmarshal(data, &notif); err != nil {
					golog.Errorf(err.Error())
					continue
				}

				golog.Debugf("Process message %s", *item.ReceiptHandle)

				if err := w.process(&notif); err != nil {
					golog.Errorf(err.Error())
					continue
				}

				// delete the message just handled
				_, err = w.sqsAPI.DeleteMessage(
					&sqs.DeleteMessageInput{
						QueueUrl:      ptr.String(w.sqsURL),
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

func (w *IncomingRawMessageWorker) process(notif *sns.IncomingRawMessageNotification) error {
	rm, err := w.dal.IncomingRawMessage(notif.ID)
	if err != nil {
		return errors.Trace(err)
	}

	switch rm.Type {

	case rawmsg.Incoming_TWILIO_SMS:
		params := rm.GetTwilio()
		smsItem := &excomms.PublishedExternalMessage_SMSItem{
			SMSItem: &excomms.SMSItem{
				Text:        params.Body,
				Attachments: make([]*excomms.MediaAttachment, params.NumMedia),
			},
		}

		mediaMap := make(map[uint64]*models.Media)
		for i, m := range params.MediaItems {

			media, err := w.uploadTwilioMediaToS3(m.ContentType, m.MediaURL)
			if err != nil {
				return errors.Trace(err)
			}
			mediaMap[media.ID] = media
			m.ID = media.ID

			cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
				Type:       models.DeleteResourceRequest_TWILIO_MEDIA,
				ResourceID: m.MediaURL,
			})

			smsItem.SMSItem.Attachments[i] = &excomms.MediaAttachment{
				URL:         media.URL,
				ContentType: m.ContentType,
			}

		}

		_, err = utils.PersistRawMessage(w.dal, mediaMap, rm)
		if err != nil {
			return errors.Trace(err)
		}

		sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
			FromChannelID: params.From,
			ToChannelID:   params.To,
			Timestamp:     rm.Timestamp,
			Direction:     excomms.PublishedExternalMessage_INBOUND,
			Type:          excomms.PublishedExternalMessage_SMS,
			Item:          smsItem,
		})

		cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_SMS,
			ResourceID: params.MessageSID,
		})

	case rawmsg.Incoming_TWILIO_VOICEMAIL:
		params := rm.GetTwilio()

		mediaMap := make(map[uint64]*models.Media, 1)

		media, err := w.uploadTwilioMediaToS3("audio/mpeg", params.RecordingURL+".mp3")
		if err != nil {
			return errors.Trace(err)
		}
		mediaMap[media.ID] = media
		params.RecordingMediaID = media.ID

		cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_RECORDING,
			ResourceID: params.RecordingSID,
		})

		// also delete the calls that originated in the voicemail
		cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_CALL,
			ResourceID: params.CallSID,
		})

		cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_CALL,
			ResourceID: params.ParentCallSID,
		})

		_, err = utils.PersistRawMessage(w.dal, mediaMap, rm)
		if err != nil {
			return errors.Trace(err)
		}

		sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
			FromChannelID: params.From,
			ToChannelID:   params.To,
			Timestamp:     rm.Timestamp,
			Direction:     excomms.PublishedExternalMessage_INBOUND,
			Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
			Item: &excomms.PublishedExternalMessage_Incoming{
				Incoming: &excomms.IncomingCallEventItem{
					Type:              excomms.IncomingCallEventItem_LEFT_VOICEMAIL,
					DurationInSeconds: params.RecordingDuration,
					URL:               media.URL,
				},
			},
		})

	case rawmsg.Incoming_SENDGRID_EMAIL:
		sgEmail := rm.GetSendGrid()

		senderAddress, err := mail.ParseAddress(sgEmail.Sender)
		if err != nil {
			return errors.Trace(fmt.Errorf("Unable to parse email address %s :%s", sgEmail.Sender, err.Error()))
		}

		recipientAddress, err := mail.ParseAddress(sgEmail.Recipient)
		if err != nil {
			return errors.Trace(fmt.Errorf("Unable to parse email address %s :%s", sgEmail.Recipient, err.Error()))
		}

		text, err := emailreplyparser.ParseReply(sgEmail.Text)
		if err != nil {
			return errors.Trace(err)
		}

		emailItem := &excomms.PublishedExternalMessage_EmailItem{
			EmailItem: &excomms.EmailItem{
				Body:    text,
				Subject: sgEmail.Subject,
			},
		}

		// lookup media objects if there are any
		mediaIDs := make([]uint64, len(sgEmail.Attachments))
		for i, item := range sgEmail.Attachments {
			mediaIDs[i] = item.ID
		}

		mediaMap, err := w.dal.LookupMedia(mediaIDs)
		if err != nil {
			return errors.Trace(err)
		}

		// populate attachments
		mediaAttachments := make([]*excomms.MediaAttachment, sgEmail.NumAttachments)
		for i, item := range sgEmail.Attachments {
			media := mediaMap[item.ID]
			mediaAttachments[i] = &excomms.MediaAttachment{
				URL:         media.URL,
				ContentType: media.Type,
			}
			if media.Name != nil {
				mediaAttachments[i].Name = *media.Name
			}
		}
		emailItem.EmailItem.Attachments = mediaAttachments

		sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
			FromChannelID: senderAddress.Address,
			ToChannelID:   recipientAddress.Address,
			Timestamp:     rm.Timestamp,
			Direction:     excomms.PublishedExternalMessage_INBOUND,
			Type:          excomms.PublishedExternalMessage_EMAIL,
			Item:          emailItem,
		})
	default:
		golog.Errorf("Unknown raw message type %s. Dropping...", rm.Type.String())
	}

	return nil
}

func (w *IncomingRawMessageWorker) uploadTwilioMediaToS3(contentType string, url string) (*models.Media, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.SetBasicAuth(w.twilioAccountSID, w.twilioAuthToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer res.Body.Close()

	// Note: have to read all the data into memory here because
	// there is no way to know the size of the data when working with a reader
	// via the response body
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	id, err := idgen.NewID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	s3URL, err := w.store.Put(strconv.FormatInt(int64(id), 10), data, contentType, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &models.Media{
		ID:   id,
		Type: contentType,
		URL:  s3URL,
	}, nil
}
