package common

import (
	"time"

	"github.com/sprucehealth/backend/encoding"
)

const (
	PCStatusUnclaimed   = "UNCLAIMED"
	PCStatusTempClaimed = "TEMP_CLAIMED"
	PCStatusClaimed     = "CLAIMED"
	PCStatusUnsuitable  = "UNSUITABLE"
)

type PatientCase struct {
	ID             encoding.ObjectID         `json:"case_id"`
	PatientID      encoding.ObjectID         `json:"patient_id"`
	PathwayID      encoding.ObjectID         `json:"pathway_id"`
	Name           string                    `json:"name"`
	MedicineBranch string                    `json:"medicine_branch"`
	CreationDate   time.Time                 `json:"creation_date"`
	Status         string                    `json:"status"`
	Diagnosis      string                    `json:"diagnosis,omitempty"`
	CareTeam       []*CareProviderAssignment `json:"care_team"`
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
