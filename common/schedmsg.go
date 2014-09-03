package common

import (
	"fmt"
	"time"
)

type ScheduledMessageStatus string

var (
	SMScheduled        ScheduledMessageStatus = "SCHEDULED"
	SMProcessing       ScheduledMessageStatus = "PROCESSING"
	SMSent             ScheduledMessageStatus = "SENT"
	SMError            ScheduledMessageStatus = "ERROR"
	SMEmailMessageType                        = "email"
	SMCaseMessageType                         = "case_message"
)

func GetScheduledMessageStatus(s string) (ScheduledMessageStatus, error) {
	sm := ScheduledMessageStatus(s)
	switch sm {
	case SMScheduled, SMProcessing, SMSent, SMError:
		return sm, nil
	}

	return ScheduledMessageStatus(""), fmt.Errorf("Unknown status: %s", s)
}

func (s *ScheduledMessageStatus) String() string {
	return string(*s)
}

func (s *ScheduledMessageStatus) Scan(src interface{}) error {
	var err error
	switch sm := src.(type) {
	case string:
		*s, err = GetScheduledMessageStatus(sm)
	case []byte:
		*s, err = GetScheduledMessageStatus(string(sm))
	}
	return err
}

type ScheduledMessage struct {
	ID          int64
	Type        string
	PatientID   int64
	MessageType string
	MessageJSON Typed
	Created     time.Time
	Scheduled   time.Time
	Completed   *time.Time
	Error       string
	Status      ScheduledMessageStatus
}

type ScheduledMessageTemplate struct {
	ID               int64
	Type             string
	SchedulePeriod   int
	MessageType      string
	MessageJSON      Typed
	CreatorAccountID int64
	Created          time.Time
}
