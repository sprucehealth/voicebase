package common

import (
	"carefront/libs/aws"
	"carefront/libs/aws/sqs"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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

type Dob struct {
	Month int
	Day   int
	Year  int
}

func (dob *Dob) UnmarshalJSON(data []byte) error {
	strDob := string(data)

	if len(data) < 2 || strDob == "null" || strDob == `""` {
		*dob = Dob{}
		return nil
	}

	// break up dob into components (of the format MM/DD/YYYY)
	dobParts := strings.Split(strDob, "/")

	if len(dobParts) < 3 {
		return errors.New("Dob incorrectly formatted. Expected format YYYY/MM/DD")
	}

	if len(dobParts[0]) != 5 || len(dobParts[1]) != 2 || len(dobParts[2]) != 3 {
		return errors.New("Dob incorrectly formatted. Expected format YYYY/MM/DD")
	}

	dobYear, err := strconv.Atoi(dobParts[0][1:]) // to remove the `"`
	if err != nil {
		return err
	}

	dobMonth, err := strconv.Atoi(dobParts[1])
	if err != nil {
		return err
	}

	dobDay, err := strconv.Atoi(dobParts[2][:len(dobParts[2])-1]) // to remove the `"`
	if err != nil {
		return err
	}

	*dob = Dob{
		Year:  dobYear,
		Month: dobMonth,
		Day:   dobDay,
	}

	return nil
}

func (dob *Dob) MarshalJSON() ([]byte, error) {
	if dob == nil {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d/%02d/%02d"`, dob.Year, dob.Month, dob.Day)), nil
}

func (dob *Dob) ToTime() time.Time {
	return time.Date(dob.Year, time.Month(dob.Month), dob.Day, 0, 0, 0, 0, time.UTC)
}

func NewDobFromTime(dobTime time.Time) Dob {
	dobYear, dobMonth, dobDay := dobTime.Date()
	dob := Dob{}
	dob.Month = int(dobMonth)
	dob.Year = dobYear
	dob.Day = dobDay
	return dob
}

func NewDobFromComponents(dobYear, dobMonth, dobDay string) (Dob, error) {
	var dob Dob
	var err error
	dob.Day, err = strconv.Atoi(dobDay)
	if err != nil {
		return dob, err
	}

	dob.Month, err = strconv.Atoi(dobMonth)
	if err != nil {
		return dob, err
	}

	dob.Year, err = strconv.Atoi(dobYear)
	if err != nil {
		return dob, err
	}

	return dob, nil
}
