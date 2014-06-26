package apiservice

import (
	"carefront/api"
	"carefront/common"
	"time"
)

func ValidateDoctorAccessToPatientFile(doctorId, patientId int64, dataAPI api.DataAPI) error {

	careTeam, err := dataAPI.GetCareTeamForPatient(patientId)
	if err != nil {
		return err
	}

	if careTeam == nil {
		return NewAccessForbiddenError()
	}

	// ensure that the doctor is part of the patient's care team
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderId == doctorId {
			return nil
		}
	}

	return NewAccessForbiddenError()
}

// ValidateReadAccessToPatientCase checks to ensure that the doctor has read access to the patient case. A doctor
// has read access so long as the the doctor is assigned to the patient as one of their doctors, and
// the case is not temporarily claimed by another doctor for exclusive access
func ValidateReadAccessToPatientCase(doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI) error {
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
				assignment.Status == api.STATUS_TEMP &&
				assignment.Expires != nil && !assignment.Expires.Before(time.Now()) {
				return nil
			}
		}

		return NewJBCQForbiddenAccessError()
	}

	// if there is no exclusive access on the patient case, then the doctor can access case for
	// reading so long as doctor can access global patient information
	return ValidateDoctorAccessToPatientFile(doctorId, patientId, dataAPI)
}

// ValidateWriteAccessToPatientCase checks to ensure that the doctor has write access to the patient case. A doctor
// has write access so long as the doctor is explicitly assigned to the case,
// and the access has not expired if the doctor is granted temporary access
func ValidateWriteAccessToPatientCase(doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI) error {
	doctorAssignments, err := dataAPI.GetDoctorsAssignedToPatientCase(patientCaseId)
	if err != nil {
		return err
	}

	// no assignments to the case, in which case the doctor does not have write access to the patient case
	if len(doctorAssignments) == 0 {
		return NewAccessForbiddenError()
	}

	// check to ensure that the doctor has temporary or complete access to the case
	for _, assignment := range doctorAssignments {
		switch assignment.Status {
		case api.STATUS_ACTIVE:
			if assignment.ProviderRole == api.DOCTOR_ROLE &&
				assignment.ProviderId == doctorId {
				// doctor has access so long as they have access to both patient file and patient information
				return ValidateDoctorAccessToPatientFile(doctorId, patientId, dataAPI)
			}
		case api.STATUS_TEMP:
			if assignment.ProviderRole == api.DOCTOR_ROLE &&
				assignment.ProviderId == doctorId &&
				assignment.Expires != nil && !assignment.Expires.Before(time.Now()) {
				// doctor has access so long as they have access to both patient file and patient information
				return ValidateDoctorAccessToPatientFile(doctorId, patientId, dataAPI)
			}
		}
	}

	// if at this point the doctor does not have access to the case,
	// then this means the doctor cannot write to the patient case
	patientCase, err := dataAPI.GetPatientCaseFromId(patientCaseId)
	if err != nil {
		return err
	}

	switch patientCase.Status {
	case common.PCStatusUnclaimed, common.PCStatusTempClaimed:
		return NewJBCQForbiddenAccessError()
	}

	return NewAccessForbiddenError()
}
