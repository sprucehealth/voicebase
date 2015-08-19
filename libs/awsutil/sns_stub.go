package awsutil

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
)

type SNS struct {
	snsiface.SNSAPI
	EndpointARN string
}

func (*SNS) AddPermission(*sns.AddPermissionInput) (*sns.AddPermissionOutput, error) {
	return nil, nil
}
func (*SNS) ConfirmSubscription(*sns.ConfirmSubscriptionInput) (*sns.ConfirmSubscriptionOutput, error) {
	return nil, nil
}
func (*SNS) CreatePlatformApplication(*sns.CreatePlatformApplicationInput) (*sns.CreatePlatformApplicationOutput, error) {
	return nil, nil
}
func (s *SNS) CreatePlatformEndpoint(*sns.CreatePlatformEndpointInput) (*sns.CreatePlatformEndpointOutput, error) {
	return &sns.CreatePlatformEndpointOutput{EndpointArn: &s.EndpointARN}, nil
}
func (*SNS) CreateTopic(*sns.CreateTopicInput) (*sns.CreateTopicOutput, error) {
	return nil, nil
}
func (*SNS) DeleteEndpoint(*sns.DeleteEndpointInput) (*sns.DeleteEndpointOutput, error) {
	return nil, nil
}
func (*SNS) DeletePlatformApplication(*sns.DeletePlatformApplicationInput) (*sns.DeletePlatformApplicationOutput, error) {
	return nil, nil
}
func (*SNS) DeleteTopic(*sns.DeleteTopicInput) (*sns.DeleteTopicOutput, error) {
	return nil, nil
}
func (*SNS) GetEndpointAttributes(*sns.GetEndpointAttributesInput) (*sns.GetEndpointAttributesOutput, error) {
	return nil, nil
}
func (*SNS) GetPlatformApplicationAttributes(*sns.GetPlatformApplicationAttributesInput) (*sns.GetPlatformApplicationAttributesOutput, error) {
	return nil, nil
}
func (*SNS) GetSubscriptionAttributes(*sns.GetSubscriptionAttributesInput) (*sns.GetSubscriptionAttributesOutput, error) {
	return nil, nil
}
func (*SNS) GetTopicAttributes(*sns.GetTopicAttributesInput) (*sns.GetTopicAttributesOutput, error) {
	return nil, nil
}
func (*SNS) ListEndpointsByPlatformApplication(*sns.ListEndpointsByPlatformApplicationInput) (*sns.ListEndpointsByPlatformApplicationOutput, error) {
	return nil, nil
}
func (*SNS) ListPlatformApplications(*sns.ListPlatformApplicationsInput) (*sns.ListPlatformApplicationsOutput, error) {
	return nil, nil
}
func (*SNS) ListSubscriptions(*sns.ListSubscriptionsInput) (*sns.ListSubscriptionsOutput, error) {
	return nil, nil
}
func (*SNS) ListSubscriptionsByTopic(*sns.ListSubscriptionsByTopicInput) (*sns.ListSubscriptionsByTopicOutput, error) {
	return nil, nil
}
func (*SNS) ListTopics(*sns.ListTopicsInput) (*sns.ListTopicsOutput, error) {
	return nil, nil
}
func (*SNS) Publish(*sns.PublishInput) (*sns.PublishOutput, error) {
	return nil, nil
}
func (*SNS) RemovePermission(*sns.RemovePermissionInput) (*sns.RemovePermissionOutput, error) {
	return nil, nil
}
func (*SNS) SetEndpointAttributes(*sns.SetEndpointAttributesInput) (*sns.SetEndpointAttributesOutput, error) {
	return nil, nil
}
func (*SNS) SetPlatformApplicationAttributes(*sns.SetPlatformApplicationAttributesInput) (*sns.SetPlatformApplicationAttributesOutput, error) {
	return nil, nil
}
func (*SNS) SetSubscriptionAttributes(*sns.SetSubscriptionAttributesInput) (*sns.SetSubscriptionAttributesOutput, error) {
	return nil, nil
}
func (*SNS) SetTopicAttributes(*sns.SetTopicAttributesInput) (*sns.SetTopicAttributesOutput, error) {
	return nil, nil
}
func (*SNS) Subscribe(*sns.SubscribeInput) (*sns.SubscribeOutput, error) {
	return nil, nil
}
func (*SNS) Unsubscribe(*sns.UnsubscribeInput) (*sns.UnsubscribeOutput, error) {
	return nil, nil
}
