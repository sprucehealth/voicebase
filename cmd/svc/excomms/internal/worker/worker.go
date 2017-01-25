package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/audioutil"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/transcription"
	"github.com/sprucehealth/backend/libs/twilio"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
)

type IncomingRawMessageWorker struct {
	started                     bool
	sqsAPI                      sqsiface.SQSAPI
	sqsURL                      string
	externalMessageTopic        string
	snsAPI                      snsiface.SNSAPI
	dal                         dal.DAL
	store                       storage.Store
	twilioAccountSID            string
	twilioAuthToken             string
	resourceCleanerTopic        string
	statErrorVoicemailUpload    string
	settings                    settings.SettingsClient
	transcriptionProvider       transcription.Provider
	transcriptionTrackingSQSURL string
}

func NewWorker(
	incomingRawMessageQueueName string,
	snsAPI snsiface.SNSAPI,
	sqsAPI sqsiface.SQSAPI,
	externalMessageTopic string,
	dal dal.DAL,
	store storage.Store,
	twilioAccountSID, twilioAuthToken string,
	resourceCleanerTopic string,
	settings settings.SettingsClient,
	transcriptionProvider transcription.Provider,
	transcriptionTrackingSQSURL string) (*IncomingRawMessageWorker, error) {

	res, err := sqsAPI.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &incomingRawMessageQueueName,
	})
	if err != nil {
		return nil, err
	}
	return &IncomingRawMessageWorker{
		sqsAPI:                      sqsAPI,
		sqsURL:                      *res.QueueUrl,
		externalMessageTopic:        externalMessageTopic,
		snsAPI:                      snsAPI,
		dal:                         dal,
		store:                       store,
		twilioAccountSID:            twilioAccountSID,
		twilioAuthToken:             twilioAuthToken,
		resourceCleanerTopic:        resourceCleanerTopic,
		settings:                    settings,
		transcriptionProvider:       transcriptionProvider,
		transcriptionTrackingSQSURL: transcriptionTrackingSQSURL,
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

				golog := golog.Context("handle", *item.ReceiptHandle)

				var m awsutil.SNSSQSMessage
				if err := json.Unmarshal([]byte(*item.Body), &m); err != nil {
					golog.Errorf("Unable to unmarshal SQS message: " + err.Error())
					continue
				}

				data, err := base64.StdEncoding.DecodeString(m.Message)
				if err != nil {
					golog.Errorf("Unable to decode string %s", err.Error())
					continue
				}

				var notif sns.IncomingRawMessageNotification
				if err := json.Unmarshal(data, &notif); err != nil {
					golog.Errorf("Unable to unmarshal message data: " + err.Error())
					continue
				}

				golog.Debugf("Process message %s", *item.ReceiptHandle)

				if err := w.process(&notif); err != nil {
					if errors.Cause(err) == awsutil.ErrMsgNotProcessedYet {
						continue
					}

					golog.Errorf("Unable to process notification: " + err.Error())
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
					golog.Errorf("Unable to delete message: " + err.Error())
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
		// ensure that the from phone number is a valid number, otherwise reject
		// the SMS
		source, err := phone.ParseNumber(params.From)
		if err != nil {
			golog.Errorf("Invalid phone number as the FROM phone number for incoming SMS: %s", params.From)
			return nil
		}

		destination, err := phone.ParseNumber(params.To)
		if err != nil {
			golog.Errorf("Invalid destination phone number %s: %s", params.To, err)
			return nil
		}

		// ensure that the source number is not a blocked number. If it is, reject the SMS
		blockedNumbers, err := w.dal.LookupBlockedNumbers(context.Background(), destination)
		if err != nil {
			return errors.Trace(err)
		}
		if blockedNumbers.Includes(source) {
			golog.Infof("Dropping SMS since number %s is blocked for %s", source, destination)
			return nil
		}

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

			media.ResourceID, err = twilio.ParseMediaSID(m.MediaURL)
			if err != nil {
				golog.Errorf("Unable to parse mediaSID from url %s : %s", m.MediaURL, err)
			}

			mediaMap[media.ID] = media
			m.ID = media.ID

			smsItem.SMSItem.Attachments[i] = &excomms.MediaAttachment{
				MediaID:     media.ID,
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

		if err := sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
			FromChannelID: params.From,
			ToChannelID:   params.To,
			Timestamp:     rm.Timestamp,
			Direction:     excomms.PublishedExternalMessage_INBOUND,
			Type:          excomms.PublishedExternalMessage_SMS,
			Item:          smsItem,
		}); err != nil {
			return errors.Trace(err)
		}

		cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_SMS,
			ResourceID: params.MessageSID,
		})

	case rawmsg.Incoming_TWILIO_VOICEMAIL:
		params := rm.GetTwilio()

		mediaMap := make(map[string]*models.Media, 1)

		media, err := w.uploadTwilioMediaToS3("audio/mpeg", params.RecordingURL+".mp3")
		if e, ok := errors.Cause(err).(errMediaNotFound); ok {
			golog.Warningf("unable to upload twilio media: %s", e)
			return awsutil.ErrMsgNotProcessedYet
		} else if err != nil {
			return errors.Trace(err)
		}
		if media.Duration == 0 {
			media.Duration = time.Duration(params.RecordingDuration) * time.Second
		}
		media.ResourceID = params.RecordingSID
		mediaMap[media.ID] = media
		params.RecordingMediaID = media.ID

		_, err = utils.PersistRawMessage(w.dal, mediaMap, rm)
		if err != nil {
			return errors.Trace(err)
		}

		incomingCall, err := w.dal.LookupIncomingCall(params.CallSID)
		if err != nil {
			return errors.Trace(err)
		}

		incomingType := excomms.IncomingCallEventItem_LEFT_VOICEMAIL
		if incomingCall.AfterHours && incomingCall.Urgent {
			incomingType = excomms.IncomingCallEventItem_LEFT_URGENT_VOICEMAIL
		}

		var transcribeVoicemail bool
		transcriptionProvider := excommsSettings.TranscriptionProviderTwilio
		valueRes, err := w.settings.GetValues(context.Background(), &settings.GetValuesRequest{
			NodeID: incomingCall.OrganizationID,
			Keys: []*settings.ConfigKey{
				{
					Key: excommsSettings.ConfigKeyTranscribeVoicemail,
				},
				{
					Key: excommsSettings.ConfigKeyTranscriptionProvider,
				},
			},
		})
		if err != nil {
			golog.Errorf("unable to get settings value for org %s key %s : %s", incomingCall.OrganizationID, excommsSettings.ConfigKeyTranscribeVoicemail, err)
		} else if len(valueRes.Values) != 2 {
			golog.Errorf("expected 2 settings to be returned for org %s but got %d", incomingCall.OrganizationID, len(valueRes.Values))
		}

		transcribeVoicemail = valueRes.Values[0].GetBoolean().Value
		transcriptionProvider = valueRes.Values[1].GetSingleSelect().Item.ID

		if transcribeVoicemail && transcriptionProvider == excommsSettings.TranscriptionProviderVoicebase {

			// check if job has already been submitted
			if job, err := w.dal.LookupTranscriptionJob(context.Background(), media.ID); errors.Cause(err) != dal.ErrTranscriptionJobNotFound && err != nil {
				return errors.Errorf("unable to query for transcription job for %s : %s", media.ID, err)
			} else if err == nil && job.Completed {
				// nothing to do if the job has already been completed
				return nil
			}

			expiringURL, err := w.store.ExpiringURL(media.ID, 30*time.Minute)
			if err != nil {
				return errors.Errorf("unable to create expiring url for media %s : %s ", media.ID, err)
			}

			// TODO: Ensure that check for validation errors on the transcription job
			// like the transcription being too short
			job, err := w.transcriptionProvider.SubmitTranscriptionJob(expiringURL)
			if err != nil {
				return errors.Errorf("unable to submit transcription job for media %s : %s", media.ID, err)
			}

			req := &trackTranscriptionRequest{
				JobID:           job.ID,
				MediaID:         media.ID,
				RawMessageID:    notif.ID,
				UrgentVoicemail: incomingType == excomms.IncomingCallEventItem_LEFT_URGENT_VOICEMAIL,
			}

			jsonData, err := json.Marshal(req)
			if err != nil {
				return errors.Errorf("unable to marshal transcription tracking request for %s : %s", req.MediaID, err)
			}

			if err := w.dal.InsertTranscriptionJob(context.Background(), &models.TranscriptionJob{
				MediaID:        media.ID,
				JobID:          job.ID,
				AvailableAfter: time.Now(),
			}); err != nil {
				return errors.Errorf("unable to insert transcription job for media %s : %s", media.ID, err)
			}

			msg := base64.StdEncoding.EncodeToString(jsonData)
			if _, err := w.sqsAPI.SendMessage(&sqs.SendMessageInput{
				QueueUrl:    &w.transcriptionTrackingSQSURL,
				MessageBody: &msg,
			}); err != nil {
				return errors.Errorf("unable to send message on sqs queue %s : %s", w.transcriptionTrackingSQSURL, err)
			}

		} else {
			if err := sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
				FromChannelID: params.From,
				ToChannelID:   params.To,
				Timestamp:     rm.Timestamp,
				Direction:     excomms.PublishedExternalMessage_INBOUND,
				Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
				Item: &excomms.PublishedExternalMessage_Incoming{
					Incoming: &excomms.IncomingCallEventItem{
						Type:                incomingType,
						DurationInSeconds:   params.RecordingDuration,
						VoicemailMediaID:    media.ID,
						VoicemailDurationNS: uint64(media.Duration.Nanoseconds()),
						TranscriptionText:   params.TranscriptionText,
					},
				},
			}); err != nil {
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
		}

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
					MediaID:     media.ID,
					ContentType: media.Type,
				}
				if media.Name != nil {
					mediaAttachments[i].Name = *media.Name
				}
			}
			emailItem.EmailItem.Attachments = mediaAttachments

			if err := sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
				FromChannelID: senderAddress.Address,
				ToChannelID:   recipientAddress.Address,
				Timestamp:     rm.Timestamp,
				Direction:     excomms.PublishedExternalMessage_INBOUND,
				Type:          excomms.PublishedExternalMessage_EMAIL,
				Item:          emailItem,
			}); err != nil {
				return errors.Trace(err)
			}
		}
	default:
		golog.Errorf("Unknown raw message type %s. Dropping...", rm.Type.String())
	}

	return nil
}

type errMediaNotFound string

func (e errMediaNotFound) Error() string {
	return string(e)
}

func (w *IncomingRawMessageWorker) uploadTwilioMediaToS3(contentType, url string) (*models.Media, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create GET request for url %q", url)
	}
	req.SetBasicAuth(w.twilioAccountSID, w.twilioAuthToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "GET failed on url %q", url)
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

		if res.StatusCode == 404 {
			return nil, errors.Trace(errMediaNotFound(fmt.Sprintf("twilio media %s not found", url)))
		}

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

	duration, err := audioutil.Duration(bytes.NewReader(data), contentType)
	if err != nil {
		golog.Errorf("Failed to calculate duration of audio: %s", err)
	}

	id, err := media.NewID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, err = w.store.Put(id, data, contentType, map[string]string{
		"x-amz-meta-duration-ns": strconv.FormatInt(duration.Nanoseconds(), 10),
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &models.Media{
		ID:       id,
		Type:     contentType,
		Duration: duration,
	}, nil
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
