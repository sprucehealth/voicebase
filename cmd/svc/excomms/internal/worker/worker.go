package worker

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

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
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/tcolgate/mp3"
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
		QueueName: &incomingRawMessageQueueName,
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

type smtpEnvelope struct {
	To []string `json:"to"`
}

func (w *IncomingRawMessageWorker) Start() {
	if w.started {
		return
	}
	w.started = true

	go func() {
		for {

			sqsRes, err := w.sqsAPI.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            &w.sqsURL,
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
					golog.Context("handle", *item.ReceiptHandle).Errorf(err.Error())
					continue
				}

				// delete the message just handled
				_, err = w.sqsAPI.DeleteMessage(
					&sqs.DeleteMessageInput{
						QueueUrl:      &w.sqsURL,
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

		mediaMap := make(map[string]*models.Media)
		for i, m := range params.MediaItems {

			media, err := w.uploadTwilioMediaToS3(m.ContentType, m.MediaURL)
			if err != nil {
				return errors.Trace(err)
			}
			mediaMap[media.ID] = media
			m.ID = media.ID

			smsItem.SMSItem.Attachments[i] = &excomms.MediaAttachment{
				URL:         media.URL,
				ContentType: m.ContentType,
			}

		}

		_, err = utils.PersistRawMessage(w.dal, mediaMap, rm)
		if err != nil {
			return errors.Trace(err)
		}

		// go through media to publish them for cleanup once we have persisted the raw message
		for _, mediaItem := range params.MediaItems {
			cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
				Type:       models.DeleteResourceRequest_TWILIO_MEDIA,
				ResourceID: mediaItem.MediaURL,
			})
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

		mediaMap := make(map[string]*models.Media, 1)

		media, err := w.uploadTwilioMediaToS3("audio/mpeg", params.RecordingURL+".mp3")
		if err != nil {
			return errors.Trace(err)
		}
		if media.Duration == 0 {
			media.Duration = time.Duration(params.RecordingDuration) * time.Second
		}
		mediaMap[media.ID] = media
		params.RecordingMediaID = media.ID

		_, err = utils.PersistRawMessage(w.dal, mediaMap, rm)
		if err != nil {
			return errors.Trace(err)
		}

		cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_RECORDING,
			ResourceID: params.RecordingSID,
		})

		if params.TranscriptionStatus == rawmsg.TwilioParams_TRANSCRIPTION_STATUS_COMPLETED {
			cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
				Type:       models.DeleteResourceRequest_TWILIO_TRANSCRIPTION,
				ResourceID: params.TranscriptionSID,
			})
		}

		incomingCall, err := w.dal.LookupIncomingCall(params.CallSID)
		if err != nil {
			return errors.Trace(err)
		}

		incomingType := excomms.IncomingCallEventItem_LEFT_VOICEMAIL
		if incomingCall.AfterHours && incomingCall.Urgent {
			incomingType = excomms.IncomingCallEventItem_LEFT_URGENT_VOICEMAIL
		}

		sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
			FromChannelID: params.From,
			ToChannelID:   params.To,
			Timestamp:     rm.Timestamp,
			Direction:     excomms.PublishedExternalMessage_INBOUND,
			Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
			Item: &excomms.PublishedExternalMessage_Incoming{
				Incoming: &excomms.IncomingCallEventItem{
					Type:                incomingType,
					DurationInSeconds:   params.RecordingDuration,
					VoicemailURL:        media.URL,
					VoicemailDurationNS: uint64(media.Duration.Nanoseconds()),
					TranscriptionText:   params.TranscriptionText,
				},
			},
		})

	case rawmsg.Incoming_SENDGRID_EMAIL:
		sgEmail := rm.GetSendGrid()

		senderAddress, err := parseAddress(sgEmail.Sender)
		if err != nil {
			return errors.Trace(fmt.Errorf("Unable to parse email address %s :%s", sgEmail.Sender, err.Error()))
		}

		// use the smtpEnvelope to determine who to send the mail to because
		// it contains the information about the recipient whether the email was
		// delivered due to a forwarding rule, the CC field or the forwarded field
		// containing the spruce email address
		var envelope smtpEnvelope
		if err := json.Unmarshal([]byte(sgEmail.SMTPEnvelope), &envelope); err != nil {
			return errors.Trace(fmt.Errorf("Unable to parse the SMTP envelope '%s' : %s", sgEmail.SMTPEnvelope, err.Error()))
		}

		for _, add := range envelope.To {
			recipientAddress, err := mail.ParseAddress(add)
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
			mediaIDs := make([]string, len(sgEmail.Attachments))
			for i, item := range sgEmail.Attachments {
				if item.DeprecatedID != 0 {
					mediaIDs[i] = strconv.FormatUint(item.DeprecatedID, 10)
				} else {
					mediaIDs[i] = item.ID
				}
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
		}
	default:
		golog.Errorf("Unknown raw message type %s. Dropping...", rm.Type.String())
	}

	return nil
}

func (w *IncomingRawMessageWorker) uploadTwilioMediaToS3(contentType, url string) (*models.Media, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Annotatef(errors.Trace(err), "failed to create GET request for url '%s'", url)
	}
	req.SetBasicAuth(w.twilioAccountSID, w.twilioAuthToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Annotatef(errors.Trace(err), "GET failed on url '%s'", url)
	}
	defer res.Body.Close()

	// Note: have to read all the data into memory here because
	// there is no way to know the size of the data when working with a reader
	// via the response body
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		// Avoid flooding the log
		if len(data) > 1000 {
			data = data[:1000]
		}
		dataStr := string(data)
		if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/") {
			// Avoid non-valid characters from breaking anything in case we get back binary
			dataStr = strconv.Quote(string(data))
		}
		return nil, errors.Trace(fmt.Errorf("Expected status code 2xx when pulling media, got %d: %s", res.StatusCode, dataStr))
	}

	var duration time.Duration
	if contentType == "audio/mpeg" {
		duration, err = mp3Duration(bytes.NewReader(data))
		if err != nil {
			golog.Errorf("Failed to calculate duration of mp3: %s", err)
		}
	}

	id, err := media.NewID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	s3URL, err := w.store.Put(id, data, contentType, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &models.Media{
		ID:       id,
		Type:     contentType,
		URL:      s3URL,
		Duration: duration,
	}, nil
}

func mp3Duration(r io.Reader) (time.Duration, error) {
	dec := mp3.NewDecoder(r)
	var frame mp3.Frame
	var duration time.Duration
	for {
		if err := dec.Decode(&frame); err != nil {
			if err == io.EOF {
				return duration, nil
			}
			return 0, errors.Trace(err)
		}
		duration += frame.Duration()
	}
}

func parseAddress(addr string) (*mail.Address, error) {
	addr = strings.TrimSpace(addr)

	idx := strings.LastIndex(addr, "<")
	if idx < 1 {
		return mail.ParseAddress(addr)
	}

	if addr[0] == '"' {
		return mail.ParseAddress(addr)
	}

	// lets quote the sting before the angle bracket to treat
	// all characters before the angle bracket as part of the name.
	// this is to work around the situation where the name is not quoted
	// and has characters like parenthesis in it which causes the
	// parser to error (eg. Joe Schmoe (Test) <joe@schmoe.com>)
	return mail.ParseAddress(strconv.Quote(addr[:idx]) + addr[idx:])
}
