package awsutil

import (
	"bytes"
	"time"
)

// FlexibleBool is a type alias of bool that supports more flexible
// parsing of JSON. As well as supporting the standard true/false it
// also allows the value to be a string. It always marshals as the
// standard bool values.
type FlexibleBool bool

var (
	trueBytes  = []byte("true")
	falseBytes = []byte("false")
)

// MarshalJSON implements json.Marshaler. It encodes the values as
// the standard JSON bool types.
func (sb FlexibleBool) MarshalJSON() ([]byte, error) {
	if bool(sb) {
		return trueBytes, nil
	}
	return falseBytes, nil
}

// UnmarshalJSON implements json.Unmarshaler. It decodes values as
// either the standard JSON bool type or as strings "true" / "false".
func (sb *FlexibleBool) UnmarshalJSON(by []byte) error {
	*sb = false
	if len(by) < 4 {
		return nil
	}
	if by[0] == '"' {
		by = by[1 : len(by)-1]
	}
	if bytes.Equal(by, trueBytes) {
		*sb = true
	}
	return nil
}

type CloudTrailSNSNotification struct {
	S3Bucket    string   `json:"s3Bucket"`
	S3ObjectKey []string `json:"s3ObjectKey"`
}

type CloudTrailLog struct {
	Records []*CloudTrailRecord
}

type CloudTrailRecord struct {
	AWSRegion         string                  `json:"awsRegion"`
	ErrorCode         string                  `json:"errorCode"`
	ErrorMessage      string                  `json:"errorMessage"`
	EventName         string                  `json:"eventName"`
	EventSource       string                  `json:"eventSource"`
	EventTime         time.Time               `json:"eventTime"`
	EventVersion      string                  `json:"eventVersion"`
	RequestParameters map[string]interface{}  `json:"requestParameters"`
	ResponseElements  map[string]interface{}  `json:"responseElements"`
	SourceIPAddress   string                  `json:"sourceIPAddress"`
	UserAgent         string                  `json:"userAgent"`
	UserIdentity      *CloudTrailUserIdentity `json:"userIdentity"`
}

type CloudTrailUserIdentity struct {
	AccessKeyID    string                    `json:"accessKeyId"`
	AccountID      int64                     `json:"accountId,string"`
	ARN            string                    `json:"arn"`
	PrincipalID    string                    `json:"principalId"`
	SessionContext *CloudTrailSessionContext `json:"sessionContext,omitempty"`
	Type           string                    `json:"type"`
	UserName       string                    `json:"userName"`
}

type CloudTrailSessionContext struct {
	Attributes struct {
		CreationDate     time.Time    `json:"creationDate"`
		MFAAuthenticated FlexibleBool `json:"mfaAuthenticated"`
	} `json:"attributes"`
	SessionIssuer *CloudTrailUserIdentity `json:"sessionIssuer,omitempty"`
}
