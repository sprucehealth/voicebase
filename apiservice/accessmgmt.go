package apiservice

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type RecordAccessRequired int

const (
	ReadAccessRequired RecordAccessRequired = 1 << iota
	WriteAccessRequired
)

func (ra RecordAccessRequired) Has(a RecordAccessRequired) bool {
	return (ra & a) != 0
}

var (
	JBCQError            = newJBCQForbiddenAccessError()
	AccessForbiddenError = NewAccessForbiddenError()
)

// AccountDoctorHasAccessToCase validates a given doctor account's access to a patient case, this will also optionally populate the case and doctorID attributes of the provided context
func AccountDoctorHasAccessToCase(ctx context.Context, accountID, caseID int64, accountRole string, requiredAccess RecordAccessRequired, dataAPI api.DataAPI) (bool, error) {
	doctorID, err := dataAPI.GetDoctorIDFromAccountID(accountID)
	if err != nil {
		return false, err
	}
	return DoctorHasAccessToCase(ctx, doctorID, caseID, accountRole, requiredAccess, dataAPI)
}

// DoctorHasAccessToCase validates a given doctor's access to a patient case, this will also optionally populate the case and doctorID attributes of the provided context
func DoctorHasAccessToCase(ctx context.Context, doctorID, caseID int64, accountRole string, requiredAccess RecordAccessRequired, dataAPI api.DataAPI) (bool, error) {
	requestCache, _ := CtxCache(ctx)
	if requestCache != nil {
		requestCache[CKDoctorID] = doctorID
	}

	patientCase, err := dataAPI.GetPatientCaseFromID(caseID)
	if err != nil {
		return false, err
	}
	if requestCache != nil {
		requestCache[CKPatientCase] = patientCase
	}

	if requiredAccess.Has(WriteAccessRequired) {
		if err := ValidateWriteAccessToPatientCase(httputil.Post, accountRole, doctorID, patientCase.PatientID, patientCase.ID.Int64(), dataAPI); err != nil {
			return false, err
		}
		requiredAccess = requiredAccess ^ WriteAccessRequired
	}
	if requiredAccess.Has(ReadAccessRequired) {
		if err := ValidateReadAccessToPatientCase(httputil.Get, accountRole, doctorID, patientCase.PatientID, patientCase.ID.Int64(), dataAPI); err != nil {
			return false, err
		}
		requiredAccess = requiredAccess ^ ReadAccessRequired
	}
	return requiredAccess == 0, nil
}

func ValidateDoctorAccessToPatientFile(httpMethod, role string, doctorID int64, patientID common.PatientID, dataAPI api.DataAPI) error {
	switch role {
	case api.RoleCC:
		if httpMethod == httputil.Get {
			return nil
		}
		return NewCareCoordinatorAccessForbiddenError()
	case api.RoleDoctor:
	default:
		return AccessForbiddenError
	}

	cases, err := dataAPI.GetCasesForPatient(patientID, common.SubmittedPatientCaseStates())
	if err != nil {
		return err
	}

	caseIDs := make([]int64, len(cases))
	for i, pc := range cases {
		caseIDs[i] = pc.ID.Int64()
	}

	careTeams, err := dataAPI.CaseCareTeams(caseIDs)
	if err != nil {
		return err
	}

	if len(careTeams) == 0 {
		return AccessForbiddenError
	}

	// ensure that the doctor is part of at least one of the patient's care teams
	for _, careTeam := range careTeams {
		for _, assignment := range careTeam.Assignments {
			if assignment.ProviderRole == api.RoleDoctor && assignment.ProviderID == doctorID {
				return nil
			}
		}
	}

	return AccessForbiddenError
}

func ValidateReadAccessToPatientCase(httpMethod, role string, doctorID int64, patientID common.PatientID, patientCaseID int64, dataAPI api.DataAPI) error {
	return ValidateAccessToPatientCase(httputil.Get, role, doctorID, patientID, patientCaseID, dataAPI)
}

func ValidateWriteAccessToPatientCase(httpMethod, role string, doctorID int64, patientID common.PatientID, patientCaseID int64, dataAPI api.DataAPI) error {
	return ValidateAccessToPatientCase(httputil.Post, role, doctorID, patientID, patientCaseID, dataAPI)
}

func ValidateAccessToPatientCase(httpMethod, role string, doctorID int64, patientID common.PatientID, patientCaseID int64, dataAPI api.DataAPI) error {
	switch role {
	case api.RoleCC:
		if httpMethod == httputil.Get {
			return nil
		}
		return NewCareCoordinatorAccessForbiddenError()
	case api.RoleDoctor:
	default:
		return AccessForbiddenError
	}

	switch httpMethod {
	case httputil.Get:
		return validateReadAccessToPatientCase(httpMethod, role, doctorID, patientID, patientCaseID, dataAPI)
	case httputil.Put, httputil.Post, httputil.Delete:
		return validateWriteAccessToPatientCase(httpMethod, role, doctorID, patientID, patientCaseID, dataAPI)
	}

	return fmt.Errorf("Unknown http method %s", httpMethod)
}

// ValidateAccessToPatientCase checks to ensure that the doctor has read access to the patient case. A doctor
// has read access so long as the the doctor is assigned to the patient as one of their doctors, and
// the case is not temporarily claimed by another doctor for exclusive access
func validateReadAccessToPatientCase(httpMethod, role string, doctorID int64, patientID common.PatientID, patientCaseID int64, dataAPI api.DataAPI) error {
	patientCase, err := dataAPI.GetPatientCaseFromID(patientCaseID)
	if err != nil {
		return err
	}

	if !patientCase.Claimed {
		doctorAssignments, err := dataAPI.GetDoctorsAssignedToPatientCase(patientCaseID)
		if err != nil {
			return err
		}

		for _, assignment := range doctorAssignments {
			// if the case is temporarily claimed, only the doctor that has the temporary claim can read the patient case
			if assignment.Status == api.StatusTemp &&
				assignment.ProviderRole == api.RoleDoctor &&
				assignment.ProviderID == doctorID &&
				assignment.Expires != nil &&
				!assignment.Expires.Before(time.Now()) {
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
func validateWriteAccessToPatientCase(httpMethod, role string, doctorID int64, patientID common.PatientID, patientCaseID int64, dataAPI api.DataAPI) error {

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
		case api.StatusActive:
			if assignment.ProviderRole == api.RoleDoctor &&
				assignment.ProviderID == doctorID {
				// doctor has access so long as they have access to both patient file and patient information
				return ValidateDoctorAccessToPatientFile(httpMethod, role, doctorID, patientID, dataAPI)
			}
		case api.StatusTemp:
			if assignment.ProviderRole == api.RoleDoctor &&
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

	if !patientCase.Claimed {
		return JBCQError
	}

	return AccessForbiddenError
}
