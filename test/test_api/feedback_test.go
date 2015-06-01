package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestFeedback(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	accountID, err := testData.AuthAPI.CreateAccount("fdsafdsafdsa@fewffdasfdsajewlkfwe.com", "fdasfda", api.RolePatient)
	test.OK(t, err)

	patient := &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	patientID := patient.ID.Int64()

	found, err := testData.DataAPI.PatientFeedbackRecorded(patientID, "one")
	test.OK(t, err)
	test.Equals(t, false, found)

	cmt := "LOVED IT! EVERYTHING IS GR34t! OnE sTAR!"
	test.OK(t, testData.DataAPI.RecordPatientFeedback(patientID, "one", 1, &cmt))

	found, err = testData.DataAPI.PatientFeedbackRecorded(patientID, "one")
	test.OK(t, err)
	test.Equals(t, true, found)
	found, err = testData.DataAPI.PatientFeedbackRecorded(patientID, "two")
	test.OK(t, err)
	test.Equals(t, false, found)

	test.OK(t, testData.DataAPI.RecordPatientFeedback(patientID, "two", 5, nil))
	found, err = testData.DataAPI.PatientFeedbackRecorded(patientID, "two")
	test.OK(t, err)
	test.Equals(t, true, found)
}
