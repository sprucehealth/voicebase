package apiservice

import (
	"carefront/common"
	"carefront/libs/erx"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func validateTreatment(treatment *common.Treatment) error {
	if treatment.DrugInternalName == "" {
		return errors.New("Drug Internal name for treatment cannot be empty")
	}

	if treatment.DosageStrength == "" {
		return errors.New("Dosage Strength for treatment cannot be empty")
	}

	if treatment.DispenseValue == 0 {
		return errors.New("DispenseValue for treatment cannot be 0")
	}

	if treatment.DispenseUnitId.Int64() == 0 {
		return errors.New("DispenseUnit	 Id for treatment cannot be 0")
	}

	if treatment.NumberRefills == 0 {
		return errors.New("Number of refills for treatment cannot be 0")
	}

	if treatment.PatientInstructions == "" {
		return errors.New("Patient Instructions for treatment cannot be empty")
	}

	if treatment.DrugDBIds == nil || len(treatment.DrugDBIds) == 0 {
		return errors.New("Drug DB Ids for treatment cannot be empty")
	}

	trimSpacesFromTreatmentFields(treatment)

	return nil
}

func checkIfDrugInTreatmentFromTemplateIsOutOfMarket(treatment *common.Treatment, doctor *common.Doctor, erxApi erx.ERxAPI) (int, *ErrorResponse) {
	if treatment.DoctorTreatmentTemplateId.Int64() != 0 {
		// check to ensure that the drug is still in market; we do so by ensuring that we are still able
		// to get back the drug db ids to identify this drug
		medicationToCheck, err := erxApi.SelectMedication(doctor.DoseSpotClinicianId, treatment.DrugInternalName, treatment.DosageStrength)
		if err != nil {
			return http.StatusInternalServerError, &ErrorResponse{
				DeveloperError: "Unable to select medication to identify whether or not it is still available in the market: " + err.Error(),
				UserError:      "Unable to check if drug is in/out of market for templated drug. Please try again later",
			}
		}

		// if not, we cannot allow the doctor to prescribe this drug given that its no longer in market (a surescripts requirement)
		if medicationToCheck == nil {
			return http.StatusBadRequest, &ErrorResponse{
				DeveloperError: "Drug is no longer in market so template cannot be used for adding treatment",
				UserError:      fmt.Sprintf("%s %s is no longer available and cannot be prescribed to the patient. We suggest that you remove this saved template from your list.", treatment.DrugInternalName, treatment.DosageStrength),
			}
		}
	}
	return http.StatusOK, nil
}

// A complete and valid drug name from dosespot is represented as "DrugName (DrugRoute - DrugForm)"
// This method tries to break up this complete internal drug name into its individual components. It's a best effort
// in that it treats the entire name presented as the drugName if the drugInternalName is of any invalid format
func breakDrugInternalNameIntoComponents(drugInternalName string) (drugName, drugForm, drugRoute string) {
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

func trimSpacesFromTreatmentFields(treatment *common.Treatment) {
	treatment.PatientInstructions = strings.TrimSpace(treatment.PatientInstructions)
	treatment.PharmacyNotes = strings.TrimSpace(treatment.PharmacyNotes)
}
