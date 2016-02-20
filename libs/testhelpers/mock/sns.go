package mock

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
)

type MockSNSAPI struct {
	*Expector
}

var _ snsiface.SNSAPI = NewSNSAPI(nil)

// NewSNSAPI returns a mock compatible SNSAPI instance
func NewSNSAPI(t *testing.T) *MockSNSAPI {
	return &MockSNSAPI{&Expector{T: t}}
}

func (s *MockSNSAPI) AddPermissionRequest(in *sns.AddPermissionInput) (*request.Request, *sns.AddPermissionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.AddPermissionOutput)
}

func (s *MockSNSAPI) AddPermission(in *sns.AddPermissionInput) (*sns.AddPermissionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.AddPermissionOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) ConfirmSubscriptionRequest(in *sns.ConfirmSubscriptionInput) (*request.Request, *sns.ConfirmSubscriptionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ConfirmSubscriptionOutput)
}

func (s *MockSNSAPI) ConfirmSubscription(in *sns.ConfirmSubscriptionInput) (*sns.ConfirmSubscriptionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ConfirmSubscriptionOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) CreatePlatformApplicationRequest(in *sns.CreatePlatformApplicationInput) (*request.Request, *sns.CreatePlatformApplicationOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.CreatePlatformApplicationOutput)
}

func (s *MockSNSAPI) CreatePlatformApplication(in *sns.CreatePlatformApplicationInput) (*sns.CreatePlatformApplicationOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.CreatePlatformApplicationOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) CreatePlatformEndpointRequest(in *sns.CreatePlatformEndpointInput) (*request.Request, *sns.CreatePlatformEndpointOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.CreatePlatformEndpointOutput)
}

func (s *MockSNSAPI) CreatePlatformEndpoint(in *sns.CreatePlatformEndpointInput) (*sns.CreatePlatformEndpointOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.CreatePlatformEndpointOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) CreateTopicRequest(in *sns.CreateTopicInput) (*request.Request, *sns.CreateTopicOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.CreateTopicOutput)
}

func (s *MockSNSAPI) CreateTopic(in *sns.CreateTopicInput) (*sns.CreateTopicOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.CreateTopicOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) DeleteEndpointRequest(in *sns.DeleteEndpointInput) (*request.Request, *sns.DeleteEndpointOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.DeleteEndpointOutput)
}

func (s *MockSNSAPI) DeleteEndpoint(in *sns.DeleteEndpointInput) (*sns.DeleteEndpointOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.DeleteEndpointOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) DeletePlatformApplicationRequest(in *sns.DeletePlatformApplicationInput) (*request.Request, *sns.DeletePlatformApplicationOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.DeletePlatformApplicationOutput)
}

func (s *MockSNSAPI) DeletePlatformApplication(in *sns.DeletePlatformApplicationInput) (*sns.DeletePlatformApplicationOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.DeletePlatformApplicationOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) DeleteTopicRequest(in *sns.DeleteTopicInput) (*request.Request, *sns.DeleteTopicOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.DeleteTopicOutput)
}

func (s *MockSNSAPI) DeleteTopic(in *sns.DeleteTopicInput) (*sns.DeleteTopicOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.DeleteTopicOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) GetEndpointAttributesRequest(in *sns.GetEndpointAttributesInput) (*request.Request, *sns.GetEndpointAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.GetEndpointAttributesOutput)
}

func (s *MockSNSAPI) GetEndpointAttributes(in *sns.GetEndpointAttributesInput) (*sns.GetEndpointAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.GetEndpointAttributesOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) GetPlatformApplicationAttributesRequest(in *sns.GetPlatformApplicationAttributesInput) (*request.Request, *sns.GetPlatformApplicationAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.GetPlatformApplicationAttributesOutput)
}

func (s *MockSNSAPI) GetPlatformApplicationAttributes(in *sns.GetPlatformApplicationAttributesInput) (*sns.GetPlatformApplicationAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.GetPlatformApplicationAttributesOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) GetSubscriptionAttributesRequest(in *sns.GetSubscriptionAttributesInput) (*request.Request, *sns.GetSubscriptionAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.GetSubscriptionAttributesOutput)
}

func (s *MockSNSAPI) GetSubscriptionAttributes(in *sns.GetSubscriptionAttributesInput) (*sns.GetSubscriptionAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.GetSubscriptionAttributesOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) GetTopicAttributesRequest(in *sns.GetTopicAttributesInput) (*request.Request, *sns.GetTopicAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.GetTopicAttributesOutput)
}

func (s *MockSNSAPI) GetTopicAttributes(in *sns.GetTopicAttributesInput) (*sns.GetTopicAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.GetTopicAttributesOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) ListEndpointsByPlatformApplicationRequest(in *sns.ListEndpointsByPlatformApplicationInput) (*request.Request, *sns.ListEndpointsByPlatformApplicationOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListEndpointsByPlatformApplicationOutput)
}

