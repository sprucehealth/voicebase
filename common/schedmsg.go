package common

import (
	"fmt"
	"time"
)

type ScheduledMessageEvent string

const (
	// supported events on which app messages can be scheduled
	SMUninsuredPatientEvent    ScheduledMessageEvent = "uninsured_patient"
	SMInsuredPatientEvent      ScheduledMessageEvent = "insured_patient"
	SMTreatmentPlanViewedEvent ScheduledMessageEvent = "treatment_plan_viewed"
)

func GetScheduledMessageEvent(s string) (ScheduledMessageEvent, error) {
	switch sm := ScheduledMessageEvent(s); sm {
	case SMInsuredPatientEvent, SMUninsuredPatientEvent, SMTreatmentPlanViewedEvent:
		return sm, nil
	}

	return ScheduledMessageEvent(""), fmt.Errorf("Unknown event: %s", s)
}

func (s *ScheduledMessageEvent) Scan(src interface{}) error {
	var err error
	switch sm := src.(type) {
	case string:
		*s, err = GetScheduledMessageEvent(sm)
	case []byte:
		*s, err = GetScheduledMessageEvent(string(sm))
	}

	return err
}

func (s ScheduledMessageEvent) MarshalJSON() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *ScheduledMessageEvent) UnmarshalJSON(data []byte) error {
	strData := string(data)

	if len(strData) == 0 {
		return nil
	}

	var err error
	if strData[0] == '"' && len(strData) > 2 {
		*s, err = GetScheduledMessageEvent(strData[1 : len(strData)-1])
	} else {
		*s, err = GetScheduledMessageEvent(strData)
	}

	return err
}

func (s ScheduledMessageEvent) String() string {
	return string(s)
}

var (
	ScheduledMessageEvents = []string{
		SMInsuredPatientEvent.String(),
		SMUninsuredPatientEvent.String(),
		SMTreatmentPlanViewedEvent.String(),
	}
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
	Event       ScheduledMessageEvent
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
	ID               int64                 `json:"id,string"`
	Name             string                `json:"name"`
	Event            ScheduledMessageEvent `json:"event"`
	SchedulePeriod   int                   `json:"scheduled_period"`
	Message          string                `json:"message"`
	CreatorAccountID int64                 `json:"-"`
	Created          time.Time             `json:"created"`
}
