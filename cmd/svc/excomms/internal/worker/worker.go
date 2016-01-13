package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/mail"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/recapco/emailreplyparser"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/excomms"
)

type IncomingRawMessageWorker struct {
	started              bool
	sqsAPI               sqsiface.SQSAPI
	sqsURL               string
	externalMessageTopic string
	snsAPI               snsiface.SNSAPI
	dal                  dal.DAL
}

func NewWorker(
	awsSession *session.Session,
	incomingRawMessageQueueName string,
	snsAPI snsiface.SNSAPI,
	externalMessageTopic string,
	dal dal.DAL) (*IncomingRawMessageWorker, error) {

	incomingRawMessageQueue := sqs.New(awsSession)
	res, err := incomingRawMessageQueue.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: ptr.String(incomingRawMessageQueueName),
	})
	if err != nil {
		return nil, err
	}
	return &IncomingRawMessageWorker{
		sqsAPI:               incomingRawMessageQueue,
		sqsURL:               *res.QueueUrl,
		externalMessageTopic: externalMessageTopic,
		snsAPI:               snsAPI,
		dal:                  dal,
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

		for i, m := range params.MediaItems {
			smsItem.SMSItem.Attachments[i] = &excomms.MediaAttachment{
				URL:         m.MediaURL,
				ContentType: m.ContentType,
			}
		}

		sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
			FromChannelID: params.From,
			ToChannelID:   params.To,
			Timestamp:     rm.Timestamp,
			Direction:     excomms.PublishedExternalMessage_INBOUND,
			Type:          excomms.PublishedExternalMessage_SMS,
			Item:          smsItem,
		})

		// TODO: Delete SMS from twilio
		// TODO: Upload any media objects attached to SMS to our system and delete from twilio

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
				// TODO: Attachments
			},
		}

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
