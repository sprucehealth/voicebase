package sns

import (
	"encoding/xml"
	"fmt"
	"time"
)

type SQSMessage struct {
	Type             string
	MessageID        string `xml:"MessageId" json:"MessageId"`
	TopicArn         string
	Subject          string
	Message          string
	Timestamp        time.Time
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	UnsubscribeURL   string
}

type SNSError struct {
	XMLName        xml.Name `xml:"ErrorResponse"`
	Type           string   `xml:"Error>Type"`
	Code           string   `xml:"Error>Code"`
	Message        string   `xml:"Error>Message"`
	RequestID      string   `xml:"Error>RequestId"`
	HTTPStatusCode int
}

func (s *SNSError) Error() string {
	return fmt.Sprintf("SNS Type=%s Code=%s HTTPStatusCode=%d: %s", s.Type, s.Code, s.HTTPStatusCode, s.Message)
}
