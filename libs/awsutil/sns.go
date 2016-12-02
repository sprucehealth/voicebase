package awsutil

import (
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/libs/crypt"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

const encryptedSNSPublishedMessage = "encrypted-sns"

type encryptedSNS struct {
	snsiface.SNSAPI
	encryptor crypt.Encrypter
}

// NewEncryptedSNS returns an initialized instance of encryptedSNS
func NewEncryptedSNS(masterKeyARN string, kms kmsiface.KMSAPI, sns snsiface.SNSAPI) (snsiface.SNSAPI, error) {
	kmsEncrypter, err := NewKMSEncrypter(masterKeyARN, kms)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize KMS encrypter: %s", err)
	}
	return &encryptedSNS{
		SNSAPI:    sns,
		encryptor: kmsEncrypter,
	}, nil
}

func (e *encryptedSNS) Publish(in *sns.PublishInput) (*sns.PublishOutput, error) {
	if in.MessageStructure != nil {
		return nil, errors.Trace(errors.New("encrypted SNS can only publish messages without structure"))
	}
	eMessage, err := e.encryptor.Encrypt([]byte(*in.Message))
	if err != nil {
		return nil, err
	}
	in.Message = ptr.String(base64.StdEncoding.EncodeToString(eMessage))
	if in.MessageAttributes == nil {
		in.MessageAttributes = make(map[string]*sns.MessageAttributeValue)
	}
	in.MessageAttributes[encryptedSNSPublishedMessage] = &sns.MessageAttributeValue{
		DataType:    ptr.String("String"),
		StringValue: ptr.String(encryptedSNSPublishedMessage),
	}
	return e.SNSAPI.Publish(in)
}

type marshaller interface {
	Marshal() ([]byte, error)
}

func PublishToSNSTopic(snsCLI snsiface.SNSAPI, topic string, m marshaller) error {
	data, err := m.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = snsCLI.Publish(&sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(data)),
		TopicArn: ptr.String(topic),
	})
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// CreateSNSTopic returns the ARN of the created topic
func CreateSNSTopic(snsCLI snsiface.SNSAPI, topicName string) (string, error) {
	createResp, err := snsCLI.CreateTopic(&sns.CreateTopicInput{
		Name: &topicName,
	})
	if err != nil {
		return "", errors.Trace(err)
	}
	return *createResp.TopicArn, nil
}

const (
	// AWSErrCodeSNSTopicNotFound the code returned from AWS when a topic isn't found
	AWSErrCodeSNSTopicNotFound = "NotFound"
)

// CreateSNSTopicIfNotExists returns the ARN of the existing or created topic
func CreateSNSTopicIfNotExists(snsCLI snsiface.SNSAPI, topicARN string) (string, error) {
	_, err := snsCLI.GetTopicAttributes(&sns.GetTopicAttributesInput{
		TopicArn: &topicARN,
	})
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == AWSErrCodeSNSTopicNotFound {
			topicName, err := ResourceNameFromARN(topicARN)
			if err != nil {
				return "", errors.Errorf("topic %s NOT FOUND. Unable to get topic name from ARN to create due to: %s", topicARN, err)
			}
			golog.Infof("Topic %s was NOT FOUND. Attempting to create it.", topicARN)
			topicARN, err = CreateSNSTopic(snsCLI, topicName)
			if err != nil {
				return "", errors.Errorf("topic %s NOT FOUND. Failed to create topic due to: %s", topicARN, err)
			}
			golog.Infof("Topic %s was successfully created", topicARN)
		} else {
			return "", errors.Errorf("failed to get attributes of topic %s: %s", topicARN, err)
		}
	} else if err != nil {
		return "", errors.Errorf("failed to get AWS error for GetTopicAttributes topic %s: %s", topicARN, err)
	}
	return topicARN, nil
}
