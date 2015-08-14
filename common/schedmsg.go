package common

import (
	"fmt"
	"strings"
	"time"
)

// ScheduledMessageStatus represents the possible values of the status field of the scheduled_message record
type ScheduledMessageStatus string

var (
	// SMScheduled indicates that a message has been scheduled to be sent
	SMScheduled ScheduledMessageStatus = "SCHEDULED"

	// SMProcessing indicates that a message has been picked up for sending
	SMProcessing ScheduledMessageStatus = "PROCESSING"

	// SMSent indicates that a message has been sucessfully sent
	SMSent ScheduledMessageStatus = "SENT"

	// SMError indicates that a message has encountered an error
	SMError ScheduledMessageStatus = "ERROR"

	// SMDeactivated indicates that a message has been deactivated and should not be sent
	SMDeactivated ScheduledMessageStatus = "DEACTIVATED"

	// SMEmailMessageType represents the scheduled message type for an email to be sent
	SMEmailMessageType = "email"

	// SMCaseMessageType represents the scheduled message type for an in app case message
	SMCaseMessageType = "case_message"

	// SMTreatmanPlanMessageType represents the scheduled message type for an in app treatment plan message
	SMTreatmanPlanMessageType = "treatment_plan_message"
)

// ParseScheduledMessageStatus returns the ScheduledMessageStatus that maps to the provided string
func ParseScheduledMessageStatus(s string) (ScheduledMessageStatus, error) {
	sm := ScheduledMessageStatus(strings.ToUpper(s))
	switch sm {
	case SMScheduled, SMProcessing, SMSent, SMError, SMDeactivated:
		return sm, nil
	}

	return ScheduledMessageStatus(""), fmt.Errorf("Unknown status: %s", s)
}

func (s *ScheduledMessageStatus) String() string {
	return string(*s)
}

// Scan implements the sql.Scanner interface for interaction with databases
func (s *ScheduledMessageStatus) Scan(src interface{}) error {
	var err error
	switch sm := src.(type) {
	case string:
		*s, err = ParseScheduledMessageStatus(sm)
	case []byte:
		*s, err = ParseScheduledMessageStatus(string(sm))
	}
	return err
}

// ScheduledMessage represents the data associated with a message that can be scheduled for future distribution
type ScheduledMessage struct {
	ID        int64
	Event     string
	PatientID PatientID
	Message   Typed
	Created   time.Time
	Scheduled time.Time
	Completed *time.Time
	Error     string
	Status    ScheduledMessageStatus
}

// ScheduledMessageTemplate represents the data associated with a starting point for a message that can be scheduled for future distribution
type ScheduledMessageTemplate struct {
	ID             int64     `json:"id,string"`
	Name           string    `json:"name"`
	Event          string    `json:"event"`
	SchedulePeriod int       `json:"scheduled_period"`
	Message        string    `json:"message"`
	Created        time.Time `json:"created"`
}
