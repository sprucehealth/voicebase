package common

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

const MinimumTokenLength = 20

// Typed is an interface implemented by structs that can return their own type name
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
	tok := strings.TrimRight(base64.URLEncoding.EncodeToString(tokBytes), "=")
	return tok, nil
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
	QueueService sqsiface.SQSAPI
	QueueURL     string
}

func NewQueue(config *aws.Config, queueName string) (*SQSQueue, error) {
	sq := sqs.New(config)
	res, err := sq.GetQueueURL(&sqs.GetQueueURLInput{QueueName: &queueName})
	if err != nil {
		return nil, err
	}
	return &SQSQueue{
		QueueService: sq,
		QueueURL:     *res.QueueURL,
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

func Initials(first, last string) string {
	var out string
	if len(first) > 0 {
		out += string(first[0])
	}
	if len(last) > 0 {
		out += string(last[0])
	}
	return out
}
