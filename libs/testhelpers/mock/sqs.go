package mock

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

type SQSAPI struct {
	*Expector
}

var _ sqsiface.SQSAPI = NewSQSAPI(nil)

// NewSQSAPI returns a mock compatible SQSAPI instance
func NewSQSAPI(t *testing.T) *SQSAPI {
	return &SQSAPI{&Expector{T: t}}
}

func (s *SQSAPI) AddPermissionRequest(in *sqs.AddPermissionInput) (*request.Request, *sqs.AddPermissionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.AddPermissionOutput)
}

func (s *SQSAPI) AddPermission(in *sqs.AddPermissionInput) (*sqs.AddPermissionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.AddPermissionOutput), SafeError(rets[1])
}

func (s *SQSAPI) ChangeMessageVisibilityRequest(in *sqs.ChangeMessageVisibilityInput) (*request.Request, *sqs.ChangeMessageVisibilityOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ChangeMessageVisibilityOutput)
}

func (s *SQSAPI) ChangeMessageVisibility(in *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ChangeMessageVisibilityOutput), SafeError(rets[1])
}

func (s *SQSAPI) ChangeMessageVisibilityBatchRequest(in *sqs.ChangeMessageVisibilityBatchInput) (*request.Request, *sqs.ChangeMessageVisibilityBatchOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ChangeMessageVisibilityBatchOutput)
}

func (s *SQSAPI) ChangeMessageVisibilityBatch(in *sqs.ChangeMessageVisibilityBatchInput) (*sqs.ChangeMessageVisibilityBatchOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ChangeMessageVisibilityBatchOutput), SafeError(rets[1])
}

func (s *SQSAPI) CreateQueueRequest(in *sqs.CreateQueueInput) (*request.Request, *sqs.CreateQueueOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.CreateQueueOutput)
}

func (s *SQSAPI) CreateQueue(in *sqs.CreateQueueInput) (*sqs.CreateQueueOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.CreateQueueOutput), SafeError(rets[1])
}

func (s *SQSAPI) DeleteMessageRequest(in *sqs.DeleteMessageInput) (*request.Request, *sqs.DeleteMessageOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.DeleteMessageOutput)
}

func (s *SQSAPI) DeleteMessage(in *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.DeleteMessageOutput), SafeError(rets[1])
}

func (s *SQSAPI) DeleteMessageBatchRequest(in *sqs.DeleteMessageBatchInput) (*request.Request, *sqs.DeleteMessageBatchOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.DeleteMessageBatchOutput)
}

func (s *SQSAPI) DeleteMessageBatch(in *sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.DeleteMessageBatchOutput), SafeError(rets[1])
}

func (s *SQSAPI) DeleteQueueRequest(in *sqs.DeleteQueueInput) (*request.Request, *sqs.DeleteQueueOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.DeleteQueueOutput)
}

func (s *SQSAPI) DeleteQueue(in *sqs.DeleteQueueInput) (*sqs.DeleteQueueOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.DeleteQueueOutput), SafeError(rets[1])
}

func (s *SQSAPI) GetQueueAttributesRequest(in *sqs.GetQueueAttributesInput) (*request.Request, *sqs.GetQueueAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.GetQueueAttributesOutput)
}

func (s *SQSAPI) GetQueueAttributes(in *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.GetQueueAttributesOutput), SafeError(rets[1])
}

func (s *SQSAPI) GetQueueUrlRequest(in *sqs.GetQueueUrlInput) (*request.Request, *sqs.GetQueueUrlOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.GetQueueUrlOutput)
}

func (s *SQSAPI) GetQueueUrl(in *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.GetQueueUrlOutput), SafeError(rets[1])
}

func (s *SQSAPI) ListDeadLetterSourceQueuesRequest(in *sqs.ListDeadLetterSourceQueuesInput) (*request.Request, *sqs.ListDeadLetterSourceQueuesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ListDeadLetterSourceQueuesOutput)
}

func (s *SQSAPI) ListDeadLetterSourceQueues(in *sqs.ListDeadLetterSourceQueuesInput) (*sqs.ListDeadLetterSourceQueuesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ListDeadLetterSourceQueuesOutput), SafeError(rets[1])
}

func (s *SQSAPI) ListQueuesRequest(in *sqs.ListQueuesInput) (*request.Request, *sqs.ListQueuesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ListQueuesOutput)
}

func (s *SQSAPI) ListQueues(in *sqs.ListQueuesInput) (*sqs.ListQueuesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ListQueuesOutput), SafeError(rets[1])
}

func (s *SQSAPI) PurgeQueueRequest(in *sqs.PurgeQueueInput) (*request.Request, *sqs.PurgeQueueOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.PurgeQueueOutput)
}

func (s *SQSAPI) PurgeQueue(in *sqs.PurgeQueueInput) (*sqs.PurgeQueueOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.PurgeQueueOutput), SafeError(rets[1])
}

func (s *SQSAPI) ReceiveMessageRequest(in *sqs.ReceiveMessageInput) (*request.Request, *sqs.ReceiveMessageOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ReceiveMessageOutput)
}

func (s *SQSAPI) ReceiveMessage(in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ReceiveMessageOutput), SafeError(rets[1])
}

func (s *SQSAPI) RemovePermissionRequest(in *sqs.RemovePermissionInput) (*request.Request, *sqs.RemovePermissionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.RemovePermissionOutput)
}

func (s *SQSAPI) RemovePermission(in *sqs.RemovePermissionInput) (*sqs.RemovePermissionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.RemovePermissionOutput), SafeError(rets[1])
}

func (s *SQSAPI) SendMessageRequest(in *sqs.SendMessageInput) (*request.Request, *sqs.SendMessageOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.SendMessageOutput)
}

func (s *SQSAPI) SendMessage(in *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.SendMessageOutput), SafeError(rets[1])
}

func (s *SQSAPI) SendMessageBatchRequest(in *sqs.SendMessageBatchInput) (*request.Request, *sqs.SendMessageBatchOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.SendMessageBatchOutput)
}

func (s *SQSAPI) SendMessageBatch(in *sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.SendMessageBatchOutput), SafeError(rets[1])
}

func (s *SQSAPI) SetQueueAttributesRequest(in *sqs.SetQueueAttributesInput) (*request.Request, *sqs.SetQueueAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.SetQueueAttributesOutput)
}

func (s *SQSAPI) SetQueueAttributes(in *sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.SetQueueAttributesOutput), SafeError(rets[1])
}
