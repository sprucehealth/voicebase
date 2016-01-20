package mock

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

type mockSQSAPI struct {
	*Expector
}

var _ sqsiface.SQSAPI = NewMockSQSAPI(nil)

// NewMockSQSAPI returns a mock compatible SQSAPI instance
func NewMockSQSAPI(t *testing.T) *mockSQSAPI {
	return &mockSQSAPI{&Expector{T: t}}
}

func (s *mockSQSAPI) AddPermissionRequest(in *sqs.AddPermissionInput) (*request.Request, *sqs.AddPermissionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.AddPermissionOutput)
}

func (s *mockSQSAPI) AddPermission(in *sqs.AddPermissionInput) (*sqs.AddPermissionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.AddPermissionOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) ChangeMessageVisibilityRequest(in *sqs.ChangeMessageVisibilityInput) (*request.Request, *sqs.ChangeMessageVisibilityOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ChangeMessageVisibilityOutput)
}

func (s *mockSQSAPI) ChangeMessageVisibility(in *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ChangeMessageVisibilityOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) ChangeMessageVisibilityBatchRequest(in *sqs.ChangeMessageVisibilityBatchInput) (*request.Request, *sqs.ChangeMessageVisibilityBatchOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ChangeMessageVisibilityBatchOutput)
}

func (s *mockSQSAPI) ChangeMessageVisibilityBatch(in *sqs.ChangeMessageVisibilityBatchInput) (*sqs.ChangeMessageVisibilityBatchOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ChangeMessageVisibilityBatchOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) CreateQueueRequest(in *sqs.CreateQueueInput) (*request.Request, *sqs.CreateQueueOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.CreateQueueOutput)
}

func (s *mockSQSAPI) CreateQueue(in *sqs.CreateQueueInput) (*sqs.CreateQueueOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.CreateQueueOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) DeleteMessageRequest(in *sqs.DeleteMessageInput) (*request.Request, *sqs.DeleteMessageOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.DeleteMessageOutput)
}

func (s *mockSQSAPI) DeleteMessage(in *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.DeleteMessageOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) DeleteMessageBatchRequest(in *sqs.DeleteMessageBatchInput) (*request.Request, *sqs.DeleteMessageBatchOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.DeleteMessageBatchOutput)
}

func (s *mockSQSAPI) DeleteMessageBatch(in *sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.DeleteMessageBatchOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) DeleteQueueRequest(in *sqs.DeleteQueueInput) (*request.Request, *sqs.DeleteQueueOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.DeleteQueueOutput)
}

func (s *mockSQSAPI) DeleteQueue(in *sqs.DeleteQueueInput) (*sqs.DeleteQueueOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.DeleteQueueOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) GetQueueAttributesRequest(in *sqs.GetQueueAttributesInput) (*request.Request, *sqs.GetQueueAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.GetQueueAttributesOutput)
}

func (s *mockSQSAPI) GetQueueAttributes(in *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.GetQueueAttributesOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) GetQueueUrlRequest(in *sqs.GetQueueUrlInput) (*request.Request, *sqs.GetQueueUrlOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.GetQueueUrlOutput)
}

func (s *mockSQSAPI) GetQueueUrl(in *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.GetQueueUrlOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) ListDeadLetterSourceQueuesRequest(in *sqs.ListDeadLetterSourceQueuesInput) (*request.Request, *sqs.ListDeadLetterSourceQueuesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ListDeadLetterSourceQueuesOutput)
}

func (s *mockSQSAPI) ListDeadLetterSourceQueues(in *sqs.ListDeadLetterSourceQueuesInput) (*sqs.ListDeadLetterSourceQueuesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ListDeadLetterSourceQueuesOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) ListQueuesRequest(in *sqs.ListQueuesInput) (*request.Request, *sqs.ListQueuesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ListQueuesOutput)
}

func (s *mockSQSAPI) ListQueues(in *sqs.ListQueuesInput) (*sqs.ListQueuesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ListQueuesOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) PurgeQueueRequest(in *sqs.PurgeQueueInput) (*request.Request, *sqs.PurgeQueueOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.PurgeQueueOutput)
}

func (s *mockSQSAPI) PurgeQueue(in *sqs.PurgeQueueInput) (*sqs.PurgeQueueOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.PurgeQueueOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) ReceiveMessageRequest(in *sqs.ReceiveMessageInput) (*request.Request, *sqs.ReceiveMessageOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.ReceiveMessageOutput)
}

func (s *mockSQSAPI) ReceiveMessage(in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.ReceiveMessageOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) RemovePermissionRequest(in *sqs.RemovePermissionInput) (*request.Request, *sqs.RemovePermissionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.RemovePermissionOutput)
}

func (s *mockSQSAPI) RemovePermission(in *sqs.RemovePermissionInput) (*sqs.RemovePermissionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.RemovePermissionOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) SendMessageRequest(in *sqs.SendMessageInput) (*request.Request, *sqs.SendMessageOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.SendMessageOutput)
}

func (s *mockSQSAPI) SendMessage(in *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.SendMessageOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) SendMessageBatchRequest(in *sqs.SendMessageBatchInput) (*request.Request, *sqs.SendMessageBatchOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.SendMessageBatchOutput)
}

func (s *mockSQSAPI) SendMessageBatch(in *sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.SendMessageBatchOutput), SafeError(rets[1])
}

func (s *mockSQSAPI) SetQueueAttributesRequest(in *sqs.SetQueueAttributesInput) (*request.Request, *sqs.SetQueueAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sqs.SetQueueAttributesOutput)
}

func (s *mockSQSAPI) SetQueueAttributes(in *sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sqs.SetQueueAttributesOutput), SafeError(rets[1])
}
