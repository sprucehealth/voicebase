package apiservice

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

var (
	JBCQError            = newJBCQForbiddenAccessError()
	AccessForbiddenError = NewAccessForbiddenError()
)

func ValidateDoctorAccessToPatientFile(httpMethod, role string, doctorID, patientID int64, dataAPI api.DataAPI) error {

	switch role {
	case api.MA_ROLE:
		if httpMethod == HTTP_GET {
			return nil
		}
		return NewCareCoordinatorAccessForbiddenError()
	case api.DOCTOR_ROLE:
	default:
		return AccessForbiddenError
	}

	careTeam, err := dataAPI.GetCareTeamForPatient(patientID)
	if err != nil {
		return err
	}

	// This case is essetially impossible as the only case where we would return an empty care team would
	//		be if err was non nil. But leaving this in here for defensive purposes.
	if careTeam == nil {
		return AccessForbiddenError
	}

	// ensure that the doctor is part of at least one of the patient's care teams
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderID == doctorID {
			return nil
		}
	}

	return AccessForbiddenError
}

func ValidateAccessToPatientCase(httpMethod, role string, doctorID, patientID, patientCaseID int64, dataAPI api.DataAPI) error {
	switch role {
	case api.MA_ROLE:
		if httpMethod == HTTP_GET {
			return nil
		}
		return NewCareCoordinatorAccessForbiddenError()
	case api.DOCTOR_ROLE:
	default:
		return AccessForbiddenError
	}

	switch httpMethod {
	case HTTP_GET:
		return validateReadAccessToPatientCase(httpMethod, role, doctorID, patientID, patientCaseID, dataAPI)
	case HTTP_PUT, HTTP_POST, HTTP_DELETE:
		return validateWriteAccessToPatientCase(httpMethod, role, doctorID, patientID, patientCaseID, dataAPI)
	}

	return fmt.Errorf("Unknown http method %s", httpMethod)
}

// ValidateAccessToPatientCase checks to ensure that the doctor has read access to the patient case. A doctor
// has read access so long as the the doctor is assigned to the patient as one of their doctors, and
// the case is not temporarily claimed by another doctor for exclusive access
func validateReadAccessToPatientCase(httpMethod, role string, doctorID, patientID, patientCaseID int64, dataAPI api.DataAPI) error {
	patientCase, err := dataAPI.GetPatientCaseFromID(patientCaseID)
	if err != nil {
		return err
	}

	// if the patient case is temporarily claimed, ensure that the current doctor
	// has exclusive access to the case
	if patientCase.Status == common.PCStatusTempClaimed {
		doctorAssignments, err := dataAPI.GetDoctorsAssignedToPatientCase(patientCaseID)
		if err != nil {
			return err
		}

		for _, assignment := range doctorAssignments {
			if assignment.ProviderRole == api.DOCTOR_ROLE &&
				assignment.ProviderID == doctorID &&
				assignment.Status == api.STATUS_TEMP &&
				assignment.Expires != nil && !assignment.Expires.Before(time.Now()) {
				return nil
			}
		}

		return JBCQError
	}

	// if there is no exclusive access on the patient case, then the doctor can access case for
	// reading so long as doctor can access global patient information
	return ValidateDoctorAccessToPatientFile(httpMethod, role, doctorID, patientID, dataAPI)
}

// ValidateWriteAccessToPatientCase checks to ensure that the doctor has write access to the patient case. A doctor
// has write access so long as the doctor is explicitly assigned to the case,
// and the access has not expired if the doctor is granted temporary access
func validateWriteAccessToPatientCase(httpMethod, role string, doctorID, patientID, patientCaseID int64, dataAPI api.DataAPI) error {
	doctorAssignments, err := dataAPI.GetDoctorsAssignedToPatientCase(patientCaseID)
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
				assignment.ProviderID == doctorID {
				// doctor has access so long as they have access to both patient file and patient information
				return ValidateDoctorAccessToPatientFile(httpMethod, role, doctorID, patientID, dataAPI)
			}
		case api.STATUS_TEMP:
			if assignment.ProviderRole == api.DOCTOR_ROLE &&
				assignment.ProviderID == doctorID &&
				assignment.Expires != nil && !assignment.Expires.Before(time.Now()) {
				// doctor has access so long as they have access to both patient file and patient information
				return ValidateDoctorAccessToPatientFile(httpMethod, role, doctorID, patientID, dataAPI)
			}
		}
	}

	// if at this point the doctor does not have access to the case,
	// then this means the doctor cannot write to the patient case
	patientCase, err := dataAPI.GetPatientCaseFromID(patientCaseID)
	if err != nil {
		return err
	}

	switch patientCase.Status {
	case common.PCStatusUnclaimed, common.PCStatusTempClaimed:
		return JBCQError
	}

	return AccessForbiddenError
}
