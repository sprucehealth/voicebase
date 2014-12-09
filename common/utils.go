package common

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"math/big"
	"os"

	goamz "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/mitchellh/goamz/aws"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/sqs"
)

const MinimumTokenLength = 20

// Any structure that implements the Typed interface
// requires a string that defines the type of the structure
type Typed interface {
	TypeName() string
}

type TypedData struct {
	Data []byte
	Type string
}

type ClientView interface {
	Validate() error
}

func (t *TypedData) TypeName() string {
	return t.Type
}

func GenerateToken() (string, error) {
	// REMINDER: Update MinimumTokenLength if this function changes
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

type ERxSourceType int64

const (
	ERxType ERxSourceType = iota
	RefillRxType
	UnlinkedDNTFTreatmentType
)

type PrescriptionStatusCheckMessage struct {
	PatientID      int64
	DoctorID       int64
	EventCheckType ERxSourceType
}

type SQSQueue struct {
	QueueService sqs.SQSService
	QueueURL     string
}

func NewQueue(auth aws.Auth, region aws.Region, queueName string) (*SQSQueue, error) {
	awsClient := &aws.Client{
		Auth: auth,
	}

	sq := &sqs.SQS{
		Region: region,
		Client: awsClient,
	}

	queueURL, err := sq.GetQueueURL(queueName, "")
	if err != nil {
		return nil, err
	}

	return &SQSQueue{
		QueueService: sq,
		QueueURL:     queueURL,
	}, nil
}

func SeekerSize(sk io.Seeker) (int64, error) {
	size, err := sk.Seek(0, os.SEEK_END)
	if err != nil {
		return 0, err
	}
	_, err = sk.Seek(0, os.SEEK_SET)
	return size, err
}

func GenerateRandomNumber(maxNum int64, maxDigits int) (string, error) {
	bigRandNum, err := rand.Int(rand.Reader, big.NewInt(maxNum))
	if err != nil {
		return "", err
	}
	randNum := bigRandNum.String()
	for len(randNum) < maxDigits {
		randNum = "0" + randNum
	}
	return randNum, nil

}
