package apiservice

import (
	"carefront/api"
	"carefront/common"
	"database/sql"
	"errors"
	"fmt"
)

var (
	NoPatientVisitFound = errors.New("No patient visit found when trying to validate that the doctor is authorized to work on this patient visit")
)

type doctorPatientVisitReviewData struct {
	DoctorId     int64
	PatientVisit *common.PatientVisit
	CareTeam     *common.PatientCareProviderGroup
}

func ValidateDoctorHasAccessToPatient(doctorID, patientID int64, dataAPI api.DataAPI) (*common.PatientCareProviderGroup, error) {
	careTeam, err := dataAPI.GetCareTeamForPatient(patientID)
	if err != nil {
		return nil, errors.New("Unable to get care team for patient " + err.Error())
	}

	if careTeam == nil {
		return nil, NotAuthorizedError("No care team assigned to patient")
	}

	// ensure that the doctor is the current primary doctor for this patient
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderId != doctorID {
			return nil, NotAuthorizedError("Doctor is unable to diagnose patient because he/she is not the primary doctor")
		}
	}

	return careTeam, nil
}

func ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, accountIdForDoctor int64, dataAPI api.DataAPI) (*doctorPatientVisitReviewData, error) {
	doctorId, err := dataAPI.GetDoctorIdFromAccountId(accountIdForDoctor)
	if err != nil {
		return nil, errors.New("Unable to get doctor id from account id " + err.Error())
	}

	patientVisit, err := dataAPI.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NoPatientVisitFound
		}
		return nil, errors.New("Unable to get patient visit from id : " + err.Error())
	}

	careTeam, err := ValidateDoctorHasAccessToPatient(doctorId, patientVisit.PatientId.Int64(), dataAPI)
	if err != nil {
		return nil, err
	}

	reviewData := &doctorPatientVisitReviewData{
		DoctorId:     doctorId,
		PatientVisit: patientVisit,
		CareTeam:     careTeam,
	}
	return reviewData, nil
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
