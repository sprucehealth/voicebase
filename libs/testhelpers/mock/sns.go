package mock

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
)

type mockSNSAPI struct {
	*Expector
}

var _ snsiface.SNSAPI = NewMockSNSAPI(nil)

// NewMockSNSAPI returns a mock compatible SNSAPI instance
func NewMockSNSAPI(t *testing.T) *mockSNSAPI {
	return &mockSNSAPI{&Expector{T: t}}
}

func (s *mockSNSAPI) AddPermissionRequest(in *sns.AddPermissionInput) (*request.Request, *sns.AddPermissionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.AddPermissionOutput)
}

func (s *mockSNSAPI) AddPermission(in *sns.AddPermissionInput) (*sns.AddPermissionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.AddPermissionOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) ConfirmSubscriptionRequest(in *sns.ConfirmSubscriptionInput) (*request.Request, *sns.ConfirmSubscriptionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ConfirmSubscriptionOutput)
}

func (s *mockSNSAPI) ConfirmSubscription(in *sns.ConfirmSubscriptionInput) (*sns.ConfirmSubscriptionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ConfirmSubscriptionOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) CreatePlatformApplicationRequest(in *sns.CreatePlatformApplicationInput) (*request.Request, *sns.CreatePlatformApplicationOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.CreatePlatformApplicationOutput)
}

func (s *mockSNSAPI) CreatePlatformApplication(in *sns.CreatePlatformApplicationInput) (*sns.CreatePlatformApplicationOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.CreatePlatformApplicationOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) CreatePlatformEndpointRequest(in *sns.CreatePlatformEndpointInput) (*request.Request, *sns.CreatePlatformEndpointOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.CreatePlatformEndpointOutput)
}

func (s *mockSNSAPI) CreatePlatformEndpoint(in *sns.CreatePlatformEndpointInput) (*sns.CreatePlatformEndpointOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.CreatePlatformEndpointOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) CreateTopicRequest(in *sns.CreateTopicInput) (*request.Request, *sns.CreateTopicOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.CreateTopicOutput)
}

func (s *mockSNSAPI) CreateTopic(in *sns.CreateTopicInput) (*sns.CreateTopicOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.CreateTopicOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) DeleteEndpointRequest(in *sns.DeleteEndpointInput) (*request.Request, *sns.DeleteEndpointOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.DeleteEndpointOutput)
}

func (s *mockSNSAPI) DeleteEndpoint(in *sns.DeleteEndpointInput) (*sns.DeleteEndpointOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.DeleteEndpointOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) DeletePlatformApplicationRequest(in *sns.DeletePlatformApplicationInput) (*request.Request, *sns.DeletePlatformApplicationOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.DeletePlatformApplicationOutput)
}

func (s *mockSNSAPI) DeletePlatformApplication(in *sns.DeletePlatformApplicationInput) (*sns.DeletePlatformApplicationOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.DeletePlatformApplicationOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) DeleteTopicRequest(in *sns.DeleteTopicInput) (*request.Request, *sns.DeleteTopicOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.DeleteTopicOutput)
}

func (s *mockSNSAPI) DeleteTopic(in *sns.DeleteTopicInput) (*sns.DeleteTopicOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.DeleteTopicOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) GetEndpointAttributesRequest(in *sns.GetEndpointAttributesInput) (*request.Request, *sns.GetEndpointAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.GetEndpointAttributesOutput)
}

func (s *mockSNSAPI) GetEndpointAttributes(in *sns.GetEndpointAttributesInput) (*sns.GetEndpointAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.GetEndpointAttributesOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) GetPlatformApplicationAttributesRequest(in *sns.GetPlatformApplicationAttributesInput) (*request.Request, *sns.GetPlatformApplicationAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.GetPlatformApplicationAttributesOutput)
}

func (s *mockSNSAPI) GetPlatformApplicationAttributes(in *sns.GetPlatformApplicationAttributesInput) (*sns.GetPlatformApplicationAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.GetPlatformApplicationAttributesOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) GetSubscriptionAttributesRequest(in *sns.GetSubscriptionAttributesInput) (*request.Request, *sns.GetSubscriptionAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.GetSubscriptionAttributesOutput)
}

func (s *mockSNSAPI) GetSubscriptionAttributes(in *sns.GetSubscriptionAttributesInput) (*sns.GetSubscriptionAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.GetSubscriptionAttributesOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) GetTopicAttributesRequest(in *sns.GetTopicAttributesInput) (*request.Request, *sns.GetTopicAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.GetTopicAttributesOutput)
}

func (s *mockSNSAPI) GetTopicAttributes(in *sns.GetTopicAttributesInput) (*sns.GetTopicAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.GetTopicAttributesOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) ListEndpointsByPlatformApplicationRequest(in *sns.ListEndpointsByPlatformApplicationInput) (*request.Request, *sns.ListEndpointsByPlatformApplicationOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListEndpointsByPlatformApplicationOutput)
}

