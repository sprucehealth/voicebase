package responses

import (
	"fmt"
	"time"
)

type Case struct {
	ID            int64            `json:"id,string"`
	PathwayTag    string           `json:"pathway_id"`
	Title         string           `json:"title"`
	Status        string           `json:"status"`
	PatientVisits []*PatientVisit  `json:"patient_visits"`
	ActiveTPs     []*TreatmentPlan `json:"active_treatment_plans,omitempty"`
	InactiveTPs   []*TreatmentPlan `json:"inactive_treatment_plans,omitempty"`
	DraftTPs      []*TreatmentPlan `json:"draft_treatment_plans,omitempty"`
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
	CreationDate      time.Time `json:"assignment_date"`
}

func (p *PatientCareTeamMember) String() string {
	return fmt.Sprintf("{ProviderID: %v, ProviderRole: %v, CreationDate: %v}", p.ProviderID, p.ProviderRole, p.CreationDate)
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
