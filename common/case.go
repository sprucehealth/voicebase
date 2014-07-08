package common

import (
	"time"

	"github.com/sprucehealth/backend/encoding"
)

type PatientCase struct {
	Id                encoding.ObjectId         `json:"case_id"`
	PatientId         encoding.ObjectId         `json:"patient_id"`
	HealthConditionId encoding.ObjectId         `json:"health_condition_id"`
	MedicineBranch    string                    `json:"medicine_branch"`
	CreationDate      time.Time                 `json:"creation_date"`
	Status            string                    `json:"status"`
	Diagnosis         string                    `json:"diagnosis,omitempty"`
	CareTeam          []*CareProviderAssignment `json:"care_team"`
}

const (
	CNTreatmentPlan = "treatment_plan"
	CNMessage       = "message"
)

type CaseNotification struct {
	Id               int64
	PatientCaseId    int64
	NotificationType string
	ItemId           int64
	CreationDate     time.Time
	Data             Typed
}

type CaseMessage struct {
	ID          int64
	CaseID      int64
	PersonID    int64
	Time        time.Time
	Body        string
	Attachments []*CaseMessageAttachment
}

type CaseMessageAttachment struct {
	ID       int64
	ItemType string
	ItemID   int64
}

type CaseMessageParticipant struct {
	CaseID   int64
	Unread   bool
	LastRead time.Time
	Person   *Person
}
