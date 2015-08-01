package common

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/encoding"
)

// CaseStatus is the current state of a case
type CaseStatus string

// Constants for possible case statuses. Flow between states:
//
//                      ┌───▶UNSUITABLE
//                      │
// OPEN───────▶ACTIVE───┴───▶INACTIVE
//   │
//   ├────────▶DELETED
//   │
//   ├────────▶PRE_SUBMISSION_TRIAGE
//   │
//   └────────▶PRE_SUBMISSION_TRIAGE_DELETED
const (
	// PCStatusOpen is the state used to indicate a case that has not been submitted to the doctor yet
	// and is considered unfinished by the patient.
	// A case transitions from the OPEN -> ACTIVE state upon the submission of the first visit in the case.
	PCStatusOpen CaseStatus = "OPEN"

	// PCStatusActive is the state used to indicate a case that has been submitted to the doctor, and
	// is within the acceptable window of treatment.
	PCStatusActive CaseStatus = "ACTIVE"

	// PCStatusInactive is the state used to indicate a submitted case that is outside the window of treatment
	// and transitioned to being inactive either automatically (based on a predefined window) or
	// as a result of a doctor/cc marking it so.
	PCStatusInactive CaseStatus = "INACTIVE"

	// PCStatusDeleted is the state used to indicate an unsubmitted case that has been marked as deleted
	// by the patient before submitting to the doctor.
	PCStatusDeleted CaseStatus = "DELETED"

	// PCStatusUnsuitable is the state used to indicate a submitted case that was marked as being unsuitable
	// for Spruce by a doctor upon reviewing the patient visits.
	PCStatusUnsuitable CaseStatus = "UNSUITABLE"

	// PCStatusPreSubmissionTriage is the state used to indicate a case that has been automatically triaged
	// pre-submission based on the information the patient entered.
	PCStatusPreSubmissionTriage CaseStatus = "PRE_SUBMISSION_TRIAGE"

	// PCStatusPreSubmissionTriageDeleted is the state used to indicate a case that was triaged pre-submission
	// and then transitioned to the deleted state upon reaching the timeout.
	PCStatusPreSubmissionTriageDeleted CaseStatus = "PRE_SUBMISSION_TRIAGE_DELETED"
)

// String implements fmt.Stringer
func (cs CaseStatus) String() string {
	return string(cs)
}

// Scan implements sql.Scanner. It expects src to be on-nil and of type string or []byte
func (cs *CaseStatus) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		*cs = CaseStatus(v)
	case []byte:
		*cs = CaseStatus(v)
	default:
		return fmt.Errorf("unsupported scan type for CaseStatus: %T", src)
	}
	return nil
}

type PatientCase struct {
	ID                encoding.ObjectID `json:"case_id"`
	PatientID         encoding.ObjectID `json:"patient_id"`
	PathwayTag        string            `json:"pathway_id"`
	Name              string            `json:"name"`
	CreationDate      time.Time         `json:"creation_date"`
	ClosedDate        *time.Time        `json:"closed_date,omitempty"`
	Status            CaseStatus        `json:"status"`
	TimeoutDate       *time.Time        `json:"-"`
	RequestedDoctorID *int64            `json:"requested_doctor_id"`

	// Claimed is set to true when the case has a doctor permanently assigned to it.
	Claimed bool `json:"claimed"`
}

// DeletedPatientCaseStates returns all the states considered deleted for a case
func DeletedPatientCaseStates() []string {
	return []string{PCStatusDeleted.String(), PCStatusPreSubmissionTriageDeleted.String()}
}

// OpenPatientCaseStates returns all the states considered open for a case
func OpenPatientCaseStates() []string {
	return []string{PCStatusOpen.String()}
}

// SubmittedPatientCaseStates returns all the states considered submitted for a case
func SubmittedPatientCaseStates() []string {
	return []string{PCStatusActive.String(), PCStatusInactive.String()}
}

type CaseNotification struct {
	ID               int64
	PatientCaseID    int64
	NotificationType string
	UID              string
	CreationDate     time.Time
	Data             Typed
}

type CaseMessage struct {
	ID           int64
	CaseID       int64
	PersonID     int64
	Time         time.Time
	Body         string
	EventText    string
	IsPrivate    bool
	Attachments  []*CaseMessageAttachment
	ReadReceipts []*ReadReceipt
}

type CaseMessageAttachment struct {
	ID       int64
	ItemType string
	ItemID   int64
	MimeType string
	Title    string
}

type CaseMessageParticipant struct {
	CaseID int64
	Person *Person
}

// ReadReceipt is the time when a person read a case message
type ReadReceipt struct {
	PersonID int64
	Time     time.Time
}

const (
	TCSStatusCreating = "CREATING"
	TCSStatusPending  = "PENDING"
)

type TrainingCase struct {
	TrainingCaseSetID int64
	PatientVisitID    int64
	TemplateName      string
}

// ByPatientCaseCreationDate implements sort.Interface to sort a slice of cases by creation date
type ByPatientCaseCreationDate []*PatientCase

func (c ByPatientCaseCreationDate) Len() int      { return len(c) }
func (c ByPatientCaseCreationDate) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c ByPatientCaseCreationDate) Less(i, j int) bool {
	return c[i].CreationDate.Before(c[j].CreationDate)
}

type PatientFeedback struct {
	PatientID int64
	Rating    int
	Comment   string
	Created   time.Time
}
