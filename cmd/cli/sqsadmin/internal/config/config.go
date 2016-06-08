package config

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/boot"
)

// Config represents the structure passed to individual commands
type Config struct {
	App *boot.App
}

// SQSClient returns an SQS client mapped to the current session
func (c *Config) SQSClient() (sqsiface.SQSAPI, error) {
	awsSession, err := c.App.AWSSession()
	if err != nil {
		return nil, fmt.Errorf("Unable to create AWSSession: %s", err)
	}
	return sqs.New(awsSession), nil
}

// KMSClient returns an KMS client mapped to the current session
func (c *Config) KMSClient() (kmsiface.KMSAPI, error) {
	awsSession, err := c.App.AWSSession()
	if err != nil {
		return nil, fmt.Errorf("Unable to create AWSSession: %s", err)
	}
	return kms.New(awsSession), nil
}
