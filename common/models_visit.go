package common

import (
	"time"

	"github.com/sprucehealth/backend/encoding"
)

// Constants for possible visit statuses. Flow between states:
//
// PENDING  ┌───▶PENDING_PARENTAL_CONSENT
//    │     │                │                 ┌────────▶TRIAGED
//    │     │        ┌───────┘                 │
//    ▼     │        ▼                         │
//  OPEN────┴───▶SUBMITTED──┬──▶ROUTED────▶REVIEWING────▶TREATED
//    │                     │      ▲
//    │                     │      │
//    ├────▶DELETED         └──▶CHARGED
//    │
//    └────▶PRE_SUBMISSION_TRIAGE
const (
	// PVStatusOpen is the state used to indicate an unsubmitted visit.
	PVStatusOpen = "OPEN"

	// PVStatusPreSubmissionTriage is the state used to indicate a visit that was triaged pre-submission
	// based on the information the patient entered.
	PVStatusPreSubmissionTriage = "PRE_SUBMISSION_TRIAGE"

	// PVStatusPending is the state used to indicate a followup visit that is created but not yet opened by the patient.
	PVStatusPending = "PENDING"

	// PVStatusPendingParentalConsent is the state used to indicate a visit is waiting for a parent or
	// guardian to consent to a patient to be treated by Spruce. The visit will transition to SUBMITTED once
	// approval has been granted.
	PVStatusPendingParentalConsent = "PENDING_PARENTAL_CONSENT"

	// PVStatusSubmitted is the state used to indicate a visit that is submitted by the patient, but has not
	// yet been routed to the unassigned queue or the doctors queue. The visit will transition from
	// SUBMITTED -> CHARGED once we charge the patient for the visit in the background, after which the
	// visit will transition from CHARGED -> ROUTED.
	// Note that its also possible for the visit to transition from SUBMITTED->ROUTED if there is no price
	// determined for a visit.
	PVStatusSubmitted = "SUBMITTED"

	// PVStatusCharged is the state used to indicate a submitted visit for which the patient was successfully charged.
	// The visit state will transition from CHARGED -> ROUTED upon succesfully being routed to the unassigned or the doctor's queue.
	PVStatusCharged = "CHARGED"

	// PVStatusRouted is the state used to indicate a submitted and charged visit that is successfully routed either to the
	// unassigned queue or the inbox of a doctor's queue.
	// The visit transitions from the ROUTED -> REVIEWING state once the doctor opens the visit to review it.
	PVStatusRouted = "ROUTED"

	// PVStatusReviewing is the state used to indicate a routed visit that is currently being reivewed by the doctor.
	PVStatusReviewing = "REVIEWING"

	// PVStatusTriaged is the state used to indicate a visit that was marked as being unsuitable for Spruce by the doctor.
	PVStatusTriaged = "TRIAGED"

	// PVStatusTreated is the state used to indicate a visit that is treated by a doctor with a successful generation of a treatment plan.
	PVStatusTreated = "TREATED"

	// PVStatusDeleted is the state used to indicate a visit as having been deleted/abandoned by the user.
	PVStatusDeleted = "DELETED"
)

// SubmittedPatientVisitStates returns the set of visit statuses for a submitted visit
func SubmittedPatientVisitStates() []string {
	return []string{PVStatusSubmitted, PVStatusCharged, PVStatusRouted, PVStatusReviewing}
}

// TreatedPatientVisitStates returns the set of visit statuses for a treated visit (including triage)
func TreatedPatientVisitStates() []string {
	return []string{PVStatusTreated, PVStatusTriaged}
}

// OpenPatientVisitStates returns the set of visit statuses for open or pending visits
func OpenPatientVisitStates() []string {
	return []string{PVStatusPending, PVStatusOpen, PVStatusPendingParentalConsent}
}

// NonOpenPatientVisitStates returns the union of the set "treated statuses | submitted statuses"
func NonOpenPatientVisitStates() []string {
	return append(TreatedPatientVisitStates(), SubmittedPatientVisitStates()...)
}

// PatientVisitSubmitted indicates whether the status represents a submitted or unsubmitted
// visit. If the status is not understood, it treats the visit to be in the submitted
// state as its the safest thing to do given the visit is treated as immutable in the submitted state.
func PatientVisitSubmitted(status string) bool {
	switch status {
	case PVStatusSubmitted, PVStatusCharged, PVStatusRouted, PVStatusReviewing, PVStatusTriaged, PVStatusTreated:
		return true
	case PVStatusPending, PVStatusOpen, PVStatusPreSubmissionTriage, PVStatusDeleted:
		return false
	}

	return true

}

// ByPatientVisitCreationDate implements sort.Interface to sort a slice of visits by creation date
type ByPatientVisitCreationDate []*PatientVisit

func (c ByPatientVisitCreationDate) Len() int      { return len(c) }
func (c ByPatientVisitCreationDate) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c ByPatientVisitCreationDate) Less(i, j int) bool {
	return c[i].CreationDate.Before(c[j].CreationDate)
}

// PatientVisit is the model for a visit started by a patient
type PatientVisit struct {
	ID              encoding.ObjectID `json:"patient_visit_id,omitempty"`
	PatientCaseID   encoding.ObjectID `json:"case_id"`
	PatientID       encoding.ObjectID `json:"patient_id,omitempty"`
	CreationDate    time.Time         `json:"creation_date,omitempty"`
	SubmittedDate   time.Time         `json:"submitted_date,omitempty"`
	ClosedDate      time.Time         `json:"closed_date,omitempty"`
	PathwayTag      string            `json:"pathway_id,omitempty"`
	Status          string            `json:"status,omitempty"`
	IsFollowup      bool              `json:"is_followup"`
	LayoutVersionID encoding.ObjectID `json:"layout_version_id,omitempty"`
	SKUType         string            `json:"-"`
}

// ByVisitSummaryCreationDate implements sort.Interface to sort a slice of visit summaries by creation date
type ByVisitSummaryCreationDate []*VisitSummary

func (c ByVisitSummaryCreationDate) Len() int      { return len(c) }
func (c ByVisitSummaryCreationDate) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c ByVisitSummaryCreationDate) Less(i, j int) bool {
	return c[i].CreationDate.Before(c[j].CreationDate)
}

// ByVisitSummarySubmissionDate implements sort.Interface to sort a slice of visit summaries by submission date
type ByVisitSummarySubmissionDate []*VisitSummary

func (c ByVisitSummarySubmissionDate) Len() int      { return len(c) }
func (c ByVisitSummarySubmissionDate) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c ByVisitSummarySubmissionDate) Less(i, j int) bool {
	if c[i].SubmittedDate == nil {
		return false
	}
	if c[j].SubmittedDate == nil {
		return true
	}
	return c[i].SubmittedDate.Before(*c[j].SubmittedDate)
}

type VisitSummary struct {
	VisitID           int64
	CaseID            int64
	CreationDate      time.Time
	SubmittedDate     *time.Time
	LockTakenDate     *time.Time
	RequestedDoctorID *int64
	DoctorID          *int64
	RoleTypeTag       *string
	PathwayName       string
	PatientAccountID  int64
	PatientFirstName  string
	PatientLastName   string
	CaseName          string
	SKUType           string
	SubmissionState   *string
	Status            string
	DoctorFirstName   *string
	DoctorLastName    *string
	LockType          *string
}
