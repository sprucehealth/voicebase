package awsutil

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
)

// SNSSQSMessage is the format of the message on an SQS queue
// when it subscribres to an SNS topic.
type SNSSQSMessage struct {
	Type             string
	MessageID        string `xml:"MessageId" json:"MessageId"`
	TopicArn         string
	Subject          string
	Message          string
	Timestamp        time.Time
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	UnsubscribeURL   string
}

type sqsWorker struct {
	startLock sync.Mutex
	started   bool
	sqsAPI    sqsiface.SQSAPI
	sqsURL    string
	processF  func([]byte) error
}

// NewSQSWorker returns a worker that consumes SQS messages
// and passes them through the provided process function
func NewSQSWorker(
	sqsAPI sqsiface.SQSAPI,
	sqsURL string,
	processF func([]byte) error,
) worker.Worker {
	return &sqsWorker{
		startLock: sync.Mutex{},
		sqsAPI:    sqsAPI,
		sqsURL:    sqsURL,
		processF:  processF,
	}
}

func (w *sqsWorker) Started() bool {
	return w.started
}

func (w *sqsWorker) Start() {
	w.startLock.Lock()
	if w.started {
		return
	}
	w.started = true
	w.startLock.Unlock()
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
				var m SNSSQSMessage
				if err := json.Unmarshal([]byte(*item.Body), &m); err != nil {
					golog.Errorf(err.Error())
					continue
				}

				golog.Debugf("Processing message %s", *item.ReceiptHandle)
				if err := w.processF([]byte(*item.Body)); err != nil {
					golog.Errorf(err.Error())
					continue
				}

				// delete the message we just handled
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
