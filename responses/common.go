/*
responses is a package intended to represent common internal response subobjects
*/

package responses

import (
	"fmt"
	"time"
)

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
	Case    *Case                    `json:"case,omitempty"`
	Members []*PatientCareTeamMember `json:"members"`
}

func (p *PatientCareTeamSummary) String() string {
	return fmt.Sprintf("{Case: %v, Members: %v}", p.Case, p.Members)
}

// An object representing a chief complaint with localization fields
type ChiefComplaint struct {
	ID            int64  `json:"id,string"`
	Name          string `json:"name,omitempty"`
	NameLocalized string `json:"name_localized,omitempty"`
}

func (c *ChiefComplaint) String() string {
	return fmt.Sprintf("{ID: %v, Name: %v, NameLocalized: %v}", c.ID, c.Name, c.NameLocalized)
}

// An object representing a case with a chief complaint
type Case struct {
	ID             int64           `json:"id,string"`
	ChiefComplaint *ChiefComplaint `json:"chief_complaint,omitempty"`
}

func (c *Case) String() string {
	return fmt.Sprintf("{ID: %v, ChiefComplaint: %v}", c.ID, c.ChiefComplaint)
}
