package messages

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"errors"
	"net/http"
)

func validateAccess(dataAPI api.DataAPI, r *http.Request, patientCase *common.PatientCase) (personID, doctorID int64, err error) {
	ctx := apiservice.GetContext(r)
	switch ctx.Role {
	case api.DOCTOR_ROLE:
		doctorID, err = dataAPI.GetDoctorIdFromAccountId(ctx.AccountId)
		if err != nil {
			return 0, 0, err
		}
		if _, err := apiservice.ValidateDoctorHasAccessToPatient(doctorID, patientCase.PatientId, dataAPI); err != nil {
			return 0, 0, err
		}
		personID, err = dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorID)
		if err != nil {
			return 0, 0, err
		}
	case api.PATIENT_ROLE:
		patientID, err := dataAPI.GetPatientIdFromAccountId(ctx.AccountId)
		if err != nil {
			return 0, 0, err
		}
		if patientCase.PatientId != patientID {
			return 0, 0, apiservice.NewValidationError("Not authorized", r)
		}
		personID, err = dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientID)
		if err != nil {
			return 0, 0, err
		}
	default:
		return 0, 0, errors.New("Unknown role " + ctx.Role)
	}

	return personID, doctorID, nil
}
