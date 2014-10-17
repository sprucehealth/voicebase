package sns

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
)

const (
	createPlatformEndpoint = "CreatePlatformEndpoint"
	deleteEndpoint         = "DeleteEndpoint"
	publish                = "Publish"
	subscribe              = "Subscribe"
	version                = "2010-03-31"
)

type createPlatformEndpointResponse struct {
	XMLName     xml.Name `xml:"CreatePlatformEndpointResponse"`
	EndpointArn string   `xml:"CreatePlatformEndpointResult>EndpointArn"`
}

type publishResponse struct {
	XMLName   xml.Name `xml:"PublishResponse"`
	MessageId string   `xml:"PublishResult>MessageId"`
}

type subscripeResponse struct {
	XMLName         xml.Name `xml:"SubscribeResponse"`
	SubscriptionArn string   `xml:"SubscribeResult>SubscriptionArn"`
}

func (sns *SNS) makeRequest(action string, args url.Values, response interface{}) error {
	// common parameters
	args.Set("Version", version)
	args.Set("Action", action)

	req, err := http.NewRequest("POST", sns.Region.SNSEndpoint, strings.NewReader(args.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := sns.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		snsError := &SNSError{
			HTTPStatusCode: res.StatusCode,
		}
		if err := xml.NewDecoder(res.Body).Decode(snsError); err != nil {
			return err
		}
		return snsError
	}

	if response != nil {
		return xml.NewDecoder(res.Body).Decode(response)
	}

	return err
}