func (s *MockSNSAPI) ListEndpointsByPlatformApplication(in *sns.ListEndpointsByPlatformApplicationInput) (*sns.ListEndpointsByPlatformApplicationOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListEndpointsByPlatformApplicationOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) ListEndpointsByPlatformApplicationPages(in *sns.ListEndpointsByPlatformApplicationInput, f func(*sns.ListEndpointsByPlatformApplicationOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *MockSNSAPI) ListPlatformApplicationsRequest(in *sns.ListPlatformApplicationsInput) (*request.Request, *sns.ListPlatformApplicationsOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListPlatformApplicationsOutput)
}

func (s *MockSNSAPI) ListPlatformApplications(in *sns.ListPlatformApplicationsInput) (*sns.ListPlatformApplicationsOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListPlatformApplicationsOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) ListPlatformApplicationsPages(in *sns.ListPlatformApplicationsInput, f func(*sns.ListPlatformApplicationsOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *MockSNSAPI) ListSubscriptionsRequest(in *sns.ListSubscriptionsInput) (*request.Request, *sns.ListSubscriptionsOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListSubscriptionsOutput)
}

func (s *MockSNSAPI) ListSubscriptions(in *sns.ListSubscriptionsInput) (*sns.ListSubscriptionsOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListSubscriptionsOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) ListSubscriptionsPages(in *sns.ListSubscriptionsInput, f func(*sns.ListSubscriptionsOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *MockSNSAPI) ListSubscriptionsByTopicRequest(in *sns.ListSubscriptionsByTopicInput) (*request.Request, *sns.ListSubscriptionsByTopicOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListSubscriptionsByTopicOutput)
}

func (s *MockSNSAPI) ListSubscriptionsByTopic(in *sns.ListSubscriptionsByTopicInput) (*sns.ListSubscriptionsByTopicOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListSubscriptionsByTopicOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) ListSubscriptionsByTopicPages(in *sns.ListSubscriptionsByTopicInput, f func(*sns.ListSubscriptionsByTopicOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *MockSNSAPI) ListTopicsRequest(in *sns.ListTopicsInput) (*request.Request, *sns.ListTopicsOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListTopicsOutput)
}

func (s *MockSNSAPI) ListTopics(in *sns.ListTopicsInput) (*sns.ListTopicsOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListTopicsOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) ListTopicsPages(in *sns.ListTopicsInput, f func(*sns.ListTopicsOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *MockSNSAPI) PublishRequest(in *sns.PublishInput) (*request.Request, *sns.PublishOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.PublishOutput)
}

func (s *MockSNSAPI) Publish(in *sns.PublishInput) (*sns.PublishOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.PublishOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) RemovePermissionRequest(in *sns.RemovePermissionInput) (*request.Request, *sns.RemovePermissionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.RemovePermissionOutput)
}

func (s *MockSNSAPI) RemovePermission(in *sns.RemovePermissionInput) (*sns.RemovePermissionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.RemovePermissionOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) SetEndpointAttributesRequest(in *sns.SetEndpointAttributesInput) (*request.Request, *sns.SetEndpointAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SetEndpointAttributesOutput)
}

func (s *MockSNSAPI) SetEndpointAttributes(in *sns.SetEndpointAttributesInput) (*sns.SetEndpointAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SetEndpointAttributesOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) SetPlatformApplicationAttributesRequest(in *sns.SetPlatformApplicationAttributesInput) (*request.Request, *sns.SetPlatformApplicationAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SetPlatformApplicationAttributesOutput)
}

func (s *MockSNSAPI) SetPlatformApplicationAttributes(in *sns.SetPlatformApplicationAttributesInput) (*sns.SetPlatformApplicationAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SetPlatformApplicationAttributesOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) SetSubscriptionAttributesRequest(in *sns.SetSubscriptionAttributesInput) (*request.Request, *sns.SetSubscriptionAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SetSubscriptionAttributesOutput)
}

func (s *MockSNSAPI) SetSubscriptionAttributes(in *sns.SetSubscriptionAttributesInput) (*sns.SetSubscriptionAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SetSubscriptionAttributesOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) SetTopicAttributesRequest(in *sns.SetTopicAttributesInput) (*request.Request, *sns.SetTopicAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SetTopicAttributesOutput)
}

func (s *MockSNSAPI) SetTopicAttributes(in *sns.SetTopicAttributesInput) (*sns.SetTopicAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SetTopicAttributesOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) SubscribeRequest(in *sns.SubscribeInput) (*request.Request, *sns.SubscribeOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SubscribeOutput)
}

func (s *MockSNSAPI) Subscribe(in *sns.SubscribeInput) (*sns.SubscribeOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SubscribeOutput), SafeError(rets[1])
}

func (s *MockSNSAPI) UnsubscribeRequest(in *sns.UnsubscribeInput) (*request.Request, *sns.UnsubscribeOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.UnsubscribeOutput)
}

func (s *MockSNSAPI) Unsubscribe(in *sns.UnsubscribeInput) (*sns.UnsubscribeOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.UnsubscribeOutput), SafeError(rets[1])
}
