package api

import (
	"errors"
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

// Doctor queue event types
const (
	DQEventTypeCaseAssignment                = "CASE_ASSIGNMENT"
	DQEventTypeCaseMessage                   = "CASE_MESSAGE"
	DQEventTypePatientVisit                  = "PATIENT_VISIT"
	DQEventTypeRefillRequest                 = "REFILL_REQUEST"
	DQEventTypeRefillTransmissionError       = "REFILL_TRANSMISSION_ERROR"
	DQEventTypeTransmissionError             = "TRANSMISSION_ERROR"
	DQEventTypeTreatmentPlan                 = "TREATMENT_PLAN"
	DQEventTypeUnlinkedDNTFTransmissionError = "UNLINKED_DNTF_TRANSMISSION_ERROR"
)

// Doctor queue item statuses
const (
	DQItemStatusOngoing        = "ONGOING"
	DQItemStatusPending        = "PENDING"
	DQItemStatusRead           = "READ"
	DQItemStatusRefillApproved = "APPROVED"
	DQItemStatusRefillDenied   = "DENIED"
	// DQItemStatusRemoved status represents an item that has been removed from the active queue.
	DQItemStatusRemoved = "REMOVED"
	DQItemStatusReplied = "REPLIED"
	DQItemStatusTreated = "TREATED"
	DQItemStatusTriaged = "TRIAGED"
)

const DisplayTypeTitleSubtitleActionable = "title_subtitle_actionable"

const tagSeparator = "|"

type byTimestamp []*DoctorQueueItem

func (a byTimestamp) Len() int      { return len(a) }
func (a byTimestamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byTimestamp) Less(i, j int) bool {
	return a[i].EnqueueDate.Before(a[j].EnqueueDate)
}

type DoctorQueueType string

const (
	DQTUnclaimedQueue DoctorQueueType = "unclaimed"
	DQTDoctorQueue    DoctorQueueType = "doctor"
)

func (dqt DoctorQueueType) String() string {
	return string(dqt)
}

func ParseDoctorQueueType(s string) DoctorQueueType {
	return DoctorQueueType(s)
}

type DoctorQueueItem struct {
	ID                   int64
	DoctorID             int64
	PatientID            common.PatientID
	EventType            string
	EnqueueDate          time.Time
	Expires              *time.Time
	ItemID               int64
	Status               string
	PatientCaseID        int64
	CareProvidingStateID int64
	Description          string
	ShortDescription     string
	ActionURL            *app_url.SpruceAction
	Tags                 []string
	QueueType            DoctorQueueType
}

func (dqi *DoctorQueueItem) Validate() error {
	if dqi.DoctorID == 0 && dqi.PatientCaseID == 0 {
		return errors.New("atleast doctor_id or patient_case_id required")
	}
	if !dqi.PatientID.IsValid {
		return errors.New("missing patient id")
	}
	if dqi.Description == "" {
		return errors.New("missing description")
	}
	if dqi.ShortDescription == "" {
		return errors.New("missing short description")
	}
	if dqi.EventType == "" {
		return errors.New("missing event_type")
	}
	if dqi.Status == "" {
		return errors.New("missing status")
	}
	return nil
}
