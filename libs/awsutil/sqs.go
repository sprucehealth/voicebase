package awsutil

import (
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
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

// SQSWorker is a worker that processes messages from SQS
type SQSWorker struct {
	started  uint32
	sqsAPI   sqsiface.SQSAPI
	sqsURL   string
	processF func(string) error
	stopCh   chan chan struct{}
}

// NewSQSWorker returns a worker that consumes SQS messages
// and passes them through the provided process function
func NewSQSWorker(
	sqsAPI sqsiface.SQSAPI,
	sqsURL string,
	processF func(string) error,
) *SQSWorker {
	return &SQSWorker{
		sqsAPI:   sqsAPI,
		sqsURL:   sqsURL,
		processF: processF,
		stopCh:   make(chan chan struct{}, 1),
	}
}

// Started resturns true iff the worker is currently running
func (w *SQSWorker) Started() bool {
	return atomic.LoadUint32(&w.started) != 0
}

// Stop signals the worker to stop waiting for a duration for it to stop.
func (w *SQSWorker) Stop(wait time.Duration) {
	if w.Started() {
		ch := make(chan struct{})
		w.stopCh <- ch
		select {
		case <-ch:
		case <-time.After(wait):
		}
	}
}

// Start starts the worker consuming messages if it's not already doing so.
func (w *SQSWorker) Start() {
	if atomic.SwapUint32(&w.started, 1) == 1 {
		return
	}
	go func() {
		defer atomic.StoreUint32(&w.started, 0)
		for {
			select {
			case ch := <-w.stopCh:
				ch <- struct{}{}
				return
			default:
			}

			sqsRes, err := w.sqsAPI.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            ptr.String(w.sqsURL),
				MaxNumberOfMessages: ptr.Int64(1),
				VisibilityTimeout:   ptr.Int64(60 * 5),
				WaitTimeSeconds:     ptr.Int64(20),
			})
			if err != nil {
				golog.Errorf("Failed to receive message: %s", err.Error())
				continue
			}

			for _, item := range sqsRes.Messages {
				if err := w.processF(*item.Body); err != nil {
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
					golog.Errorf("Failed to delete message: %s", err.Error())
				}
			}
		}
	}()
}
