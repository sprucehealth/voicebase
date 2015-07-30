package responses

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/pharmacy"
)

// Patient is a response layer representation of the patient
type Patient struct {
	ID                 int64                  `json:"id,string"`
	IsUnlinked         bool                   `json:"is_unlinked"`
	FirstName          string                 `json:"first_name"`
	LastName           string                 `json:"last_name"`
	MiddleName         string                 `json:"middle_name"`
	Suffix             string                 `json:"suffix"`
	Prefix             string                 `json:"prefix"`
	DOB                string                 `json:"dob"`
	Email              string                 `json:"email"`
	Gender             string                 `json:"gender"`
	ZipCode            string                 `json:"zip_code"`
	State              string                 `json:"state_code"`
	PhoneNumbers       []*common.PhoneNumber  `json:"phone_numbers"`
	PrimaryPhoneNumber string                 `json:"primary_phone_number,omitempty"`
	ERxID              int64                  `json:"erx_patient_id,string"`
	Pharmacy           *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
	Address            *common.Address        `json:"address,omitempty"`
	PersonID           int64                  `json:"person_id"`
	PromptStatus       string                 `json:"prompt_status"`
	Training           bool                   `json:"is_training"`
	HasParentalConsent bool                   `json:"has_parental_consent"`
	ParentInfoExists   bool                   `json:"parent_info_exists"`
}

// TransformPatient converts a data layer representation of the patient
// to a response layer representation
func TransformPatient(pat *common.Patient) *Patient {
	p := &Patient{
		ID:                 pat.ID.Int64(),
		IsUnlinked:         pat.IsUnlinked,
		FirstName:          pat.FirstName,
		LastName:           pat.LastName,
		MiddleName:         pat.MiddleName,
		Suffix:             pat.Suffix,
		Prefix:             pat.Prefix,
		DOB:                pat.DOB.String(),
		Email:              pat.Email,
		Gender:             pat.Gender,
		ZipCode:            pat.ZipCode,
		State:              pat.StateFromZipCode,
		PhoneNumbers:       pat.PhoneNumbers,
		ERxID:              pat.ERxPatientID.Int64(),
		Pharmacy:           pat.Pharmacy,
		Address:            pat.PatientAddress,
		PersonID:           pat.PersonID,
		PromptStatus:       pat.PromptStatus.String(),
		Training:           pat.Training,
		HasParentalConsent: pat.HasParentalConsent,
		ParentInfoExists:   pat.HasParentalConsent,
	}

	if len(pat.PhoneNumbers) > 0 {
		p.PrimaryPhoneNumber = pat.PhoneNumbers[0].Phone.String()
	}
	return p
}
