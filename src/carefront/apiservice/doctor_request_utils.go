package apiservice

import (
	"carefront/api"
	"errors"
	"fmt"
	"net/http"
)

var (
	NoPatientVisitFound = errors.New("No patient visit found when trying to validate that the doctor is authorized to work on this patient visit")
)

func ValidateDoctorAccessToPatientFile(doctorId, patientId int64, DataApi api.DataAPI) (int, error) {
	httpStatusCode := http.StatusOK

	careTeam, err := DataApi.GetCareTeamForPatient(patientId)
	if err != nil {
		httpStatusCode = http.StatusInternalServerError
		err = errors.New("Unable to get care team for patient visit id " + err.Error())
		return httpStatusCode, err
	}

	if careTeam == nil {
		httpStatusCode = http.StatusForbidden
		err = errors.New("No care team assigned to patient visit so cannot diagnose patient visit")
		return httpStatusCode, err
	}

	// ensure that the doctor is part of the patient's care team
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderId != doctorId {
			httpStatusCode = http.StatusForbidden
			err = errors.New("Doctor is unable to diagnose patient because he/she is not the primary doctor")
			return httpStatusCode, err
		}
	}

	return http.StatusOK, nil
}

func EnsurePatientVisitInExpectedStatus(dataAPI api.DataAPI, patientVisitId int64, expectedState string) error {
	// you can only add treatments if the patient visit is in the REVIEWING state
	patientVisit, err := dataAPI.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		return errors.New("Unable to get patient visit from id: " + err.Error())
	}

	if patientVisit.Status != expectedState {
		return fmt.Errorf("Unable to take intended action on the patient visit since it is not in the %s state. Current status: %s", expectedState, patientVisit.Status)
	}
	return nil
}
