package api

import (
	"errors"
	"time"

	"github.com/sprucehealth/backend/app_url"
)

const (
	DQEventTypePatientVisit                  = "PATIENT_VISIT"
	DQEventTypeTreatmentPlan                 = "TREATMENT_PLAN"
	DQEventTypeRefillRequest                 = "REFILL_REQUEST"
	DQEventTypeTransmissionError             = "TRANSMISSION_ERROR"
	DQEventTypeUnlinkedDNTFTransmissionError = "UNLINKED_DNTF_TRANSMISSION_ERROR"
	DQEventTypeRefillTransmissionError       = "REFILL_TRANSMISSION_ERROR"
	DQEventTypeCaseMessage                   = "CASE_MESSAGE"
	DQEventTypeCaseAssignment                = "CASE_ASSIGNMENT"
	DQItemStatusPending                      = "PENDING"
	DQItemStatusTreated                      = "TREATED"
	DQItemStatusTriaged                      = "TRIAGED"
	DQItemStatusOngoing                      = "ONGOING"
	DQItemStatusRefillApproved               = "APPROVED"
	DQItemStatusRefillDenied                 = "DENIED"
	DQItemStatusReplied                      = "REPLIED"
	DQItemStatusRead                         = "READ"
	DisplayTypeTitleSubtitleActionable       = "title_subtitle_actionable"
)

type ByTimestamp []*DoctorQueueItem

func (a ByTimestamp) Len() int      { return len(a) }
func (a ByTimestamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTimestamp) Less(i, j int) bool {
	return a[i].EnqueueDate.Before(a[j].EnqueueDate)
}

type DoctorQueueItem struct {
	ID                   int64
	DoctorID             int64
	PatientID            int64
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
}

func (dqi *DoctorQueueItem) Validate() error {
	if dqi.DoctorID == 0 && dqi.PatientCaseID == 0 {
		return errors.New("atleast doctor_id or patient_case_id required")
	}
	if dqi.PatientID == 0 {
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
