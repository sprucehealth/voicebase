package common

import (
	"carefront/libs/aws"
	"carefront/libs/aws/sqs"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	goamz "launchpad.net/goamz/aws"
)

// Any structure that implements the Typed interface
// requires a string that defines the type of the structure
type Typed interface {
	TypeName() string
}

type TypedData struct {
	Data []byte
	Type string
}

func (t *TypedData) TypeName() string {
	return t.Type
}

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

type ClientHeaders struct {
	AppType      string
	AppEnv       string
	AppVersion   string
	AppBuild     string
	OS           string
	OSVersion    string
	DeviceType   string // Phone | Tablet
	DeviceModel  string
	ScreenWidth  int
	ScreenHeight int
	DPI          int
	Scale        float64
	DeviceID     string
}

func ParseClientHeaders(r *http.Request) *ClientHeaders {
	h := &ClientHeaders{}

	// S-Version: [Type];[Env];[Version];[Build]
	parts := strings.Split(r.Header.Get("S-Version"), ";")
	if len(parts) >= 1 {
		h.AppType = parts[0]
	}
	if len(parts) >= 2 {
		h.AppEnv = parts[1]
	}
	if len(parts) >= 3 {
		h.AppVersion = parts[2]
	}
	if len(parts) >= 4 {
		h.AppBuild = parts[3]
	}

	// S-OS: [OS];[OS Version]
	parts = strings.Split(r.Header.Get("S-OS"), ";")
	if len(parts) >= 1 {
		h.OS = parts[0]
	}
	if len(parts) >= 2 {
		h.OSVersion = parts[1]
	}

	// S-Device: [Phone|Tablet];[Model];[Screen Width];[Screen Height];[DPI/Scale*]
	parts = strings.Split(r.Header.Get("S-Device"), ";")
	if len(parts) >= 1 {
		h.DeviceType = parts[0]
	}
	if len(parts) >= 2 {
		h.DeviceModel = parts[1]
	}
	if len(parts) >= 3 {
		if i, err := strconv.Atoi(parts[2]); err == nil {
			h.ScreenWidth = i
		}
	}
	if len(parts) >= 4 {
		if i, err := strconv.Atoi(parts[3]); err == nil {
			h.ScreenHeight = i
		}
	}
	if len(parts) >= 5 {
		if h.OS == "iOS" {
			if f, err := strconv.ParseFloat(parts[4], 32); err == nil {
				h.Scale = f
			}
		} else {
			if i, err := strconv.Atoi(parts[4]); err == nil {
				h.DPI = i
			}
		}
	}

	// S-Device-ID: [Device ID]
	h.DeviceID = r.Header.Get("S-Device-ID")

	return h
}
