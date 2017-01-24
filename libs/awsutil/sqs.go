package awsutil

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"context"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/crypt"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

const (
	sqsWorkerVisibilityTimeoutSeconds = 60 * 5
	sqsWorkerWaitTimeSeconds          = 20
	sqsWorkerMaxMessages              = 1
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
	started                  uint32
	sqsAPI                   sqsiface.SQSAPI
	sqsURL                   string
	processF                 func(context.Context, string) error
	stopCh                   chan chan struct{}
	visibilityTimeoutSeconds int64
	waitTimeSeconds          int64
	maxMessages              int64
}

// ErrMsgNotProcessedYet is a specific error returned when the procesing function
// wants to communicate that the message should not be deleted yet as it has not been processed.
var ErrMsgNotProcessedYet = errors.New("sqs message not processed yet")

// ErrDelayedRetry is a specific error to signal to the worker to retry
// processing of the sqs message after the specified duration (set to be the visibility
// timeout of the message)
type ErrDelayedRetry struct {
	Duration time.Duration
}

func (e ErrDelayedRetry) Error() string {
	return fmt.Sprintf("retry after %s", e.Duration.String())
}

func ErrRetryAfter(duration time.Duration) error {
	return &ErrDelayedRetry{
		Duration: duration,
	}
}

// VisibilityTimeout set ths visibility timeout in seconds for the messages
// received from the SQS queue.
func VisibilityTimeout(timeoutSeconds int64) func(*SQSWorker) {
	return func(s *SQSWorker) {
		s.waitTimeSeconds = timeoutSeconds
	}
}

// WaitTimeSeconds sets the time in seconds to wait before returning
// from a receive messages call.
func WaitTimeSeconds(timeSeconds int64) func(*SQSWorker) {
	return func(s *SQSWorker) {
		s.waitTimeSeconds = timeSeconds
	}
}

// MaxMessages sets the maximum number of messages possible to return
// from a receive message call to the SQS queue.
func MaxMessages(num int64) func(*SQSWorker) {
	return func(s *SQSWorker) {
		s.maxMessages = num
	}
}

// NewSQSWorker returns a worker that consumes SQS messages
// and passes them through the provided process function
func NewSQSWorker(
	sqsAPI sqsiface.SQSAPI,
	sqsURL string,
	processF func(context.Context, string) error,
	options ...func(*SQSWorker),
) *SQSWorker {
	s := &SQSWorker{
		sqsAPI:   sqsAPI,
		sqsURL:   sqsURL,
		processF: processF,
		stopCh:   make(chan chan struct{}, 1),
		visibilityTimeoutSeconds: sqsWorkerVisibilityTimeoutSeconds,
		waitTimeSeconds:          sqsWorkerWaitTimeSeconds,
		maxMessages:              sqsWorkerMaxMessages,
	}

	for _, option := range options {
		option(s)
	}

	return s
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
				QueueUrl:            &w.sqsURL,
				MaxNumberOfMessages: &w.maxMessages,
				VisibilityTimeout:   &w.visibilityTimeoutSeconds,
				WaitTimeSeconds:     &w.waitTimeSeconds,
			})
			if err != nil {
				golog.Errorf("SQSError: [QueueURL: %s] - Failed to receive message: %s", w.sqsURL, err)
				continue
			}

			for _, item := range sqsRes.Messages {
				w.process(item)
			}
		}
	}()
}

func (w *SQSWorker) process(msg *sqs.Message) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, sqsWorkerVisibilityTimeoutSeconds*time.Second)
	defer cancel()
	if err := w.processF(ctx, *msg.Body); err != nil {
		if errors.Cause(err) == ErrMsgNotProcessedYet {
			return
		} else if edr, ok := errors.Cause(err).(*ErrDelayedRetry); ok {
			if _, err := w.sqsAPI.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
				QueueUrl:          ptr.String(w.sqsURL),
				ReceiptHandle:     msg.ReceiptHandle,
				VisibilityTimeout: ptr.Int64(int64(edr.Duration.Seconds())),
			}); err != nil {
				golog.Context("handle", *msg.ReceiptHandle).Errorf("Failed to change message visibility: %s", err.Error())
			}
			return
		}
		golog.Context("handle", *msg.ReceiptHandle).Errorf(err.Error())
		return
	}

	// delete the message we just handled
	_, err := w.sqsAPI.DeleteMessage(
		&sqs.DeleteMessageInput{
			QueueUrl:      ptr.String(w.sqsURL),
			ReceiptHandle: msg.ReceiptHandle,
		},
	)
	if err != nil {
		golog.Context("handle", *msg.ReceiptHandle).Errorf("Failed to delete message: %s", err.Error())
	}
}

type encryptedSQS struct {
	sqsiface.SQSAPI
	encrypter crypt.Encrypter
	decrypter crypt.Decrypter
}

