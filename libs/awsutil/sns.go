package awsutil

import (
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/libs/crypt"
	"github.com/sprucehealth/backend/libs/errors"
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
