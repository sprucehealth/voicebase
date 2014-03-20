package common

import (
	"carefront/libs/pharmacy"
	"reflect"
	"time"
)

type PrescriptionStatus struct {
	TreatmentId        int64     `json:"-"`
	PrescriptionId     int64     `json:"erx_id,string,omitempty"`
	PrescriptionStatus string    `json:"erx_status,omitempty"`
	StatusTimestamp    time.Time `json:"erx_status_timestamp,omitempty"`
	ReportedTimestamp  time.Time `json:"reported_timestamp,omitempty"`
	StatusDetails      string    `json:"erx_status_details,omitempty"`
}

type Treatment struct {
	Id                        *ObjectId                `json:"treatment_id,omitempty"`
	DoctorTreatmentTemplateId *ObjectId                `json:"dr_treatment_template_id,omitempty"`
	PrescriptionId            *ObjectId                `json:"erx_id,omitempty"`
	ErxMedicationId           *ObjectId                `json:"-"`
	PrescriptionStatus        string                   `json:"erx_status,omitempty"`
	PharmacyLocalId           *ObjectId                `json:"-"`
	ErxPharmacyId             int64                    `json:"-"`
	StatusDetails             string                   `json:"erx_status_details,omitempty"`
	TreatmentPlanId           *ObjectId                `json:"treatment_plan_id,omitempty"`
	PatientVisitId            *ObjectId                `json:"patient_visit_id,omitempty"`
	PatientId                 *ObjectId                `json:"-"`
	DrugDBIds                 map[string]string        `json:"drug_db_ids,omitempty"`
	DrugInternalName          string                   `json:"drug_internal_name,omitempty"`
	DrugName                  string                   `json:"drug_name"`
	DrugRoute                 string                   `json:"drug_route,omitempty"`
	DrugForm                  string                   `json:"drug_form,omitempty"`
	DosageStrength            string                   `json:"dosage_strength,omitempty"`
	DispenseValue             int64                    `json:"dispense_value,string,omitempty"`
	DispenseUnitId            *ObjectId                `json:"dispense_unit_id,omitempty"`
	DispenseUnitDescription   string                   `json:"dispense_unit_description,omitempty"`
	NumberRefills             int64                    `json:"refills,string,omitempty"`
	SubstitutionsAllowed      bool                     `json:"substitutions_allowed,omitempty"`
	DaysSupply                int64                    `json:"days_supply,string,omitempty"`
	PharmacyNotes             string                   `json:"pharmacy_notes,omitempty"`
	PatientInstructions       string                   `json:"patient_instructions,omitempty"`
	CreationDate              *time.Time               `json:"creation_date,omitempty"`
	TransmissionErrorDate     *time.Time               `json:"error_date,omitempty"`
	ErxSentDate               *time.Time               `json:"erx_sent_date,omitempty"`
	ErxLastDateFilled         *time.Time               `json:"erx_last_filled_date,omitempty"`
	ErxReferenceNumber        string                   `json:"-"`
	Status                    string                   `json:"-"`
	OTC                       bool                     `json:"otc,omitempty"`
	IsControlledSubstance     bool                     `json:"-"`
	SupplementalInstructions  []*DoctorInstructionItem `json:"supplemental_instructions,omitempty"`
	Pharmacy                  *pharmacy.PharmacyData   `json:"pharmacy,omitempty"`
	PrescriberId              int64                    `json:"-"`
	Doctor                    *Doctor                  `json:"doctor,omitempty"`
	DoseSpotClinicianId       int64                    `json:"-"`
	RxHistory                 []*PrescriptionStatus    `json:"erx_history,omitempty"`
	OriginatingTreatmentId    int64                    `json:"-"`
}

// defining an equals method on the treatment so that
// we can compare two treatments based on the fields that
// are important to be the same between treatments
func (t *Treatment) Equals(other *Treatment) bool {

	if t == nil || other == nil {
		return false
	}

	return t.PrescriptionId.Int64() == other.PrescriptionId.Int64() &&
		reflect.DeepEqual(t.DrugDBIds, other.DrugDBIds) &&
		t.DosageStrength == other.DosageStrength &&
		t.DispenseValue == other.DispenseValue &&
		t.DispenseUnitId.Int64() == other.DispenseUnitId.Int64() &&
		t.NumberRefills == other.NumberRefills &&
		t.SubstitutionsAllowed == other.SubstitutionsAllowed &&
		t.DaysSupply == other.DaysSupply &&
		t.PatientInstructions == other.PatientInstructions &&
		t.PharmacyLocalId.Int64() == other.PharmacyLocalId.Int64()
}