// NewEncryptedSQS returns an initialized instance of encryptedSQS
func NewEncryptedSQS(masterKeyARN string, kms kmsiface.KMSAPI, sqs sqsiface.SQSAPI) (sqsiface.SQSAPI, error) {
	kmsEncrypter, err := NewKMSEncrypter(masterKeyARN, kms)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize KMS encrypter: %s", err)
	}
	return &encryptedSQS{
		SQSAPI:    sqs,
		encrypter: kmsEncrypter,
		decrypter: NewKMSDecrypter(masterKeyARN, kms),
	}, nil
}

func (e *encryptedSQS) SendMessage(in *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	eBody, err := e.encrypter.Encrypt([]byte(*in.MessageBody))
	if err != nil {
		return nil, errors.Trace(err)
	}
	in.MessageBody = ptr.String(base64.StdEncoding.EncodeToString(eBody))
	return e.SQSAPI.SendMessage(in)
}

func (e *encryptedSQS) SendMessageBatch(in *sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error) {
	return nil, errors.New("Not implemented")
}

func (e *encryptedSQS) ReceiveMessage(in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	resp, err := e.SQSAPI.ReceiveMessage(in)
	if err != nil {
		return nil, errors.Trace(err)
	}
	for i, m := range resp.Messages {
		// If our message was produced byt the encrypted sns publisher, we need to do some wrangling to get it back
		// Hack: Attempt to detect non blob payloads by looking for json encoding
		if *m.Body != "" && (*m.Body)[0] == '{' {
			snsMessage := &SNSSQSMessage{}
			if err := json.Unmarshal([]byte(*m.Body), snsMessage); err != nil {
				return nil, errors.Trace(err)
			}
			eMessage, err := base64.StdEncoding.DecodeString(snsMessage.Message)
			if err != nil {
				return nil, errors.Trace(err)
			}
			dMessage, err := e.decrypter.Decrypt(eMessage)
			if err != nil {
				return nil, errors.Trace(err)
			}
			snsMessage.Message = string(dMessage)
			bMessage, err := json.Marshal(snsMessage)
			if err != nil {
				return nil, errors.Trace(err)
			}
			resp.Messages[i].Body = ptr.String(string(bMessage))
		} else {
			// If it is just a normal sqs message then we can just decode and decrypt
			sBody, err := base64.StdEncoding.DecodeString(*m.Body)
			if err != nil {
				return nil, errors.Trace(err)
			}
			dBody, err := e.decrypter.Decrypt([]byte(sBody))
			if err != nil {
				return nil, errors.Trace(err)
			}
			resp.Messages[i].Body = ptr.String(string(dBody))
		}
	}
	return resp, nil
}

// CreateSQSQueue returns the URL of the created queue
func CreateSQSQueue(sqsCLI sqsiface.SQSAPI, name string) (string, error) {
	in := &sqs.CreateQueueInput{
		QueueName: &name,
	}
	createResp, err := sqsCLI.CreateQueue(in)
	if err != nil {
		return "", errors.Trace(err)
	}
	return *createResp.QueueUrl, nil
}

const (
	// AWSErrCodeSQSQueueNotFound the code returned from AWS when a queue isn't found
	AWSErrCodeSQSQueueNotFound = "AWS.SimpleQueueService.NonExistentQueue"
)

// CreateSQSQueueIfNotExists returns the ARN of the existing or created queue
func CreateSQSQueueIfNotExists(sqsCLI sqsiface.SQSAPI, queueURL string) (string, error) {
	requestedAttributed := []*string{ptr.String("QueueArn")}
	attrResp, err := sqsCLI.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       &queueURL,
		AttributeNames: requestedAttributed,
	})
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == AWSErrCodeSQSQueueNotFound {
			queueName, err := ResourceNameFromSQSURL(queueURL)
			if err != nil {
				return "", errors.Errorf("queue %s NOT FOUND. Unable to get queue name from URL to create due to: %s", queueURL, err)
			}
			golog.Infof("Queue %s was NOT FOUND. Attempting to create it.", queueName)
			queueURL, err = CreateSQSQueue(sqsCLI, queueName)
			if err != nil {
				return "", errors.Errorf("queue %s NOT FOUND. Failed to create queue due to: %s", queueName, err)
			}
			golog.Infof("Queue %s was successfully created", queueName)
			attrResp, err = sqsCLI.GetQueueAttributes(&sqs.GetQueueAttributesInput{
				QueueUrl:       &queueURL,
				AttributeNames: requestedAttributed,
			})
			if err != nil {
				return "", errors.Errorf("failed to get AWS error for GetQueueAttributes queue %s: %s", queueURL, err)
			}
		} else {
			return "", errors.Errorf("failed to get attributes of queue %s: %s", queueURL, err)
		}
	} else if err != nil {
		return "", errors.Errorf("failed to get AWS error for GetQueueAttributes queue %s: %s", queueURL, err)
	}
	parn, ok := attrResp.Attributes["QueueArn"]
	if !ok {
		return "", errors.Errorf("failed to get Queue ARN out of Attributes response for %s", queueURL)
	}
	return *parn, nil
}
