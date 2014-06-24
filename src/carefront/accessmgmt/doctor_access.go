package accessmgmt

import (
	"carefront/api"
	"carefront/common"
	"net/http"
)

func ValidateDoctorAccessToPatientFile(doctorId, patientId int64, dataAPI api.DataAPI, r *http.Request) error {

	careTeam, err := dataAPI.GetCareTeamForPatient(patientId)
	if err != nil {
		return err
	}

	if careTeam == nil {
		return errors.NewAccessForbiddenError(r)
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
		return errors.NewAccessForbiddenError(r)
	}

	return nil
}

// apiservice.ValidateReadAccessToPatientCase checks to ensure that the doctor has read access to the patient case. A doctor
// has read access so long as the case is not temporarily claimed by another doctor for exclusive access
func ValidateReadAccessToPatientCase(doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI, r *http.Request) error {
	patientCase, err := dataAPI.GetPatientCaseFromId(patientCaseId)
	if err != nil {
		return err
	}

	// if the patient case is temporarily claimed, ensure that the current doctor
	// has exclusive access to the case
	if patientCase.Status == common.PCStatusTempClaimed {
		doctorAssignments, err := dataAPI.GetDoctorsAssignedToPatientCase(patientCaseId)
		if err != nil {
			return err
		}

		for _, assignment := range doctorAssignments {
			if assignment.ProviderRole == api.DOCTOR_ROLE &&
				assignment.ProviderId == doctorId &&
				assignment.Status == api.STATUS_TEMP {
				return nil
			}
		}

		return errors.NewJBCQForbiddenAccessError(r)
	}

	// if there is no exclusive access on the patient case, then the doctor can access case for
	// reading so long as doctor can patient information
	return ValidateDoctorAccessToPatientFile(doctorId, patientId, dataAPI)
}

// ValidateWriteAccessToPatientCase checks to ensure that the doctor has write access to the patient case. A doctor
// has write access so long as the doctor is assigned to the case
func ValidateWriteAccessToPatientCase(doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI, r *http.Request) error {
	doctorAssignments, err := dataAPI.GetDoctorsAssignedToPatientCase(patientCaseId)
	if err != nil {
		return err
	}

	// check to ensure that the doctor has temporary or complete access to the case
	for _, assignment := range doctorAssignments {
		switch assignment.Status {
		case api.STATUS_ACTIVE, api.STATUS_TEMP:
			if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderId == doctorId {
				return nil
			}
		case api.STATUS_TEMP_INACTIVE:
			if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderId == doctorId {
				return errors.NewJBCQForbiddenAccessError(r)
			}
		}
	}

	// if at this point the doctor does not have access to the case, then this means the doctor cannot write to the patient case
	patientCase, err := dataAPI.GetPatientCaseFromId(patientCaseId)
	if err != nil {
		return err
	} else if patientCase.Status == common.PCStatusTempClaimed {
		return errors.NewJBCQForbiddenAccessError(r)
	}

	return errors.NewAccessForbiddenError(r)
}
