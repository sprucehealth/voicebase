package responses

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

type Case struct {
	ID            int64                    `json:"id,string"`
	PathwayTag    string                   `json:"pathway_id"`
	Title         string                   `json:"title"`
	Status        string                   `json:"status"`
	CreationDate  *time.Time               `json:"creation_date,omitempty"`
	CareTeam      []*PatientCareTeamMember `json:"care_team"`
	Diagnosis     string                   `json:"diagnosis,omitempty"`
	PatientVisits []*PatientVisit          `json:"patient_visits"`
	ActiveTPs     []*TreatmentPlan         `json:"active_treatment_plans,omitempty"`
	InactiveTPs   []*TreatmentPlan         `json:"inactive_treatment_plans,omitempty"`
	DraftTPs      []*TreatmentPlan         `json:"draft_treatment_plans,omitempty"`

	// Deprecated
	Name   string `json:"name"`
	CaseID int64  `json:"case_id,string,omitempty"`
}

func (c *Case) String() string {
	return fmt.Sprintf("{ID: %v}", c.ID)
}

// An entry representing an individual care team member
type PatientCareTeamMember struct {
	ProviderRole      string    `json:"provider_role"`
	ProviderID        int64     `json:"provider_id,string"`
	FirstName         string    `json:"first_name,omitempty"`
	LastName          string    `json:"last_name,omitempty"`
	ShortTitle        string    `json:"short_title,omitempty"`
	LongTitle         string    `json:"long_title,omitempty"`
	ShortDisplayName  string    `json:"short_display_name,omitempty"`
	LongDisplayName   string    `json:"long_display_name,omitempty"`
	SmallThumbnailURL string    `json:"small_thumbnail_url,omitempty"`
	LargeThumbnailURL string    `json:"large_thumbnail_url,omitempty"`
	ThumbnailURL      string    `json:"thumbnail_url,omitempty"`
	CreationDate      time.Time `json:"assignment_date"`
}

func (p *PatientCareTeamMember) String() string {
	return fmt.Sprintf("{ProviderID: %v, ProviderRole: %v, CreationDate: %v}", p.ProviderID, p.ProviderRole, p.CreationDate)
}

func TransformCareTeamMember(member *common.CareProviderAssignment, apiDomain string) *PatientCareTeamMember {
	return &PatientCareTeamMember{
		ProviderRole:      member.ProviderRole,
		ProviderID:        member.ProviderID,
		FirstName:         member.FirstName,
		LastName:          member.LastName,
		ShortTitle:        member.ShortTitle,
		LongTitle:         member.LongTitle,
		ShortDisplayName:  member.ShortDisplayName,
		LongDisplayName:   member.LongDisplayName,
		ThumbnailURL:      app_url.LargeThumbnailURL(apiDomain, member.ProviderRole, member.ProviderID),
		LargeThumbnailURL: app_url.LargeThumbnailURL(apiDomain, member.ProviderRole, member.ProviderID),
		SmallThumbnailURL: app_url.SmallThumbnailURL(apiDomain, member.ProviderRole, member.ProviderID),
		CreationDate:      member.CreationDate,
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
	ID            int64     `json:"id,string"`
	Title         string    `json:"title"`
	SubmittedDate time.Time `json:"submitted_date"`
	Status        string    `json:"status"`
}
