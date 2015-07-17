package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestParentalConsent(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	accountID, err := testData.AuthAPI.CreateAccount("patient@sprucehealth.com", "12345", api.RolePatient)
	test.OK(t, err)
	patient := &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	patientID := patient.ID.Int64()

	accountID, err = testData.AuthAPI.CreateAccount("parent@sprucehealth.com", "12345", api.RolePatient)
	test.OK(t, err)
	patient = &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	parentPatientID := patient.ID.Int64()

	patient, err = testData.DataAPI.Patient(patientID, true)
	test.OK(t, err)
	test.Equals(t, false, patient.HasParentalConsent)

	consent, err := testData.DataAPI.ParentChildConsent(parentPatientID, patientID)
	test.Assert(t, err != nil, "Expected error for non-existant link between parent and child")
	test.Assert(t, api.IsErrNotFound(err), "Expected IsErrNotFound")
	test.Equals(t, false, consent)

	test.OK(t, testData.DataAPI.LinkParentChild(parentPatientID, patientID, "likely-just-a-friend"))

	patient, err = testData.DataAPI.Patient(patientID, true)
	test.OK(t, err)
	test.Equals(t, false, patient.HasParentalConsent)

	consent, err = testData.DataAPI.ParentChildConsent(parentPatientID, patientID)
	test.OK(t, err)
	test.Equals(t, false, consent)

	test.OK(t, testData.DataAPI.GrantParentChildConsent(parentPatientID, patientID))

	patient, err = testData.DataAPI.Patient(patientID, true)
	test.OK(t, err)
	test.Equals(t, true, patient.HasParentalConsent)

	consent, err = testData.DataAPI.ParentChildConsent(parentPatientID, patientID)
	test.OK(t, err)
	test.Equals(t, true, consent)
}
