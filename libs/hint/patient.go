package hint

import (
	"errors"
	"time"
)

// Phone represents a typed phone number
type Phone struct {
	Type   string `json:"type"`
	Number string `json:"number"`
}

// PatientParams represents the mutable fields of a patient.
type PatientParams struct {
	FirstName                string   `json:"first_name,omitempty"`
	LastName                 string   `json:"last_name,omitempty"`
	Email                    string   `json:"email,omitempty"`
	Gender                   string   `json:"gender,omitempty"`
	DOB                      string   `json:"dob,omitempty"`
	HealthInsuranceGroupID   string   `json:"health_insurance_group_id,omitempty"`
	HealthInsuranceMemberID  string   `json:"health_insurance_member_id,omitempty"`
	HealthInsurancePayerName string   `json:"health_insurance_payer_name,omitempty"`
	HealthInsurancePayerID   string   `json:"health_insurance_payer_id,omitempty"`
	AddressLine1             string   `json:"address_line_1,omitempty"`
	AddressLine2             string   `json:"address_line_2,omitempty"`
	AddressCity              string   `json:"address_city,omitempty"`
	AddressState             string   `json:"address_state,omitempty"`
	AddressZip               string   `json:"address_zip,omitempty"`
	AddressCountry           string   `json:"address_country,omitempty"`
	ExternalSourceID         string   `json:"external_source_id,omitempty"`
	ExternalSourceName       string   `json:"external_source_name,omitempty"`
	ExternalLinkID           string   `json:"external_link_id,omitempty"`
	Phones                   []*Phone `json:"phones,omitempty"`
}

// Validate ensures that the required fields in when creating
// or updating a patient are present.
func (p *PatientParams) Validate() error {
	if p.FirstName == "" {
		return errors.New("first_name required")
	} else if p.LastName == "" {
		return errors.New("last_name required")
	}
	return nil
}

// Patient represents a patient registered as part of a practice on hint.
type Patient struct {
	ID                       string        `json:"id"`
	CreatedAt                time.Time     `json:"created_at,omitempty"`
	UpdatedAt                time.Time     `json:"updated_at,omitempty"`
	FirstName                string        `json:"first_name,omitempty"`
	LastName                 string        `json:"last_name,omitempty"`
	Name                     string        `json:"name,omitempty"`
	Email                    string        `json:"email,omitempty"`
	DOB                      string        `json:"dob,omitempty"`
	Age                      int           `json:"age,omitempty"`
	Gender                   string        `json:"gender,omitempty"`
	MembershipStatus         string        `json:"membership_status,omitempty"`
	JoinedPracticeDate       string        `json:"date,omitempty"`
	ExternalSourceID         string        `json:"external_source_id,omitempty"`
	ExternalSourceName       string        `json:"external_source_name,omitempty"`
	ExternalLinkID           string        `json:"external_link_id,omitempty"`
	Practitioner             *Practitioner `json:"practitioner,omitempty"`
	LeadSource               string        `json:"lead_source,omitempty"`
	HealthInsuranceGroupID   string        `json:"health_insurance_group_id,omitempty"`
	HealthInsuranceMemberID  string        `json:"health_insurance_member_id,omitempty"`
	HealthInsurancePayerName string        `json:"health_insurance_payer_name,omitempty"`
	HealthInsurancePayerID   string        `json:"health_insurance_payer_id,omitempty"`
	Phones                   []*Phone      `json:"phones,omitempty"`
	AddressLine1             string        `json:"address_line_1,omitempty"`
	AddressLine2             string        `json:"address_line_2,omitempty"`
	AddressZip               string        `json:"address_zip,omitempty"`
	AddressCity              string        `json:"address_city,omitempty"`
	AddressState             string        `json:"address_state,omitempty"`
	AddressCountry           string        `json:"address_country,omitempty"`
}
