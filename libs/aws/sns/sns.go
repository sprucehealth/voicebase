package sns

import (
	"encoding/json"
	"net/url"

	"github.com/sprucehealth/backend/libs/aws"
)

type SNSService interface {
	CreatePlatformEndpoint(string, string) (string, error)
	DeleteEndpoint(string) error
	Publish(interface{}, string) error
	SubscribePlatformEndpointToTopic(string, string) error
}

type SNS struct {
	aws.Region
	Client *aws.Client
}

func (sns *SNS) CreatePlatformEndpoint(platformEndpointArn, token string) (string, error) {
	args := url.Values{}
	args.Set("PlatformApplicationArn", platformEndpointArn)
	args.Set("Token", token)

	response := &createPlatformEndpointResponse{}
	err := sns.makeRequest(createPlatformEndpoint, args, response)
	return response.EndpointArn, err
}

func (sns *SNS) DeleteEndpoint(endpointArn string) error {
	args := url.Values{}
	args.Set("EndpointArn", endpointArn)

	return sns.makeRequest(deleteEndpoint, args, nil)
}

func (sns *SNS) Publish(message interface{}, targetArn string) error {
	args := url.Values{}
	args.Set("TargetArn", targetArn)
	args.Set("MessageStructure", "json")

	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}
	args.Set("Message", string(jsonData))

	err = sns.makeRequest(publish, args, nil)
	return err
}

func (sns *SNS) SubscribePlatformEndpointToTopic(platformEndpointArn, topicArn string) error {
	args := url.Values{}
	args.Set("Endpoint", platformEndpointArn)
	args.Set("Protocol", "application")
	args.Set("TopicArn", topicArn)

	err := sns.makeRequest(subscribe, args, nil)
	return err
}
