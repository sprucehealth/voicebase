package awsutil

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs"
)

type SQS struct {
	handle   int
	mu       sync.Mutex
	Messages map[string]map[string]string
}

func (s *SQS) newHandle() string {
	s.handle++
	return strconv.Itoa(s.handle)
}

func (s *SQS) SendMessage(req *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Messages == nil {
		s.Messages = make(map[string]map[string]string)
	}
	q := s.Messages[*req.QueueURL]
	if q == nil {
		q = make(map[string]string)
		s.Messages[*req.QueueURL] = q
	}
	q[s.newHandle()] = *req.MessageBody
	return &sqs.SendMessageOutput{}, nil
}

func (s *SQS) ReceiveMessage(req *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	s.mu.Lock()

	maxNumberOfMessages := 1
	if req.MaxNumberOfMessages != nil {
		maxNumberOfMessages = int(*req.MaxNumberOfMessages)
	}

	var msgs []*sqs.Message
	if s.Messages != nil && maxNumberOfMessages > 0 {
		if q := s.Messages[*req.QueueURL]; q != nil {
			for h, m := range q {
				msgs = append(msgs, &sqs.Message{
					MessageID:     aws.String(h),
					ReceiptHandle: aws.String(h),
					Body:          aws.String(m),
				})
				if len(msgs) == maxNumberOfMessages {
					break
				}
			}
		}
	}

	s.mu.Unlock()

	if len(msgs) == 0 && req.WaitTimeSeconds != nil && *req.WaitTimeSeconds != 0 {
		// Sleep for a bit so we don't create a busy loop. Since this is just for mocking / testing
		// no need to trigger on a new messages.
		time.Sleep(time.Millisecond * 100)
	}
	return &sqs.ReceiveMessageOutput{Messages: msgs}, nil
}

func (s *SQS) DeleteMessage(req *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Messages == nil {
		return &sqs.DeleteMessageOutput{}, nil
	}
	if q := s.Messages[*req.QueueURL]; q != nil {
		delete(q, *req.ReceiptHandle)
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func (s *SQS) GetQueueURL(req *sqs.GetQueueURLInput) (*sqs.GetQueueURLOutput, error) {
	return &sqs.GetQueueURLOutput{QueueURL: req.QueueName}, nil
}
func (s *SQS) AddPermission(*sqs.AddPermissionInput) (*sqs.AddPermissionOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) ChangeMessageVisibility(*sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) ChangeMessageVisibilityBatch(*sqs.ChangeMessageVisibilityBatchInput) (*sqs.ChangeMessageVisibilityBatchOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) CreateQueue(*sqs.CreateQueueInput) (*sqs.CreateQueueOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) DeleteMessageBatch(*sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) DeleteQueue(*sqs.DeleteQueueInput) (*sqs.DeleteQueueOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) ListDeadLetterSourceQueues(*sqs.ListDeadLetterSourceQueuesInput) (*sqs.ListDeadLetterSourceQueuesOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) ListQueues(*sqs.ListQueuesInput) (*sqs.ListQueuesOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) PurgeQueue(*sqs.PurgeQueueInput) (*sqs.PurgeQueueOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) RemovePermission(*sqs.RemovePermissionInput) (*sqs.RemovePermissionOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) SendMessageBatch(*sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
func (s *SQS) SetQueueAttributes(*sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error) {
	return nil, errors.New("awstest.sqs: not implemented")
}
