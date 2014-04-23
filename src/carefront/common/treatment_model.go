package common

import (
	"carefront/encoding"
	"carefront/libs/pharmacy"
	"reflect"
	"time"
)

type Treatment struct {
	Id                        encoding.ObjectId             `json:"treatment_id,omitempty"`
	DoctorTreatmentTemplateId encoding.ObjectId             `json:"dr_treatment_template_id,omitempty"`
	StatusDetails             string                        `json:"erx_status_details,omitempty"`
	TreatmentPlanId           encoding.ObjectId             `json:"treatment_plan_id,omitempty"`
	PatientVisitId            encoding.ObjectId             `json:"patient_visit_id,omitempty"`
	DrugDBIds                 map[string]string             `json:"drug_db_ids,omitempty"`
	DrugInternalName          string                        `json:"drug_internal_name,omitempty"`
	DrugName                  string                        `json:"drug_name"`
	DrugRoute                 string                        `json:"drug_route,omitempty"`
	DrugForm                  string                        `json:"drug_form,omitempty"`
	DosageStrength            string                        `json:"dosage_strength,omitempty"`
	DispenseValue             encoding.HighPrecisionFloat64 `json:"dispense_value"`
	DispenseUnitId            encoding.ObjectId             `json:"dispense_unit_id,omitempty"`
	DispenseUnitDescription   string                        `json:"dispense_unit_description,omitempty"`
	NumberRefills             encoding.NullInt64            `json:"refills,omitempty"`
	SubstitutionsAllowed      bool                          `json:"substitutions_allowed"`
	DaysSupply                encoding.NullInt64            `json:"days_supply"`
	PharmacyNotes             string                        `json:"pharmacy_notes,omitempty"`
	PatientInstructions       string                        `json:"patient_instructions,omitempty"`
	CreationDate              *time.Time                    `json:"creation_date,omitempty"`
	Status                    string                        `json:"-"`
	OTC                       bool                          `json:"otc,omitempty"`
	IsControlledSubstance     bool                          `json:"-"`
	SupplementalInstructions  []*DoctorInstructionItem      `json:"supplemental_instructions,omitempty"`
	Doctor                    *Doctor                       `json:"doctor,omitempty"`
	PatientId                 encoding.ObjectId             `json:"patient_id,omitempty"`
	Patient                   *Patient                      `json:"patient,omitempty"`
	DoctorId                  encoding.ObjectId             `json:"doctor_id,omitempty"`
	OriginatingTreatmentId    int64                         `json:"-"`
	ERx                       *ERxData                      `json:"erx,omitempty"`
}

type ERxData struct {
	DoseSpotClinicianId   int64                  `json:"-"`
	RxHistory             []StatusEvent          `json:"history,omitempty"`
	Pharmacy              *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
	ErxSentDate           *time.Time             `json:"sent_date,omitempty"`
	ErxLastDateFilled     *time.Time             `json:"last_filled_date,omitempty"`
	ErxReferenceNumber    string                 `json:"-"`
	TransmissionErrorDate *time.Time             `json:"error_date,omitempty"`
	ErxPharmacyId         int64                  `json:"-"`
	ErxMedicationId       encoding.ObjectId      `json:"-"`
	PrescriptionId        encoding.ObjectId      `json:"-"`
	PrescriptionStatus    string                 `json:"status,omitempty"`
	PharmacyLocalId       encoding.ObjectId      `json:"-"`
}

// defining an equals method on the treatment so that
// we can compare two treatments based on the fields that
// are important to be the same between treatments
func (t *Treatment) Equals(other *Treatment) bool {

	if t == nil || other == nil {
		return false
	}

	if t.ERx == nil && other.ERx != nil {
		return false
	}

	if t.ERx != nil && other.ERx == nil {
		return false
	}

	return t.ERx.PrescriptionId.Int64() == other.ERx.PrescriptionId.Int64() &&
		reflect.DeepEqual(t.DrugDBIds, other.DrugDBIds) &&
		t.DosageStrength == other.DosageStrength &&
		t.DispenseValue == other.DispenseValue &&
		t.DispenseUnitId.Int64() == other.DispenseUnitId.Int64() &&
		t.NumberRefills == other.NumberRefills &&
		t.SubstitutionsAllowed == other.SubstitutionsAllowed &&
		t.DaysSupply == other.DaysSupply &&
		t.PatientInstructions == other.PatientInstructions &&
		t.ERx.PharmacyLocalId.Int64() == other.ERx.PharmacyLocalId.Int64()
}
