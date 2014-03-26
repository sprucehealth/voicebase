package common

import (
	"carefront/libs/aws"
	"carefront/libs/aws/sqs"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"

	goamz "launchpad.net/goamz/aws"
)

func GenerateToken() (string, error) {
	tokBytes := make([]byte, 16)
	if _, err := rand.Read(tokBytes); err != nil {
		return "", err
	}

	tok := base64.URLEncoding.EncodeToString(tokBytes)
	return tok, nil
}

func AWSAuthAdapter(auth aws.Auth) goamz.Auth {
	keys := auth.Keys()
	return goamz.Auth{
		AccessKey: keys.AccessKey,
		SecretKey: keys.SecretKey,
		Token:     keys.Token,
	}
}

type StatusEventCheckType int64

const (
	ERxType StatusEventCheckType = iota
	RefillRxType
	UnlinkedDNTFTreatmentType
)

type PrescriptionStatusCheckMessage struct {
	PatientId      int64
	DoctorId       int64
	EventCheckType StatusEventCheckType
}

type SQSQueue struct {
	QueueService sqs.SQSService
	QueueUrl     string
}

func NewQueue(auth aws.Auth, region aws.Region, queueName string) (*SQSQueue, error) {
	awsClient := &aws.Client{
		Auth: auth,
	}

	sq := &sqs.SQS{
		Region: region,
		Client: awsClient,
	}

	queueUrl, err := sq.GetQueueUrl(queueName, "")
	if err != nil {
		return nil, err
	}

	return &SQSQueue{
		QueueService: sq,
		QueueUrl:     queueUrl,
	}, nil
}

// This is an object used for the (un)marshalling
// of data models ids, such that null values passed from the client
// can be treated as 0 values.
type ObjectId int64

func (id *ObjectId) UnmarshalJSON(data []byte) error {

	strData := string(data)
	// only treating the case of an empty string or a null value
	// as value being 0.
	// otherwise relying on integer parser
	if len(data) < 2 || strData == "null" || strData == `""` {
		*id = 0
		return nil
	}
	intId, err := strconv.ParseInt(strData[1:len(strData)-1], 10, 64)
	*id = ObjectId(intId)
	return err
}

func (id *ObjectId) MarshalJSON() ([]byte, error) {
	if id == nil {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d"`, *id)), nil
}

func NewObjectId(intId int64) *ObjectId {
	objectId := ObjectId(intId)
	return &objectId
}

func (id *ObjectId) Int64() int64 {
	if id == nil {
		return 0
	}
	return int64(*id)
}
