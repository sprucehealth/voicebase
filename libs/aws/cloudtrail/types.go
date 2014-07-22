package cloudtrail

import (
	"bytes"
	"time"
)

type FlexibleBool bool

var (
	trueBytes  = []byte("true")
	falseBytes = []byte("false")
)

func (sb FlexibleBool) MarshalJSON() ([]byte, error) {
	if bool(sb) {
		return trueBytes, nil
	}
	return falseBytes, nil
}

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

type SNSNotification struct {
	S3Bucket    string   `json:"s3Bucket"`
	S3ObjectKey []string `json:"s3ObjectKey"`
}

type Log struct {
	Records []*Record
}

type Record struct {
	AWSRegion         string                 `json:"awsRegion"`
	ErrorCode         string                 `json:"errorCode"`
	ErrorMessage      string                 `json:"errorMessage"`
	EventName         string                 `json:"eventName"`
	EventSource       string                 `json:"eventSource"`
	EventTime         time.Time              `json:"eventTime"`
	EventVersion      string                 `json:"eventVersion"`
	RequestParameters map[string]interface{} `json:"requestParameters"`
	ResponseElements  map[string]interface{} `json:"responseElements"`
	SourceIPAddress   string                 `json:"sourceIPAddress"`
	UserAgent         string                 `json:"userAgent"`
	UserIdentity      *UserIdentity          `json:"userIdentity"`
}

type UserIdentity struct {
	AccessKeyID    string          `json:"accessKeyId"`
	AccountID      int64           `json:"accountId,string"`
	ARN            string          `json:"arn"`
	PrincipalID    string          `json:"principalId"`
	SessionContext *SessionContext `json:"sessionContext,omitempty"`
	Type           string          `json:"type"`
	UserName       string          `json:"userName"`
}

type SessionContext struct {
	Attributes struct {
		CreationDate     time.Time    `json:"creationDate"`
		MFAAuthenticated FlexibleBool `json:"mfaAuthenticated"`
	} `json:"attributes"`
	SessionIssuer *UserIdentity `json:"sessionIssuer,omitempty"`
}
