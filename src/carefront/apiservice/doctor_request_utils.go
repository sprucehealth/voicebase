package apiservice

import (
	"carefront/api"
	"carefront/common"
	"database/sql"
	"errors"
	"net/http"
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
