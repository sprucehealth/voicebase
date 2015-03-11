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
		ID:                      visit.PatientVisitID.Int64(),
		Title:                   title,
		Status:                  visit.Status,
		DeprecatedSubmittedDate: visit.SubmittedDate,
		SubmittedEpoch:          visit.SubmittedDate.Unix(),
	}
}
