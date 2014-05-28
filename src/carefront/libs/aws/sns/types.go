package sns

import (
	"encoding/xml"
	"fmt"
	"time"
)

type SQSMessage struct {
	Type             string
	MessageId        string
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
	XMLName   xml.Name `xml:"ErrorResponse"`
	Type      string   `xml:"Error>Type"`
	Code      string   `xml:"Error>Code"`
	Message   string   `xml:"Error>Message"`
	RequestId string   `xml:"Error>RequestId"`
}

func (s *SNSError) Error() string {
	return fmt.Sprintf("SNS Error:\n -Code:%s\n -Message:%s", s.Code, s.Message)
}