func (s *mockSNSAPI) ListEndpointsByPlatformApplication(in *sns.ListEndpointsByPlatformApplicationInput) (*sns.ListEndpointsByPlatformApplicationOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListEndpointsByPlatformApplicationOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) ListEndpointsByPlatformApplicationPages(in *sns.ListEndpointsByPlatformApplicationInput, f func(*sns.ListEndpointsByPlatformApplicationOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *mockSNSAPI) ListPlatformApplicationsRequest(in *sns.ListPlatformApplicationsInput) (*request.Request, *sns.ListPlatformApplicationsOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListPlatformApplicationsOutput)
}

func (s *mockSNSAPI) ListPlatformApplications(in *sns.ListPlatformApplicationsInput) (*sns.ListPlatformApplicationsOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListPlatformApplicationsOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) ListPlatformApplicationsPages(in *sns.ListPlatformApplicationsInput, f func(*sns.ListPlatformApplicationsOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *mockSNSAPI) ListSubscriptionsRequest(in *sns.ListSubscriptionsInput) (*request.Request, *sns.ListSubscriptionsOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListSubscriptionsOutput)
}

func (s *mockSNSAPI) ListSubscriptions(in *sns.ListSubscriptionsInput) (*sns.ListSubscriptionsOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListSubscriptionsOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) ListSubscriptionsPages(in *sns.ListSubscriptionsInput, f func(*sns.ListSubscriptionsOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *mockSNSAPI) ListSubscriptionsByTopicRequest(in *sns.ListSubscriptionsByTopicInput) (*request.Request, *sns.ListSubscriptionsByTopicOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListSubscriptionsByTopicOutput)
}

func (s *mockSNSAPI) ListSubscriptionsByTopic(in *sns.ListSubscriptionsByTopicInput) (*sns.ListSubscriptionsByTopicOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListSubscriptionsByTopicOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) ListSubscriptionsByTopicPages(in *sns.ListSubscriptionsByTopicInput, f func(*sns.ListSubscriptionsByTopicOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *mockSNSAPI) ListTopicsRequest(in *sns.ListTopicsInput) (*request.Request, *sns.ListTopicsOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.ListTopicsOutput)
}

func (s *mockSNSAPI) ListTopics(in *sns.ListTopicsInput) (*sns.ListTopicsOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.ListTopicsOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) ListTopicsPages(in *sns.ListTopicsInput, f func(*sns.ListTopicsOutput, bool) bool) error {
	rets := s.Record(in, f)
	if len(rets) == 0 {
		return nil
	}
	return SafeError(rets[0])
}

func (s *mockSNSAPI) PublishRequest(in *sns.PublishInput) (*request.Request, *sns.PublishOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.PublishOutput)
}

func (s *mockSNSAPI) Publish(in *sns.PublishInput) (*sns.PublishOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.PublishOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) RemovePermissionRequest(in *sns.RemovePermissionInput) (*request.Request, *sns.RemovePermissionOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.RemovePermissionOutput)
}

func (s *mockSNSAPI) RemovePermission(in *sns.RemovePermissionInput) (*sns.RemovePermissionOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.RemovePermissionOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) SetEndpointAttributesRequest(in *sns.SetEndpointAttributesInput) (*request.Request, *sns.SetEndpointAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SetEndpointAttributesOutput)
}

func (s *mockSNSAPI) SetEndpointAttributes(in *sns.SetEndpointAttributesInput) (*sns.SetEndpointAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SetEndpointAttributesOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) SetPlatformApplicationAttributesRequest(in *sns.SetPlatformApplicationAttributesInput) (*request.Request, *sns.SetPlatformApplicationAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SetPlatformApplicationAttributesOutput)
}

func (s *mockSNSAPI) SetPlatformApplicationAttributes(in *sns.SetPlatformApplicationAttributesInput) (*sns.SetPlatformApplicationAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SetPlatformApplicationAttributesOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) SetSubscriptionAttributesRequest(in *sns.SetSubscriptionAttributesInput) (*request.Request, *sns.SetSubscriptionAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SetSubscriptionAttributesOutput)
}

func (s *mockSNSAPI) SetSubscriptionAttributes(in *sns.SetSubscriptionAttributesInput) (*sns.SetSubscriptionAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SetSubscriptionAttributesOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) SetTopicAttributesRequest(in *sns.SetTopicAttributesInput) (*request.Request, *sns.SetTopicAttributesOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SetTopicAttributesOutput)
}

func (s *mockSNSAPI) SetTopicAttributes(in *sns.SetTopicAttributesInput) (*sns.SetTopicAttributesOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SetTopicAttributesOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) SubscribeRequest(in *sns.SubscribeInput) (*request.Request, *sns.SubscribeOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.SubscribeOutput)
}

func (s *mockSNSAPI) Subscribe(in *sns.SubscribeInput) (*sns.SubscribeOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.SubscribeOutput), SafeError(rets[1])
}

func (s *mockSNSAPI) UnsubscribeRequest(in *sns.UnsubscribeInput) (*request.Request, *sns.UnsubscribeOutput) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*request.Request), rets[1].(*sns.UnsubscribeOutput)
}

func (s *mockSNSAPI) Unsubscribe(in *sns.UnsubscribeInput) (*sns.UnsubscribeOutput, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*sns.UnsubscribeOutput), SafeError(rets[1])
}
