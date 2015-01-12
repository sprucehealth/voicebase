package apiservice

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/surescripts"
)

func ValidateTreatment(treatment *common.Treatment) error {
	if treatment.DrugInternalName == "" {
		return errors.New("Drug Internal name for treatment cannot be empty")
	}

	if treatment.DosageStrength == "" {
		return errors.New("Dosage Strength for treatment cannot be empty")
	}

	if treatment.DispenseValue.Float64() == 0 {
		return errors.New("DispenseValue for treatment cannot be 0")
	}

	// only allow a total of 10 digits with 1 decimal or 11 digits without a decimal
	dispenseValueStr := strconv.FormatFloat(treatment.DispenseValue.Float64(), 'f', -1, 64)
	if len(dispenseValueStr) > 11 {
		return errors.New("Dispense value invalid. Can only be 10 digits with a decimal point or 11 digits without decimal")
	}

	if treatment.DispenseUnitID.Int64() == 0 {
		return errors.New("DispenseUnit	 Id for treatment cannot be 0")
	}

	if treatment.PatientInstructions == "" {
		return errors.New("SIG for treatment cannot be empty")
	}

	if len(treatment.PatientInstructions) > surescripts.MaxPatientInstructionsLength {
		return errors.New("SIG should not be greater than 140 characters")
	}

	if treatment.DrugDBIDs == nil || len(treatment.DrugDBIDs) == 0 {
		return errors.New("Drug DB Ids for treatment cannot be empty")
	}

	if treatment.NumberRefills.Int64Value > surescripts.MaxNumberRefillsMaxValue {
		return errors.New(("Number of refills has to be less than 99"))
	}

	if treatment.DaysSupply.IsValid && treatment.DaysSupply.Int64Value == 0 {
		return errors.New("Days Supply cannot be 0")
	}

	if treatment.DaysSupply.Int64Value > surescripts.MaxDaysSupplyMaxValue {
		return errors.New("Days supply cannot be greater than 999")
	}

	if len(treatment.PharmacyNotes) > surescripts.MaxPharmacyNotesLength {
		return errors.New("Pharmacy notes should not be great than 210 characters")
	}

	TrimSpacesFromTreatmentFields(treatment)

	return nil
}

// IsDrugOutOfMarket returns an error if the drug is out of market and nil if not. The drug is searched
// for the in drug database. If it exists in the database, it is considered to be in market, and out of
// market if not.
func IsDrugOutOfMarket(treatment *common.Treatment, doctor *common.Doctor, erxAPI erx.ERxAPI) error {
	// check to ensure that the drug is still in market; we do so by ensuring that we are still able
	// to get back the drug db ids to identify this drug
	medicationToCheck, err := erxAPI.SelectMedication(doctor.DoseSpotClinicianID, treatment.DrugInternalName, treatment.DosageStrength)
	if err != nil {
		return &SpruceError{
			HTTPStatusCode: http.StatusInternalServerError,
			DeveloperError: "Unable to select medication to identify whether or not it is still available in the market: " + err.Error(),
			UserError:      "Unable to check if drug is in/out of market for templated drug. Please try again later",
		}
	}

	// if not, we cannot allow the doctor to prescribe this drug given that its no longer in market (a surescripts requirement)
	if medicationToCheck == nil {
		return &SpruceError{
			HTTPStatusCode: http.StatusBadRequest,
			DeveloperError: "Drug is no longer in market so template cannot be used for adding treatment",
			UserError:      fmt.Sprintf("%s %s is no longer available and cannot be prescribed to the patient. We suggest that you remove this saved template from your list.", treatment.DrugInternalName, treatment.DosageStrength),
		}
	}
	return nil
}

// A complete and valid drug name from dosespot is represented as "DrugName (DrugRoute - DrugForm)"
// This method tries to break up this complete internal drug name into its individual components. It's a best effort
// in that it treats the entire name presented as the drugName if the drugInternalName is of any invalid format
func BreakDrugInternalNameIntoComponents(drugInternalName string) (drugName, drugForm, drugRoute string) {
	indexOfParanthesis := strings.IndexRune(drugInternalName, '(')
	// nothing to do if the name is not in the required format.
	// fail gracefully by returning the drug internal name for the drug name and
	if indexOfParanthesis == -1 {
		drugName = drugInternalName
		return
	}

	// treat the entire name as the drugName if there is no closing paranthesis
	indexOfClosingParanthesis := strings.IndexRune(drugInternalName, ')')
	if indexOfClosingParanthesis == -1 {
		drugName = drugInternalName
		return
	}

	// treat the entire name as the drugName if there is no hyphen
	indexOfHyphen := strings.IndexRune(drugInternalName[indexOfParanthesis:], '-')
	if indexOfHyphen == -1 {
		drugName = drugInternalName
		return
	}

	// located the position of the hyphen within the actual string
	indexOfHyphen = indexOfParanthesis + indexOfHyphen

	// treat the entire name as the drug name if hyphen is found after the closing paranthesis
	if indexOfHyphen > indexOfClosingParanthesis {
		drugName = drugInternalName
		return
	}

	drugName = strings.TrimSpace(drugInternalName[:indexOfParanthesis])
	drugRoute = strings.TrimSpace(drugInternalName[indexOfParanthesis+1 : indexOfHyphen])
	drugForm = strings.TrimSpace(drugInternalName[indexOfHyphen+1 : indexOfClosingParanthesis])
	return
}

func TrimSpacesFromTreatmentFields(treatment *common.Treatment) {
	treatment.PatientInstructions = strings.TrimSpace(treatment.PatientInstructions)
	treatment.PharmacyNotes = strings.TrimSpace(treatment.PharmacyNotes)
}
