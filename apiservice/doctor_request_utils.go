package apiservice

import (
	"errors"
	"fmt"
	"github.com/sprucehealth/backend/api"
)

var (
	NoPatientVisitFound = errors.New("No patient visit found when trying to validate that the doctor is authorized to work on this patient visit")
)

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
