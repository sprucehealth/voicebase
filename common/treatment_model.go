package common

import (
	"fmt"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/pharmacy"
)

type TreatmentStatus string

var (
	TStatusCreated  TreatmentStatus = "CREATED"
	TStatusInactive TreatmentStatus = "INACTIVE"
	TStatusSent     TreatmentStatus = "SENT"
	TStatusDeleted  TreatmentStatus = "DELETED"
)

func GetTreatmentStatus(t string) (TreatmentStatus, error) {
	switch ts := TreatmentStatus(t); ts {
	case TStatusCreated, TStatusInactive, TStatusSent, TStatusDeleted:
		return ts, nil
	}
	return TreatmentStatus(""), fmt.Errorf("Unknown treatment status: %s", t)
}

func (t TreatmentStatus) String() string {
	return string(t)
}

func (t *TreatmentStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = GetTreatmentStatus(ts)
	case []byte:
		*t, err = GetTreatmentStatus(string(ts))
	}
	return err
}

type Treatment struct {
	ID                        encoding.ObjectID             `json:"treatment_id,omitempty"`
	DoctorTreatmentTemplateID encoding.ObjectID             `json:"dr_treatment_template_id,omitempty"`
	StatusDetails             string                        `json:"erx_status_details,omitempty"`
	TreatmentPlanID           encoding.ObjectID             `json:"treatment_plan_id,omitempty"`
	DrugDBIDs                 map[string]string             `json:"drug_db_ids,omitempty"`
	DrugInternalName          string                        `json:"drug_internal_name,omitempty"`
	DrugName                  string                        `json:"drug_name"`
	DrugRoute                 string                        `json:"drug_route,omitempty"`
	DrugForm                  string                        `json:"drug_form,omitempty"`
	GenericDrugName           string                        `json:"generic_drug_name,omitempty"`
	DosageStrength            string                        `json:"dosage_strength,omitempty"`
	DispenseValue             encoding.HighPrecisionFloat64 `json:"dispense_value"`
	DispenseUnitID            encoding.ObjectID             `json:"dispense_unit_id,omitempty"`
	DispenseUnitDescription   string                        `json:"dispense_unit_description,omitempty"`
	NumberRefills             encoding.NullInt64            `json:"refills,omitempty"`
	SubstitutionsAllowed      bool                          `json:"substitutions_allowed"`
	DaysSupply                encoding.NullInt64            `json:"days_supply"`
	PharmacyNotes             string                        `json:"pharmacy_notes,omitempty"`
	PatientInstructions       string                        `json:"patient_instructions,omitempty"`
	CreationDate              *time.Time                    `json:"creation_date,omitempty"`
	Status                    TreatmentStatus               `json:"-"`
	OTC                       bool                          `json:"otc,omitempty"`
	IsControlledSubstance     bool                          `json:"-"`
	SupplementalInstructions  []*DoctorInstructionItem      `json:"supplemental_instructions,omitempty"`
	Doctor                    *Doctor                       `json:"doctor,omitempty"`
	PatientID                 encoding.ObjectID             `json:"patient_id,omitempty"`
	Patient                   *Patient                      `json:"patient,omitempty"`
	DoctorID                  encoding.ObjectID             `json:"doctor_id,omitempty"`
	OriginatingTreatmentID    int64                         `json:"-"`
	ERx                       *ERxData                      `json:"erx,omitempty"`
	HasRxGuide                bool                          `json:"has_rx_guide,omitempty"`
}

type ERxData struct {
	DoseSpotClinicianID   int64                  `json:"-"`
	RxHistory             []StatusEvent          `json:"history,omitempty"`
	Pharmacy              *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
	ErxSentDate           *time.Time             `json:"sent_date,omitempty"`
	ErxLastDateFilled     *time.Time             `json:"last_filled_date,omitempty"`
	ErxReferenceNumber    string                 `json:"-"`
	TransmissionErrorDate *time.Time             `json:"error_date,omitempty"`
	ErxPharmacyID         int64                  `json:"-"`
	ErxMedicationID       encoding.ObjectID      `json:"-"`
	PrescriptionID        encoding.ObjectID      `json:"-"`
	PrescriptionStatus    string                 `json:"status,omitempty"`
	PharmacyLocalID       encoding.ObjectID      `json:"-"`
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

	// only check erx related data if treatment erx is non-empty
	if t.ERx != nil && other.ERx != nil {
		if !(t.ERx.PrescriptionID.Int64() == other.ERx.PrescriptionID.Int64() &&
			t.ERx.PharmacyLocalID.Int64() == other.ERx.PharmacyLocalID.Int64()) {
			return false
		}
	}

	return reflect.DeepEqual(t.DrugDBIDs, other.DrugDBIDs) &&
		t.DosageStrength == other.DosageStrength &&
		t.DispenseValue == other.DispenseValue &&
		t.DispenseUnitID.Int64() == other.DispenseUnitID.Int64() &&
		t.NumberRefills == other.NumberRefills &&
		t.SubstitutionsAllowed == other.SubstitutionsAllowed &&
		t.DaysSupply == other.DaysSupply &&
		t.PatientInstructions == other.PatientInstructions &&
		t.PharmacyNotes == other.PharmacyNotes
}
