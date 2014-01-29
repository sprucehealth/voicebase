package apiservice

import (
	"carefront/api"
	"carefront/common"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	NoPatientVisitFound = errors.New("No patient visit found when trying to validate that the doctor is authorized to work on this patient visit")
)

func ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, accountIdForDoctor int64, DataApi api.DataAPI) (doctorId int64, patientVisit *common.PatientVisit, careTeam *common.PatientCareProviderGroup, httpStatusCode int, err error) {
	httpStatusCode = http.StatusOK
	doctorId, err = DataApi.GetDoctorIdFromAccountId(accountIdForDoctor)
	if err != nil {
		httpStatusCode = http.StatusInternalServerError
		err = errors.New("Unable to get doctor id from account id " + err.Error())
		return
	}

	patientVisit, err = DataApi.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		if err == sql.ErrNoRows {
			httpStatusCode = http.StatusBadRequest
			err = NoPatientVisitFound
			return
		}
		httpStatusCode = http.StatusInternalServerError
		err = errors.New("Unable to get patient visit from id : " + err.Error())
		return
	}

	careTeam, err = DataApi.GetCareTeamForPatient(patientVisit.PatientId)
	if err != nil {
		httpStatusCode = http.StatusInternalServerError
		err = errors.New("Unable to get care team for patient visit id " + err.Error())
		return
	}

	if careTeam == nil {
		httpStatusCode = http.StatusForbidden
		err = errors.New("No care team assigned to patient visit so cannot diagnose patient visit")
		return
	}

	// ensure that the doctor is the current primary doctor for this patient
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderId != doctorId {
			httpStatusCode = http.StatusForbidden
			err = errors.New("Doctor is unable to diagnose patient because he/she is not the primary doctor")
			return
		}
	}
	return
}

func EnsurePatientVisitInExpectedStatus(DataApi api.DataAPI, patientVisitId int64, expectedState string) error {
	// you can only add treatments if the patient visit is in the REVIEWING state
	patientVisit, err := DataApi.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		return errors.New("Unable to get patient visit from id: " + err.Error())
	}

	if patientVisit.Status != expectedState {
		return fmt.Errorf("Unable to take intended action on the patient visit since it is not in the %s state. Current status: %s", expectedState, patientVisit.Status)
	}
	return nil
}

func breakDrugInternalNameIntoComponents(drugInternalName string) (drugName, drugForm, drugRoute string) {
	indexOfParanthesis := strings.Index(drugInternalName, "(")
	// nothing to do if the name is not in the required format.
	// fail gracefully by returning the drug internal name for the drug name and
	if indexOfParanthesis == -1 {
		drugName = drugInternalName
		return
	}

	indexOfClosingParanthesis := strings.Index(drugInternalName, ")")
	indexOfHyphen := indexOfParanthesis + strings.Index(drugInternalName[indexOfParanthesis:], "-")
	if indexOfHyphen == -1 || indexOfHyphen < indexOfParanthesis || indexOfHyphen > indexOfClosingParanthesis {
		drugName = drugInternalName
		return
	}

	drugName = strings.TrimSpace(drugInternalName[:indexOfParanthesis])
	drugRoute = strings.TrimSpace(drugInternalName[indexOfParanthesis+1 : indexOfHyphen])
	drugForm = strings.TrimSpace(drugInternalName[indexOfHyphen+1 : indexOfClosingParanthesis])
	return
}
