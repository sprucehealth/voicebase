package common

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/encoding"
)

type CaseStatus string

const (
	PCStatusUnclaimed           CaseStatus = "UNCLAIMED"
	PCStatusTempClaimed         CaseStatus = "TEMP_CLAIMED"
	PCStatusClaimed             CaseStatus = "CLAIMED"
	PCStatusUnsuitable          CaseStatus = "UNSUITABLE"
	PCStatusPreSubmissionTriage CaseStatus = "PRE_SUBMISSION_TRIAGE"
)

func (cs CaseStatus) String() string {
	return string(cs)
}

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

// ActivePatientCaseStates returns the possible states an active case can be in.
func ActivePatientCaseStates() []string {
	return []string{PCStatusUnclaimed.String(), PCStatusTempClaimed.String(), PCStatusClaimed.String()}
}

type PatientCase struct {
	ID             encoding.ObjectID `json:"case_id"`
	PatientID      encoding.ObjectID `json:"patient_id"`
	PathwayTag     string            `json:"pathway_id"`
	Name           string            `json:"name"`
	MedicineBranch string            `json:"medicine_branch"`
	CreationDate   time.Time         `json:"creation_date"`
	ClosedDate     *time.Time        `json:"closed_date,omitempty"`
	Status         CaseStatus        `json:"status"`
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
	ID          int64
	CaseID      int64
	PersonID    int64
	Time        time.Time
	Body        string
	EventText   string
	IsPrivate   bool
	Attachments []*CaseMessageAttachment
}

type CaseMessageAttachment struct {
	ID       int64
	ItemType string
	ItemID   int64
	MimeType string
	Title    string
}

type CaseMessageParticipant struct {
	CaseID   int64
	Unread   bool
	LastRead time.Time
	Person   *Person
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
