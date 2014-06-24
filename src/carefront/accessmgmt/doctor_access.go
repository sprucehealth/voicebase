package accessmgmt

import (
	"carefront/api"
	"errors"
	"net/http"
)

func ValidateDoctorAccessToPatientFile(doctorId, patientId int64, dataAPI api.DataAPI) (int, error) {
	httpStatusCode := http.StatusOK

	careTeam, err := dataAPI.GetCareTeamForPatient(patientId)
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
	doctorFound := false
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderId == doctorId {
			doctorFound = true
			break
		}
	}

	if !doctorFound {
		httpStatusCode = http.StatusForbidden
		err = errors.New("Doctor is unable to diagnose patient because he/she is not the primary doctor")
		return httpStatusCode, err

	}

	return http.StatusOK, nil
}

func ValidateReadAccessToPatientCase(doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI) (int, error) {
	return 0, nil
}

func ValidateWriteAccessToPatientCase(doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI) (int, error) {
	return 0, nil
}
