package apiservice

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

var (
	JBCQError            = newJBCQForbiddenAccessError()
	AccessForbiddenError = newAccessForbiddenError()
)

func ValidateDoctorAccessToPatientFile(doctorId, patientId int64, dataAPI api.DataAPI) error {

	careTeam, err := dataAPI.GetCareTeamForPatient(patientId)
	if err != nil {
		return err
	}

	if careTeam == nil {
		return AccessForbiddenError
	}

	// ensure that the doctor is part of the patient's care team
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderID == doctorId {
			return nil
		}
	}

	return AccessForbiddenError
}

func ValidateAccessToPatientCase(httpMethod string, doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI) error {

	switch httpMethod {
	case HTTP_GET:
		return validateReadAccessToPatientCase(doctorId, patientId, patientCaseId, dataAPI)
	case HTTP_PUT, HTTP_POST, HTTP_DELETE:
		return validateWriteAccessToPatientCase(doctorId, patientId, patientCaseId, dataAPI)
	default:
		return fmt.Errorf("Unknown http method %s", httpMethod)
	}

	return nil
}

// ValidateAccessToPatientCase(r.Method, checks to ensure that the doctor has read access to the patient case. A doctor
// has read access so long as the the doctor is assigned to the patient as one of their doctors, and
// the case is not temporarily claimed by another doctor for exclusive access
func validateReadAccessToPatientCase(doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI) error {
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
				assignment.ProviderID == doctorId &&
				assignment.Status == api.STATUS_TEMP &&
				assignment.Expires != nil && !assignment.Expires.Before(time.Now()) {
				return nil
			}
		}

		return JBCQError
	}

	// if there is no exclusive access on the patient case, then the doctor can access case for
	// reading so long as doctor can access global patient information
	return ValidateDoctorAccessToPatientFile(doctorId, patientId, dataAPI)
}

// ValidateWriteAccessToPatientCase checks to ensure that the doctor has write access to the patient case. A doctor
// has write access so long as the doctor is explicitly assigned to the case,
// and the access has not expired if the doctor is granted temporary access
func validateWriteAccessToPatientCase(doctorId, patientId, patientCaseId int64, dataAPI api.DataAPI) error {
	doctorAssignments, err := dataAPI.GetDoctorsAssignedToPatientCase(patientCaseId)
	if err != nil {
		return err
	}

	// no assignments to the case, in which case the doctor does not have write access to the patient case
	if len(doctorAssignments) == 0 {
		return AccessForbiddenError
	}

	// check to ensure that the doctor has temporary or complete access to the case
	for _, assignment := range doctorAssignments {
		switch assignment.Status {
		case api.STATUS_ACTIVE:
			if assignment.ProviderRole == api.DOCTOR_ROLE &&
				assignment.ProviderID == doctorId {
				// doctor has access so long as they have access to both patient file and patient information
				return ValidateDoctorAccessToPatientFile(doctorId, patientId, dataAPI)
			}
		case api.STATUS_TEMP:
			if assignment.ProviderRole == api.DOCTOR_ROLE &&
				assignment.ProviderID == doctorId &&
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
		return JBCQError
	}

	return AccessForbiddenError
}
