package responses

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

type Case struct {
	ID                     int64                    `json:"id,string"`
	PathwayTag             string                   `json:"pathway_id"`
	PathwayName            string                   `json:"pathway_name"`
	Title                  string                   `json:"title"`
	Status                 string                   `json:"status"`
	DisplayStatus          string                   `json:"display_status"`
	DeprecatedCreationDate *time.Time               `json:"creation_date,omitempty"`
	CreationEpoch          int64                    `json:"creation_epoch,omitempty"`
	CareTeam               []*PatientCareTeamMember `json:"care_team"`
	Diagnosis              string                   `json:"diagnosis,omitempty"`
	PatientVisits          []*PatientVisit          `json:"patient_visits"`
	ActiveTPs              []*TreatmentPlan         `json:"active_treatment_plans,omitempty"`
	InactiveTPs            []*TreatmentPlan         `json:"inactive_treatment_plans,omitempty"`
	DraftTPs               []*TreatmentPlan         `json:"draft_treatment_plans,omitempty"`

	// Deprecated
	Name   string `json:"name"`
	CaseID int64  `json:"case_id,string,omitempty"`
}

func (c *Case) String() string {
	return fmt.Sprintf("{ID: %v}", c.ID)
}

func NewCase(pc *common.PatientCase, careTeamMembers []*PatientCareTeamMember, diagnosis string) *Case {

	firstLetter := false
	displayStatus := strings.Map(func(r rune) rune {
		if !firstLetter {
			firstLetter = true
			return unicode.ToTitle(r)
		}
		return unicode.ToLower(r)
	}, pc.Status.String())

	return &Case{
		ID:          pc.ID.Int64(),
		CaseID:      pc.ID.Int64(),
		PathwayTag:  pc.PathwayTag,
		PathwayName: pc.Name,
		Title:       fmt.Sprintf("%s Case", pc.Name),
		DeprecatedCreationDate: &pc.CreationDate,
		CreationEpoch:          pc.CreationDate.Unix(),
		Status:                 pc.Status.String(),
		DisplayStatus:          displayStatus,
		Diagnosis:              diagnosis,
		CareTeam:               careTeamMembers,
	}
}

// An entry representing an individual care team member
type PatientCareTeamMember struct {
	*CareProvider
	ProviderRole           string    `json:"provider_role"`
	DeprecatedCreationDate time.Time `json:"assignment_date"`
	CreationEpoch          int64     `json:"assignment_epoch"`
}

func (p *PatientCareTeamMember) String() string {
	return fmt.Sprintf("{ProviderID: %v, ProviderRole: %v, DeprecatedCreationDate: %v, CreationEpoch: %v}", p.CareProvider.ProviderID, p.ProviderRole, p.DeprecatedCreationDate, p.CreationEpoch)
}

func TransformCareTeamMember(member *common.CareProviderAssignment, apiDomain string) *PatientCareTeamMember {
	return &PatientCareTeamMember{
		CareProvider: &CareProvider{
			ProviderID:                  member.ProviderID,
			FirstName:                   member.FirstName,
			LastName:                    member.LastName,
			ShortTitle:                  member.ShortTitle,
			LongTitle:                   member.LongTitle,
			ShortDisplayName:            member.ShortDisplayName,
			LongDisplayName:             member.LongDisplayName,
			ThumbnailURL:                app_url.ThumbnailURL(apiDomain, member.ProviderRole, member.ProviderID),
			DeprecatedLargeThumbnailURL: app_url.ThumbnailURL(apiDomain, member.ProviderRole, member.ProviderID),
			DeprecatedSmallThumbnailURL: app_url.ThumbnailURL(apiDomain, member.ProviderRole, member.ProviderID),
		},
		ProviderRole:           member.ProviderRole,
		DeprecatedCreationDate: member.CreationDate,
		CreationEpoch:          member.CreationDate.Unix(),
	}
}

// A summary object representing an individual care team
type PatientCareTeamSummary struct {
	CaseID  int64                    `json:"case_id,string"`
	Members []*PatientCareTeamMember `json:"members"`
}

func (p *PatientCareTeamSummary) String() string {
	return fmt.Sprintf("{CaseID: %v, Members: %v}", p.CaseID, p.Members)
}

type PatientVisit struct {
	ID                      int64     `json:"id,string"`
	Title                   string    `json:"title"`
	DeprecatedSubmittedDate time.Time `json:"submitted_date"`
	SubmittedEpoch          int64     `json:"submitted_epoch"`
	Status                  string    `json:"status"`
}

func NewPatientVisit(visit *common.PatientVisit, title string) *PatientVisit {
	return &PatientVisit{
		ID:                      visit.ID.Int64(),
		Title:                   title,
		Status:                  visit.Status,
		DeprecatedSubmittedDate: visit.SubmittedDate,
		SubmittedEpoch:          visit.SubmittedDate.Unix(),
	}
}

type PHISafeVisitSummary struct {
	VisitID         int64   `json:"visit_id,string"`         // patient_visit.id
	CaseID          int64   `json:"case_id,string"`          // patient_visit.patient_case_id
	CreationEpoch   int64   `json:"creation_epoch,string"`   // patient_visit.creation_date
	SubmittedEpoch  int64   `json:"submitted_epoch,string"`  // patient_visit.submitted_date
	LockTakenEpoch  int64   `json:"lock_taken_epoch,string"` // patient_case_care_provider_assignment.creation_date
	DoctorID        *int64  `json:"doctor_id,string"`        // doctor.id
	FirstAvailable  bool    `json:"first_available"`         // patient_case.requested_doctor_id
	Pathway         string  `json:"pathway"`                 // clinical_pathway.name
	DoctorWithLock  string  `json:"doctor_with_lock"`        // patient_case.requested_doctor_id
	PatientInitials string  `json:"patient_initials"`        // patient.first_name, patient.last_name
	CaseName        string  `json:"case_name"`               // patient_case.name
	Type            string  `json:"type"`                    // sku.type
	SubmissionState *string `json:"submission_state"`        // patient_location.state
	Status          string  `json:"status"`                  // patient_visit.status
	LockType        *string `json:"lock_type"`
}

func TransformVisitSummary(summary *common.VisitSummary) *PHISafeVisitSummary {
	response := &PHISafeVisitSummary{
		VisitID:         summary.VisitID,
		CaseID:          summary.CaseID,
		CreationEpoch:   summary.CreationDate.Unix(),
		Pathway:         summary.PathwayName,
		CaseName:        summary.CaseName,
		Type:            summary.SKUType,
		SubmissionState: summary.SubmissionState,
		Status:          summary.Status,
		LockType:        summary.LockType,
		DoctorID:        summary.DoctorID,
	}
	if summary.RequestedDoctorID == nil {
		response.FirstAvailable = true
	}
	if summary.DoctorFirstName != nil {
		response.DoctorWithLock += *summary.DoctorFirstName
	}
	if summary.DoctorLastName != nil {
		response.DoctorWithLock += " " + *summary.DoctorLastName
	}

	var firstInitial string
	var lastInitial string
	if len(summary.PatientFirstName) > 0 {
		firstInitial = string(summary.PatientFirstName[0])
	}
	if len(summary.PatientLastName) > 0 {
		lastInitial = string(summary.PatientLastName[0])
	}
	response.PatientInitials = fmt.Sprintf("%s%s", firstInitial, lastInitial)
	if summary.LockTakenDate != nil {
		response.LockTakenEpoch = summary.LockTakenDate.Unix()
	}
	if summary.SubmittedDate != nil {
		response.SubmittedEpoch = summary.SubmittedDate.Unix()
	}
	return response
}
