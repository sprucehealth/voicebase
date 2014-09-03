package common

import (
	"fmt"
	"time"
)

type scheduledMessageStatus string

var (
	SMScheduled        scheduledMessageStatus = "SCHEDULED"
	SMSent             scheduledMessageStatus = "SENT"
	SMError            scheduledMessageStatus = "ERROR"
	SMEmailMessageType                        = "email"
	SMCaseMessageType                         = "case_message"
)

func GetScheduledMessageStatus(s string) (scheduledMessageStatus, error) {
	sm := scheduledMessageStatus(s)
	switch sm {
	case SMScheduled, SMSent, SMError:
		return sm, nil
	}

	return scheduledMessageStatus(""), fmt.Errorf("Unknown status: %s", s)
}

func (s *scheduledMessageStatus) String() string {
	return string(*s)
}

func (s *scheduledMessageStatus) Scan(src interface{}) error {
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
	Status      scheduledMessageStatus
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
